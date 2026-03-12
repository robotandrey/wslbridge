package platform

import (
	"fmt"

	"wslbridge/internal/execx"
)

// Windows is a stub platform for Windows.
type Windows struct{}

// Name returns the platform name.
func (Windows) Name() string { return "windows" }

// EnsureDeps is not implemented for Windows yet.
func (Windows) EnsureDeps(r execx.Runner) error {
	return fmt.Errorf("windows deps are not implemented yet")
}
