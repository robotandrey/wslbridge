package init_ubuntu

import (
	"fmt"
	"os"

	"wslbridge/internal/config"
	appruntime "wslbridge/internal/runtime"
	"wslbridge/internal/tun2socks"
)

// initService orchestrates the init workflow while keeping Command.Run minimal.
type initService struct {
	rt               appruntime.Runtime
	flags            flags
	cfg              config.Config
	hasCfg           bool
	defaultRouteLine string
	tun2socksBin     string
}

func (s *initService) run() error {
	if err := s.prepare(); err != nil {
		return err
	}

	if alreadyEnabled(s.defaultRouteLine, s.cfg, s.rt.Paths.Tun2SocksPIDFile) {
		return nil
	}

	if err := s.configureNetworkInputs(); err != nil {
		return err
	}

	return s.enableTrafficBridge()
}

func (s *initService) prepare() error {
	if err := ensureRuntimeDirs(s.rt); err != nil {
		return err
	}
	if err := ensureDeps(s.rt, s.flags.skipDeps); err != nil {
		return err
	}

	cfg, hasCfg, err := loadConfig(s.rt.Paths.ConfigPath)
	if err != nil {
		return err
	}
	s.cfg = cfg
	s.hasCfg = hasCfg
	applyDefaults(&s.cfg)

	defaultRouteLine, err := getDefaultRouteLine(s.rt)
	if err != nil {
		return err
	}
	s.defaultRouteLine = defaultRouteLine
	return nil
}

func (s *initService) configureNetworkInputs() error {
	if err := ensureSudo(s.rt); err != nil {
		return err
	}

	if isWSL() {
		if err := configureWSL(s.rt, &s.cfg, s.hasCfg); err != nil {
			return err
		}
	}

	saveDefaultRoute(s.rt, s.defaultRouteLine)

	logStep("Detecting SOCKS gateway")
	gw, err := detectSocksGateway(s.rt, s.cfg, s.defaultRouteLine)
	if err != nil {
		return err
	}
	s.cfg.Socks.Host = gw
	fmt.Println("SOCKS gateway:", s.cfg.Socks.Host)

	socksPort, err := resolveSocksPort(s.cfg, s.hasCfg, s.flags)
	if err != nil {
		return err
	}
	s.cfg.Socks.Port = socksPort

	logStep("Saving configuration")
	if err := config.Save(s.rt.Paths.ConfigPath, s.cfg); err != nil {
		return err
	}
	fmt.Println("saved config:", s.rt.Paths.ConfigPath)

	return nil
}

func (s *initService) enableTrafficBridge() error {
	logStep("Ensuring tun2socks binary is available")
	tun2socksBin, err := tun2socks.EnsureBin()
	if err != nil {
		return err
	}
	s.tun2socksBin = tun2socksBin
	fmt.Println("tun2socks:", s.tun2socksBin)

	logStep("Configuring tun interface and default route")
	if err := setupTunAndRoutes(s.rt, s.cfg); err != nil {
		return err
	}

	if handleRunningTun2Socks(s.rt, s.flags, s.rt.Paths.Tun2SocksPIDFile) {
		return nil
	}

	logStep("Starting tun2socks daemon")
	pid, err := tun2socks.Start(s.tun2socksBin, s.cfg)
	if err != nil {
		return err
	}
	if err := os.WriteFile(s.rt.Paths.Tun2SocksPIDFile, []byte(fmt.Sprintf("%d\n", pid)), 0o644); err != nil {
		return err
	}

	fmt.Println("tun2socks pid:", pid)
	fmt.Println("log: /tmp/tun2socks.log")
	return nil
}
