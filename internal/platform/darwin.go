package platform

import (
	"fmt"

	"wslbridge/internal/execx"
)

// Darwin is a stub platform for macOS.
type Darwin struct{}

// Name returns the platform name.
func (Darwin) Name() string { return "darwin" }

// EnsureDeps is not implemented for macOS yet.
func (Darwin) EnsureDeps(r execx.Runner) error {
	return fmt.Errorf("macOS deps are not implemented yet")
}
