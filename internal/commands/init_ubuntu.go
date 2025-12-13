package commands

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strconv"
	"strings"

	"wslbridge/internal/cli"
	"wslbridge/internal/config"
	appruntime "wslbridge/internal/runtime"
)

type InitUbuntu struct{}

func (InitUbuntu) Name() string { return "init-ubuntu" }
func (InitUbuntu) Help() string {
	return "Install deps, auto-detect SOCKS gateway, setup tun2socks routing (Ubuntu/WSL)"
}

func (InitUbuntu) Run(rt appruntime.Runtime, args []string) error {
	if goruntime.GOOS != "linux" {
		return fmt.Errorf("init-ubuntu supports only linux")
	}

	// Flags
	skipDeps := false
	force := false
	socksPortOverride := 0

	for _, a := range args {
		switch {
		case a == "--skip-deps":
			skipDeps = true
		case a == "--force":
			force = true
		case strings.HasPrefix(a, "--socks-port="):
			v := strings.TrimPrefix(a, "--socks-port=")
			n, err := strconv.Atoi(v)
			if err != nil || n < 1 || n > 65535 {
				return fmt.Errorf("invalid --socks-port=%q (must be 1..65535)", v)
			}
			socksPortOverride = n
		default:
			return fmt.Errorf("unknown arg: %s", a)
		}
	}

	// Ensure dirs
	if err := os.MkdirAll(filepath.Dir(rt.Paths.ConfigPath), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(rt.Paths.ShareDir, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(rt.Paths.StateDir, 0o755); err != nil {
		return err
	}

	// deps (apt)
	if !skipDeps {
		if err := rt.Platform.EnsureDeps(rt.Runner); err != nil {
			return err
		}
	}

	// Load existing config (optional)
	var cfg config.Config
	hasCfg := false
	if c, err := config.Load(rt.Paths.ConfigPath); err == nil {
		cfg = c
		hasCfg = true
	} else {
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}

	// Defaults
	const (
		defaultSocksPort = 1080
		defaultTunDev    = "tun0"
		defaultTunCIDR   = "10.0.0.2/24"
	)

	// Determine current default route
	defaultRouteLine, err := getDefaultRouteLine(rt)
	if err != nil {
		return err
	}

	// Save original default route line to state (for down, and for gateway detection fallback)
	if defaultRouteLine != "" {
		_ = os.WriteFile(rt.Paths.DefaultRouteFile, []byte(defaultRouteLine+"\n"), 0o644)
	}

	// Detect SOCKS gateway
	gatewayIP, err := detectGatewayIP(defaultRouteLine, rt.Paths.DefaultRouteFile)
	if err != nil {
		return err
	}
	fmt.Println("detected SOCKS gateway:", gatewayIP)

	// Socks port: minimal interaction.
	// Priority: flag override > existing config > default > prompt if force or config missing? (keep minimal)
	socksPort := defaultSocksPort
	if hasCfg && cfg.Socks.Port != 0 {
		socksPort = cfg.Socks.Port
	}
	if socksPortOverride != 0 {
		socksPort = socksPortOverride
	} else {
		// If config exists: show current; user can press Enter (keep current).
		// If no config: default is enough; no need to ask unless you WANT to allow change.
		// We'll ask only when --force is passed (explicit intent).
		if force {
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
	}

	// TUN params: no questions, "как у всех", but respect config if set.
	tunDev := defaultTunDev
	tunCIDR := defaultTunCIDR
	if hasCfg {
		if cfg.Tun.Dev != "" {
			tunDev = cfg.Tun.Dev
		}
		if cfg.Tun.CIDR != "" {
			tunCIDR = cfg.Tun.CIDR
		}
	}

	// Update and save config automatically
	cfg.Socks.Host = gatewayIP
	cfg.Socks.Port = socksPort
	cfg.Tun.Dev = tunDev
	cfg.Tun.CIDR = tunCIDR

	if err := config.Save(rt.Paths.ConfigPath, cfg); err != nil {
		return err
	}
	fmt.Println("saved config:", rt.Paths.ConfigPath)

	// Ensure tun2socks installed
	tun2socksBin, err := ensureTun2SocksBin()
	if err != nil {
		return err
	}
	fmt.Println("tun2socks:", tun2socksBin)

	// sudo auth once (password prompt here)
	if err := rt.Runner.Run("sudo", "-v"); err != nil {
		return fmt.Errorf("sudo auth failed: %w", err)
	}

	// Setup tun + default route exactly like script
	if err := setupTunAndRoutes(rt, cfg); err != nil {
		return err
	}

	// Start tun2socks like script (nohup + log), capture PID
	pid, err := startTun2Socks(rt, tun2socksBin, cfg)
	if err != nil {
		return err
	}

	if err := os.WriteFile(rt.Paths.Tun2SocksPIDFile, []byte(fmt.Sprintf("%d\n", pid)), 0o644); err != nil {
		return err
	}

	fmt.Println("tun2socks pid:", pid)
	fmt.Println("log: /tmp/tun2socks.log")

	// Smoke test (best-effort)
	_ = rt.Runner.Run("bash", "-lc", `sleep 1; curl -s --max-time 10 ifconfig.me && echo`)

	return nil
}

// --- Helpers ---

func getDefaultRouteLine(rt appruntime.Runtime) (string, error) {
	// "ip route show default" can output multiple lines; take first non-empty.
	out, err := rt.Runner.RunCapture("ip", "route", "show", "default")
	if err != nil {
		return "", fmt.Errorf("ip route show default failed: %w", err)
	}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line, nil
		}
	}
	return "", nil
}

func detectGatewayIP(currentDefaultRouteLine, savedDefaultRouteFile string) (string, error) {
	if gw, ok := parseViaIP(currentDefaultRouteLine); ok {
		return gw, nil
	}

	// If current is "default dev tun0 ..." there is no "via".
	// Fall back to saved original route.
	if savedDefaultRouteFile != "" {
		if b, err := os.ReadFile(savedDefaultRouteFile); err == nil {
			if gw, ok := parseViaIP(string(b)); ok {
				return gw, nil
			}
			return "", fmt.Errorf("saved default route has no 'via': %q", strings.TrimSpace(string(b)))
		}
	}

	return "", fmt.Errorf(
		"could not detect default gateway: current default route has no 'via' (%q). "+
			"Run this command BEFORE switching default route to tun, or keep a saved default route in %s",
		strings.TrimSpace(currentDefaultRouteLine),
		savedDefaultRouteFile,
	)
}

func parseViaIP(routeLine string) (string, bool) {
	fields := strings.Fields(strings.TrimSpace(routeLine))
	for i := 0; i < len(fields)-1; i++ {
		if fields[i] == "via" {
			return fields[i+1], true
		}
	}
	return "", false
}

func ensureTun2SocksBin() (string, error) {
	// 1) If tun2socks already in PATH — use it
	if p, err := exec.LookPath("tun2socks"); err == nil {
		return p, nil
	}

	// 2) Common Go install locations
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	// Prefer $GOBIN if set
	if gobin := os.Getenv("GOBIN"); gobin != "" {
		bin := filepath.Join(gobin, "tun2socks")
		if isExecutable(bin) {
			return bin, nil
		}
	}

	bin := filepath.Join(home, "go", "bin", "tun2socks")
	if isExecutable(bin) {
		return bin, nil
	}

	// 3) Install via go
	if _, err := exec.LookPath("go"); err != nil {
		return "", fmt.Errorf("tun2socks not found and go is not installed")
	}

	cmd := exec.Command("go", "install", "github.com/eycorsican/go-tun2socks/cmd/tun2socks@latest")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to install tun2socks: %w", err)
	}

	// Re-check after install
	if p, err := exec.LookPath("tun2socks"); err == nil {
		return p, nil
	}
	if isExecutable(bin) {
		return bin, nil
	}
	if gobin := os.Getenv("GOBIN"); gobin != "" {
		b2 := filepath.Join(gobin, "tun2socks")
		if isExecutable(b2) {
			return b2, nil
		}
	}

	return "", fmt.Errorf("tun2socks install finished but binary not found (check your GOBIN / GOPATH)")
}

