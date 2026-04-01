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
	"wslbridge/internal/tun2socks"
)

// Command implements the Ubuntu/WSL init workflow.
type Command struct{}

// Name returns the command name.
func (Command) Name() string { return "init-ubuntu" }

// Help returns the command description.
func (Command) Help() string {
	return "Install deps, configure WSL DNS/hosts, setup tun2socks routing (Ubuntu/WSL)"
}

const (
	defaultSocksPort = 1080
	defaultTunDev    = "tun0"
	defaultTunCIDR   = "10.0.0.2/24"
)

type flags struct {
	skipDeps          bool
	force             bool
	socksPortOverride int
}

func logStep(msg string) {
	fmt.Println("--", msg)
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

// Run executes the init workflow for Ubuntu/WSL.
func (Command) Run(rt appruntime.Runtime, args []string) error {
	if goruntime.GOOS != "linux" {
		return fmt.Errorf("init-ubuntu supports only linux")
	}

	f, err := parseFlags(args)
	if err != nil {
		return err
	}

	return (&initService{rt: rt, flags: f}).run()
}

func ensureRuntimeDirs(rt appruntime.Runtime) error {
	if err := os.MkdirAll(filepath.Dir(rt.Paths.ConfigPath), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(rt.Paths.ShareDir, 0o755); err != nil {
		return err
	}
	return os.MkdirAll(rt.Paths.StateDir, 0o755)
}

func ensureDeps(rt appruntime.Runtime, skip bool) error {
	if skip {
		return nil
	}
	logStep("Ensuring dependencies are installed (apt)")
	return rt.Platform.EnsureDeps(rt.Runner)
}

func loadConfig(path string) (config.Config, bool, error) {
	if c, err := config.Load(path); err == nil {
		return c, true, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return config.Config{}, false, err
	}
	return config.Config{}, false, nil
}

func applyDefaults(cfg *config.Config) {
	if cfg.Tun.Dev == "" {
		cfg.Tun.Dev = defaultTunDev
	}
	if cfg.Tun.CIDR == "" {
		cfg.Tun.CIDR = defaultTunCIDR
	}
}

func alreadyEnabled(defaultRouteLine string, cfg config.Config, pidFile string) bool {
	if defaultIsTun(defaultRouteLine, cfg.Tun.Dev) && tun2socks.IsRunning(pidFile) {
		fmt.Println("already enabled: default route is on", cfg.Tun.Dev, "and tun2socks is running")
		return true
	}
	return false
}

func ensureSudo(rt appruntime.Runtime) error {
	if err := rt.Runner.Run("sudo", "-v"); err != nil {
		return fmt.Errorf("sudo auth failed: %w", err)
	}
	return nil
}

func configureWSL(rt appruntime.Runtime, cfg *config.Config, hasCfg bool) error {
	pr := cli.NewPrompter(os.Stdin, os.Stdout)

	curDNS := ""
	if hasCfg && cfg.DNS.Nameserver != "" {
		curDNS = cfg.DNS.Nameserver
	}

	dns, err := pr.AskString("DNS nameserver (WSL)", "", curDNS, cli.ValidateIP)
	if err != nil {
		return err
	}
	cfg.DNS.Nameserver = dns

	logStep("Configuring WSL DNS and resolv.conf")
	if err := configureWSLConf(rt); err != nil {
		return err
	}
	if err := writeResolvConf(rt, cfg.DNS.Nameserver); err != nil {
		return err
	}
	fmt.Println("WSL DNS configured. NOTE: wsl.conf changes may require WSL restart.")
	return nil
}

func saveDefaultRoute(rt appruntime.Runtime, defaultRouteLine string) {
	if defaultRouteLine != "" {
		_ = os.WriteFile(rt.Paths.DefaultRouteFile, []byte(defaultRouteLine+"\n"), 0o644)
	}
}

func resolveSocksPort(cfg config.Config, hasCfg bool, f flags) (int, error) {
	socksPort := defaultSocksPort
	if hasCfg && cfg.Socks.Port != 0 {
		socksPort = cfg.Socks.Port
	}
	if f.socksPortOverride != 0 {
		return f.socksPortOverride, nil
	}
	if !f.force {
		return socksPort, nil
	}

	pr := cli.NewPrompter(os.Stdin, os.Stdout)
	cur := ""
	if hasCfg && cfg.Socks.Port != 0 {
		cur = strconv.Itoa(cfg.Socks.Port)
	}
	portStr, perr := pr.AskString("SOCKS port", strconv.Itoa(defaultSocksPort), cur, cli.ValidatePort)
	if perr != nil {
		return 0, perr
	}
	n, _ := strconv.Atoi(strings.TrimSpace(portStr))
	return n, nil
}

func handleRunningTun2Socks(rt appruntime.Runtime, f flags, pidFile string) bool {
	if !tun2socks.IsRunning(pidFile) {
		return false
	}
	if f.force {
		fmt.Println("tun2socks is running, restarting due to --force")
		_ = tun2socks.StopIfRunning(rt, pidFile)
		return false
	}
	fmt.Println("tun2socks is already running; skipping start (use --force to restart)")
	return true
}
