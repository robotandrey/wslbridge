package tun2socks

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"wslbridge/internal/config"
	appruntime "wslbridge/internal/runtime"
)

// EnsureBin locates or installs the tun2socks binary.
func EnsureBin() (string, error) {
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

	cmd := exec.Command("go", "install", "github.com/xjasonlyu/tun2socks/v2@v2.6.0")
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

// Start launches tun2socks and returns its PID.
func Start(bin string, cfg config.Config, logPath string) (int, error) {
	logf, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return 0, fmt.Errorf("open tun2socks log: %w", err)
	}
	defer logf.Close()

	device := "tun://" + cfg.Tun.Dev
	proxy := fmt.Sprintf("socks5://%s:%d", cfg.Socks.Host, cfg.Socks.Port)

	cmd := exec.Command("sudo", "nohup", bin, "-device", device, "-proxy", proxy, "-loglevel", "info")
	cmd.Stdout = logf
	cmd.Stderr = logf

	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("start tun2socks: %w", err)
	}

	pid, err := waitForPID(10 * time.Second)
	if err != nil {
		return 0, fmt.Errorf("start tun2socks: %w", err)
	}
	return pid, nil
}

// IsRunning reports whether tun2socks is running.
func IsRunning(pidFile string) bool {
	b, err := os.ReadFile(pidFile)
	if err != nil {
		return anyRunning()
	}

	pidStr := strings.TrimSpace(string(b))
	if pidStr == "" {
		return false
	}
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return false
	}
	cmd := exec.Command("kill", "-0", pidStr)
	if err := cmd.Run(); err == nil {
		return true
	}
	return pidInList(pid)
}

// StopIfRunning stops tun2socks if it is running.
func StopIfRunning(rt appruntime.Runtime, pidFile string) error {
	b, err := os.ReadFile(pidFile)
	if err != nil {
		return stopByName(rt)
	}
	pidStr := strings.TrimSpace(string(b))
	if pidStr == "" {
		return stopByName(rt)
	}
	if _, err := strconv.Atoi(pidStr); err != nil {
		return stopByName(rt)
	}
	_ = rt.Runner.Run("sudo", "kill", pidStr)
	_ = os.Remove(pidFile)
	return nil
}

func isExecutable(path string) bool {
	st, err := os.Stat(path)
	if err != nil {
		return false
	}
	return (st.Mode() & 0o111) != 0
}

func waitForPID(timeout time.Duration) (int, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if pid, ok := anyPID(); ok {
			return pid, nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return 0, fmt.Errorf("timed out waiting for tun2socks pid")
}

func anyPID() (int, bool) {
	out, err := exec.Command("pgrep", "-x", "tun2socks").Output()
	if err != nil {
		return 0, false
	}
	fields := strings.Fields(string(out))
	if len(fields) == 0 {
		return 0, false
	}
	pid, err := strconv.Atoi(fields[0])
	if err != nil || pid <= 0 {
		return 0, false
	}
	return pid, true
}

func anyRunning() bool {
	out, err := exec.Command("pgrep", "-x", "tun2socks").Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) != ""
}

func pidInList(pid int) bool {
	out, err := exec.Command("pgrep", "-x", "tun2socks").Output()
	if err != nil {
		return false
	}
	for _, field := range strings.Fields(string(out)) {
		n, err := strconv.Atoi(field)
		if err != nil {
			continue
		}
		if n == pid {
			return true
		}
	}
	return false
}

func stopByName(rt appruntime.Runtime) error {
	out, err := exec.Command("pgrep", "-x", "tun2socks").Output()
	if err != nil {
		return nil
	}
	for _, field := range strings.Fields(string(out)) {
		if _, err := strconv.Atoi(field); err != nil {
			continue
		}
		_ = rt.Runner.Run("sudo", "kill", field)
	}
	return nil
}