func isExecutable(path string) bool {
	st, err := os.Stat(path)
	if err != nil {
		return false
	}
	return (st.Mode() & 0o111) != 0
}

func setupTunAndRoutes(rt appruntime.Runtime, cfg config.Config) error {
	// Create tun (ignore if exists)
	_ = rt.Runner.Run("sudo", "ip", "tuntap", "add", "mode", "tun", "dev", cfg.Tun.Dev)
	_ = rt.Runner.Run("sudo", "ip", "addr", "add", cfg.Tun.CIDR, "dev", cfg.Tun.Dev)

	if err := rt.Runner.Run("sudo", "ip", "link", "set", cfg.Tun.Dev, "up"); err != nil {
		return fmt.Errorf("ip link set %s up: %w", cfg.Tun.Dev, err)
	}

	// Replace default route
	_ = rt.Runner.Run("sudo", "ip", "route", "del", "default")
	if err := rt.Runner.Run("sudo", "ip", "route", "add", "default", "dev", cfg.Tun.Dev); err != nil {
		return fmt.Errorf("ip route add default dev %s: %w", cfg.Tun.Dev, err)
	}

	return nil
}

func startTun2Socks(rt appruntime.Runtime, bin string, cfg config.Config) (int, error) {
	// Like the script:
	// sudo nohup BIN -device tun://tun0 -proxy socks5://IP:PORT -loglevel info >/tmp/tun2socks.log 2>&1 & echo $!
	cmd := fmt.Sprintf(
		`sudo nohup %q -device "tun://%s" -proxy "socks5://%s:%d" -loglevel info >/tmp/tun2socks.log 2>&1 & echo $!`,
		bin, cfg.Tun.Dev, cfg.Socks.Host, cfg.Socks.Port,
	)

	out, err := rt.Runner.RunCapture("bash", "-lc", cmd)
	if err != nil {
		return 0, fmt.Errorf("start tun2socks: %w", err)
	}

	pidStr := strings.TrimSpace(out)
	pid, err := strconv.Atoi(pidStr)
	if err != nil || pid <= 0 {
		return 0, fmt.Errorf("unexpected tun2socks pid output: %q", pidStr)
	}
	return pid, nil
}
