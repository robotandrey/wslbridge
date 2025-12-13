package platform

import (
	"fmt"
	"runtime"
	"wslbridge/internal/execx"
)

type Platform interface {
	Name() string
	EnsureDeps(r execx.Runner) error
}

func Detect() (Platform, error) {
	switch runtime.GOOS {
	case "linux":
		return Ubuntu{}, nil
	case "darwin":
		// return MacOS{}, nil
		return nil, fmt.Errorf("unsupported OS")
	default:
		return nil, fmt.Errorf("unsupported OS")
	}
}
