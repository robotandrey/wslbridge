package platform

import (
	"fmt"
	"runtime"
	"wslbridge/internal/execx"
)

// Platform abstracts OS-specific dependency management.
type Platform interface {
	Name() string
	EnsureDeps(r execx.Runner) error
}

// Detect returns the current platform implementation.
func Detect() (Platform, error) {
	switch runtime.GOOS {
	case "linux":
		return Ubuntu{}, nil
	case "darwin":
		return Darwin{}, nil
	case "windows":
		return Windows{}, nil
	default:
		return nil, fmt.Errorf("unsupported OS")
	}
}
