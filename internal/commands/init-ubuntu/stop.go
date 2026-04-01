package init_ubuntu

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"wslbridge/internal/config"
	appruntime "wslbridge/internal/runtime"
	"wslbridge/internal/tun2socks"
)

// StopCommand implements the Ubuntu/WSL stop workflow.
type StopCommand struct{}

// Name returns the command name.
func (StopCommand) Name() string { return "stop-ubuntu" }

// Help returns the command description.
func (StopCommand) Help() string { return "Stop tun2socks and restore routes (Ubuntu/WSL)" }

// Run executes the stop workflow for Ubuntu/WSL.
func (StopCommand) Run(rt appruntime.Runtime, args []string) error {
	if err := parseStopFlags(args); err != nil {
		return err
	}

	var cfg config.Config
	if c, err := config.Load(rt.Paths.ConfigPath); err == nil {
		cfg = c
	} else if !errors.Is(err, os.ErrNotExist) {
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
	if err := tun2socks.StopIfRunning(rt, rt.Paths.Tun2SocksPIDFile); err != nil {
		return err
	}

	logStep("Removing tun interface (if present)")
	_ = rt.Runner.Run("sudo", "ip", "link", "set", cfg.Tun.Dev, "down")
	_ = rt.Runner.Run("sudo", "ip", "tuntap", "del", "mode", "tun", "dev", cfg.Tun.Dev)

	if isWSL() {
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
	args, err := buildRestoreDefaultRouteArgs(line)
	if err != nil {
		return err
	}
	if err := rt.Runner.Run("sudo", args...); err != nil {
		return fmt.Errorf("restore default route: %w", err)
	}
	return nil
}

func buildRestoreDefaultRouteArgs(line string) ([]string, error) {
	fields := strings.Fields(strings.TrimSpace(line))
	if len(fields) == 0 || fields[0] != "default" {
		return nil, fmt.Errorf("invalid default route line: %q", line)
	}

	pairKeys := map[string]bool{
		"via":      true,
		"dev":      true,
		"metric":   true,
		"src":      true,
		"proto":    true,
		"scope":    true,
		"table":    true,
		"mtu":      true,
		"advmss":   true,
		"rtt":      true,
		"rttvar":   true,
		"window":   true,
		"cwnd":     true,
		"initcwnd": true,
		"initrwnd": true,
		"pref":     true,
	}
	flagKeys := map[string]bool{
		"onlink": true,
	}

	args := []string{"ip", "route", "replace", "default"}
	hasNextHop := false
	for i := 1; i < len(fields); i++ {
		token := fields[i]
		switch {
		case pairKeys[token]:
			if i+1 >= len(fields) {
				return nil, fmt.Errorf("invalid default route line: %q", line)
			}
			args = append(args, token, fields[i+1])
			if token == "via" || token == "dev" {
				hasNextHop = true
			}
			i++
		case flagKeys[token]:
			args = append(args, token)
		default:
			// Ignore kernel-emitted tokens that are not safe to replay verbatim.
		}
	}

	if !hasNextHop {
		return nil, fmt.Errorf("invalid default route line: %q", line)
	}
	return args, nil
}
