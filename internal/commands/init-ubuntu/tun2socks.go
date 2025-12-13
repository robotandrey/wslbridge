package init_ubuntu

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"wslbridge/internal/config"
	appruntime "wslbridge/internal/runtime"
)

func ensureTun2SocksBin() (string, error) {
	if p, err := exec.LookPath("tun2socks"); err == nil {
		return p, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

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

	if p, err := exec.LookPath("tun2socks"); err == nil {
		return p, nil
	}
	if isExecutable(bin) {
		return bin, nil
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

func startTun2Socks(rt appruntime.Runtime, bin string, cfg config.Config) (int, error) {
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

func tun2socksIsRunning(pidFile string) bool {
	b, err := os.ReadFile(pidFile)
	if err != nil {
		// Fall back to detecting any running tun2socks process
		cmd := exec.Command("pgrep", "-x", "tun2socks")
		out, perr := cmd.Output()
		if perr != nil {
			return false
		}
		return strings.TrimSpace(string(out)) != ""
	}

	pidStr := strings.TrimSpace(string(b))
	if pidStr == "" {
		return false
	}
	cmd := exec.Command("bash", "-lc", "kill -0 "+pidStr+" >/dev/null 2>&1")
	return cmd.Run() == nil
}

func stopTun2SocksIfRunning(rt appruntime.Runtime, pidFile string) error {
	b, err := os.ReadFile(pidFile)
	if err != nil {
		return nil
	}
	pidStr := strings.TrimSpace(string(b))
	if pidStr == "" {
		return nil
	}
	// попытка аккуратно убить
	_ = rt.Runner.Run("bash", "-lc", "sudo kill "+pidStr+" >/dev/null 2>&1 || true")
	_ = os.Remove(pidFile)
	return nil
}
