package runtime

import (
	"os"
	"path/filepath"
)

type Paths struct {
	ConfigPath       string
	ShareDir         string
	StateDir         string
	DefaultRouteFile string
	Tun2SocksPIDFile string
}

func DefaultPaths() (Paths, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Paths{}, err
	}

	cfgDir := filepath.Join(home, ".config", "wslbridge")
	share := filepath.Join(home, ".local", "share", "wslbridge")
	state := filepath.Join(home, ".local", "state", "wslbridge")

	return Paths{
		ConfigPath:       filepath.Join(cfgDir, "config.yaml"),
		ShareDir:         share,
		StateDir:         state,
		DefaultRouteFile: filepath.Join(state, "default_route.txt"),
		Tun2SocksPIDFile: filepath.Join(state, "tun2socks.pid"),
	}, nil
}
