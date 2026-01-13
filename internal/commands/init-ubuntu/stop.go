package init_ubuntu

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"wslbridge/internal/config"
	appruntime "wslbridge/internal/runtime"
)

type StopCommand struct{}

func (StopCommand) Name() string { return "stop-ubuntu" }
func (StopCommand) Help() string { return "Stop tun2socks and restore routes (Ubuntu/WSL)" }

func (StopCommand) Run(rt appruntime.Runtime, args []string) error {
	if err := parseStopFlags(args); err != nil {
		return err
	}

	// load config (optional)
	var cfg config.Config
	if c, err := config.Load(rt.Paths.ConfigPath); err == nil {
		cfg = c
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	if cfg.Tun.Dev == "" {
		cfg.Tun.Dev = "tun0"
	}

	restoredRoute := false
	if b, err := os.ReadFile(rt.Paths.DefaultRouteFile); err == nil {
		line := strings.TrimSpace(string(b))
		if line != "" {
			logStep("Restoring default route")
			if err := restoreDefaultRoute(rt, line); err != nil {
				return err
			}
			restoredRoute = true
		}
	}
	if !restoredRoute {
		logStep("Default route backup not found; skipping route restore")
	}

	logStep("Stopping tun2socks (if running)")
	if err := stopTun2SocksIfRunning(rt, rt.Paths.Tun2SocksPIDFile); err != nil {
		return err
	}

	logStep("Removing tun interface (if present)")
	_ = rt.Runner.Run("sudo", "ip", "link", "set", cfg.Tun.Dev, "down")
	_ = rt.Runner.Run("sudo", "ip", "tuntap", "del", "mode", "tun", "dev", cfg.Tun.Dev)

	if IsWSL() {
		logStep("Restoring WSL config (if backup exists)")
		if restored, err := restoreWSLConf(rt); err != nil {
			return err
		} else if restored {
			fmt.Println("restored /etc/wsl.conf (may require WSL restart)")
		}

		if restored, err := restoreResolvConf(rt); err != nil {
			return err
		} else if restored {
			fmt.Println("restored /etc/resolv.conf (may require WSL restart)")
		}
	}

	return nil
}

func parseStopFlags(args []string) error {
	for _, a := range args {
		switch a {
		default:
			return fmt.Errorf("unknown arg: %s", a)
		}
	}
	return nil
}

func restoreDefaultRoute(rt appruntime.Runtime, line string) error {
	fields := strings.Fields(line)
	if len(fields) == 0 || fields[0] != "default" {
		return fmt.Errorf("invalid default route line: %q", line)
	}
	args := append([]string{"ip", "route", "replace"}, fields...)
	if err := rt.Runner.Run("sudo", args...); err != nil {
		return fmt.Errorf("restore default route: %w", err)
	}
	return nil
}
