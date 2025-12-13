package init_ubuntu

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strconv"
	"strings"

	"wslbridge/internal/cli"
	"wslbridge/internal/config"
	appruntime "wslbridge/internal/runtime"
)

type Command struct{}

func (Command) Name() string { return "init-ubuntu" }
func (Command) Help() string {
	return "Install deps, configure WSL DNS/hosts, setup tun2socks routing (Ubuntu/WSL)"
}

type flags struct {
	skipDeps          bool
	force             bool
	socksPortOverride int
}

func parseFlags(args []string) (flags, error) {
	var f flags
	for _, a := range args {
		switch {
		case a == "--skip-deps":
			f.skipDeps = true
		case a == "--force":
			f.force = true
		case strings.HasPrefix(a, "--socks-port="):
			v := strings.TrimPrefix(a, "--socks-port=")
			n, err := strconv.Atoi(v)
			if err != nil || n < 1 || n > 65535 {
				return flags{}, fmt.Errorf("invalid --socks-port=%q (must be 1..65535)", v)
			}
			f.socksPortOverride = n
		default:
			return flags{}, fmt.Errorf("unknown arg: %s", a)
		}
	}
	return f, nil
}

func (Command) Run(rt appruntime.Runtime, args []string) error {
	if goruntime.GOOS != "linux" {
		return fmt.Errorf("init-ubuntu supports only linux")
	}

	f, err := parseFlags(args)
	if err != nil {
		return err
	}

	// ensure dirs
	if err := os.MkdirAll(filepath.Dir(rt.Paths.ConfigPath), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(rt.Paths.ShareDir, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(rt.Paths.StateDir, 0o755); err != nil {
		return err
	}

	// deps
	if !f.skipDeps {
		if err := rt.Platform.EnsureDeps(rt.Runner); err != nil {
			return err
		}
	}

	// load config (optional)
	var cfg config.Config
	hasCfg := false
	if c, err := config.Load(rt.Paths.ConfigPath); err == nil {
		cfg = c
		hasCfg = true
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	// defaults
	const (
		defaultSocksPort = 1080
		defaultTunDev    = "tun0"
		defaultTunCIDR   = "10.0.0.2/24"
	)

	// effective tun params (respect config)
	if cfg.Tun.Dev == "" {
		cfg.Tun.Dev = defaultTunDev
	}
	if cfg.Tun.CIDR == "" {
		cfg.Tun.CIDR = defaultTunCIDR
	}

	// early check route + running
	defaultRouteLine, err := getDefaultRouteLine(rt)
	if err != nil {
		return err
	}
	if defaultIsTun(defaultRouteLine, cfg.Tun.Dev) && tun2socksIsRunning(rt.Paths.Tun2SocksPIDFile) {
		fmt.Println("already enabled: default route is on", cfg.Tun.Dev, "and tun2socks is running")
		return nil
	}

	// Need sudo for WSL config + routes anyway.
	if err := rt.Runner.Run("sudo", "-v"); err != nil {
		return fmt.Errorf("sudo auth failed: %w", err)
	}

	// WSL-specific: disable generateHosts/generateResolvConf and set resolv.conf nameserver
	if isWSL() {
		pr := cli.NewPrompter(os.Stdin, os.Stdout)

		curDNS := ""
		if hasCfg && cfg.DNS.Nameserver != "" {
			curDNS = cfg.DNS.Nameserver
		}

		// default пустой — чтобы не “угадать” чужую корп-сетку
		dns, err := pr.AskString("DNS nameserver (WSL)", "", curDNS, cli.ValidateIP)
		if err != nil {
			return err
		}
		cfg.DNS.Nameserver = dns

		if err := configureWSLConf(rt); err != nil {
			return err
		}
		if err := writeResolvConf(rt, cfg.DNS.Nameserver); err != nil {
			return err
		}
		fmt.Println("WSL DNS configured. NOTE: wsl.conf changes may require WSL restart.")
	}

	// Save current default route line to state (best-effort)
	if defaultRouteLine != "" {
		_ = os.WriteFile(rt.Paths.DefaultRouteFile, []byte(defaultRouteLine+"\n"), 0o644)
	}

	// Detect SOCKS gateway
	gw, err := detectSocksGateway(rt, cfg, defaultRouteLine)
	if err != nil {
		return err
	}
	cfg.Socks.Host = gw
	fmt.Println("SOCKS gateway:", cfg.Socks.Host)

	// SOCKS port
	socksPort := defaultSocksPort
	if hasCfg && cfg.Socks.Port != 0 {
		socksPort = cfg.Socks.Port
	}
	if f.socksPortOverride != 0 {
		socksPort = f.socksPortOverride
	} else if f.force {
		pr := cli.NewPrompter(os.Stdin, os.Stdout)
		cur := ""
		if hasCfg && cfg.Socks.Port != 0 {
			cur = strconv.Itoa(cfg.Socks.Port)
		}
		portStr, perr := pr.AskString("SOCKS port", strconv.Itoa(defaultSocksPort), cur, cli.ValidatePort)
		if perr != nil {
			return perr
		}
		n, _ := strconv.Atoi(strings.TrimSpace(portStr))
		socksPort = n
	}
	cfg.Socks.Port = socksPort

	// Save config
	if err := config.Save(rt.Paths.ConfigPath, cfg); err != nil {
		return err
	}
	fmt.Println("saved config:", rt.Paths.ConfigPath)

	// Ensure tun2socks
	tun2socksBin, err := ensureTun2SocksBin()
	if err != nil {
		return err
		fmt.Println("tun2socks:", tun2socksBin)
	}

	// Setup routes + start
	if err := setupTunAndRoutes(rt, cfg); err != nil {
		return err
	}

	if tun2socksIsRunning(rt.Paths.Tun2SocksPIDFile) {
		if f.force {
			fmt.Println("tun2socks is running, restarting due to --force")
			_ = stopTun2SocksIfRunning(rt, rt.Paths.Tun2SocksPIDFile)
		} else {
			fmt.Println("tun2socks is already running; skipping start (use --force to restart)")
			return nil
		}
	}

	pid, err := startTun2Socks(rt, tun2socksBin, cfg)
	if err != nil {
		return err
	}
	if err := os.WriteFile(rt.Paths.Tun2SocksPIDFile, []byte(fmt.Sprintf("%d\n", pid)), 0o644); err != nil {
		return err
	}

	fmt.Println("tun2socks pid:", pid)
	fmt.Println("log: /tmp/tun2socks.log")
	return nil
}
