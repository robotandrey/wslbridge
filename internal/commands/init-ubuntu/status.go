package init_ubuntu

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	appruntime "wslbridge/internal/runtime"
	"wslbridge/internal/tun2socks"
)

// StatusCommand implements the Ubuntu/WSL status workflow.
type StatusCommand struct{}

// Name returns the command name.
func (StatusCommand) Name() string { return "status-ubuntu" }

// Help returns the command description.
func (StatusCommand) Help() string { return "Show tun2socks and routing status (Ubuntu/WSL)" }

// Run executes the status workflow for Ubuntu/WSL.
func (StatusCommand) Run(rt appruntime.Runtime, args []string) error {
	if err := parseStatusFlags(args); err != nil {
		return err
	}

	cfg, _, err := loadConfig(rt.Paths.ConfigPath)
	if err != nil {
		return err
	}
	applyDefaults(&cfg)

	defaultRouteLine, err := getDefaultRouteLine(rt)
	if err != nil {
		return err
	}

	fmt.Println("Config:", rt.Paths.ConfigPath)
	fmt.Println("WSL:", isWSL())

	routeLine := defaultRouteLine
	if strings.TrimSpace(routeLine) == "" {
		routeLine = "(none)"
	}
	fmt.Println("Default route:", routeLine)
	fmt.Println("Default is tun:", defaultIsTun(defaultRouteLine, cfg.Tun.Dev))

	fmt.Println("Tun dev:", cfg.Tun.Dev)
	fmt.Println("Tun link:", boolLabel(linkExists(cfg.Tun.Dev)))

	if cfg.Socks.Host != "" && cfg.Socks.Port != 0 {
		fmt.Printf("SOCKS: %s:%d\n", cfg.Socks.Host, cfg.Socks.Port)
	} else {
		fmt.Println("SOCKS: (not configured)")
	}

	running := tun2socks.IsRunning(rt.Paths.Tun2SocksPIDFile)
	fmt.Println("Tun2socks running:", boolLabel(running))
	fmt.Println("Tun2socks pid file:", rt.Paths.Tun2SocksPIDFile)
	if pid, ok := readPID(rt.Paths.Tun2SocksPIDFile); ok {
		if running {
			fmt.Println("Tun2socks pid:", pid)
		} else {
			fmt.Println("Tun2socks pid:", pid, "(stale)")
		}
	} else {
		fmt.Println("Tun2socks pid:", "(not found)")
	}

	return nil
}

func parseStatusFlags(args []string) error {
	for _, a := range args {
		switch a {
		default:
			return fmt.Errorf("unknown arg: %s", a)
		}
	}
	return nil
}

func readPID(path string) (int, bool) {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	pidStr := strings.TrimSpace(string(b))
	if pidStr == "" {
		return 0, false
	}
	pid, err := strconv.Atoi(pidStr)
	if err != nil || pid <= 0 {
		return 0, false
	}
	return pid, true
}

func boolLabel(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}
