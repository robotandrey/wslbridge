package init_ubuntu

import (
	"fmt"
	"io"
	"os/exec"
	"strings"

	"wslbridge/internal/config"
	appruntime "wslbridge/internal/runtime"
)

func getDefaultRouteLine(rt appruntime.Runtime) (string, error) {
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

func defaultIsTun(routeLine string, tunDev string) bool {
	return strings.Contains(routeLine, "default") && strings.Contains(routeLine, "dev "+tunDev)
}

func setupTunAndRoutes(rt appruntime.Runtime, cfg config.Config) error {
	if !linkExists(cfg.Tun.Dev) {
		if err := rt.Runner.Run("sudo", "ip", "tuntap", "add", "mode", "tun", "dev", cfg.Tun.Dev); err != nil {
			return fmt.Errorf("ip tuntap add %s: %w", cfg.Tun.Dev, err)
		}
	}
	if err := rt.Runner.Run("sudo", "ip", "addr", "replace", cfg.Tun.CIDR, "dev", cfg.Tun.Dev); err != nil {
		return fmt.Errorf("ip addr replace %s dev %s: %w", cfg.Tun.CIDR, cfg.Tun.Dev, err)
	}

	if err := rt.Runner.Run("sudo", "ip", "link", "set", cfg.Tun.Dev, "up"); err != nil {
		return fmt.Errorf("ip link set %s up: %w", cfg.Tun.Dev, err)
	}

	if err := rt.Runner.Run("sudo", "ip", "route", "replace", "default", "dev", cfg.Tun.Dev); err != nil {
		return fmt.Errorf("ip route replace default dev %s: %w", cfg.Tun.Dev, err)
	}
	return nil
}

func linkExists(dev string) bool {
	cmd := exec.Command("ip", "link", "show", "dev", dev)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Run() == nil
}
