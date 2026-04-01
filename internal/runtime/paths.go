package runtime

import (
	"os"
	"path/filepath"
)

// Paths groups filesystem paths used by the app.
type Paths struct {
	ConfigPath       string
	ShareDir         string
	StateDir         string
	DefaultRouteFile string
	Tun2SocksPIDFile string
	Tun2SocksLogFile string
	DBProxyPIDFile   string
	DBProxyMetaFile  string
	DBProxyLogFile   string
}

// DefaultPaths returns default user-scoped paths.
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
		Tun2SocksLogFile: filepath.Join(state, "tun2socks.log"),
		DBProxyPIDFile:   filepath.Join(state, "db-proxy.pid"),
		DBProxyMetaFile:  filepath.Join(state, "db-proxy.json"),
		DBProxyLogFile:   filepath.Join(state, "db-proxy.log"),
	}, nil
}
