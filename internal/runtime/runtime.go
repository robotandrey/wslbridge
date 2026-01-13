package runtime

import (
	"wslbridge/internal/execx"
	"wslbridge/internal/platform"
)

type Runtime struct {
	Paths    Paths
	Runner   execx.Runner
	Platform platform.Platform
}

func New(r execx.Runner, p platform.Platform) (Runtime, error) {
	paths, err := DefaultPaths()
	if err != nil {
		return Runtime{}, err
	}
	// default: prefer project-local config when in wslbridge repo
	cfgPath, err := ResolveConfigPath()
	if err != nil {
		return Runtime{}, err
	}
	paths.ConfigPath = cfgPath

	return Runtime{Paths: paths, Runner: r, Platform: p}, nil
}
