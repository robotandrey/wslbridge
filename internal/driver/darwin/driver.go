package darwin

import (
	"fmt"

	appruntime "wslbridge/internal/runtime"
)

// Driver implements macOS behavior (stub).
type Driver struct{}

// Name returns the driver name.
func (Driver) Name() string { return "darwin" }

// Init is not implemented for macOS yet.
func (Driver) Init(rt appruntime.Runtime, args []string) error {
	return fmt.Errorf("macOS init is not implemented yet")
}

// Stop is not implemented for macOS yet.
func (Driver) Stop(rt appruntime.Runtime, args []string) error {
	return fmt.Errorf("macOS stop is not implemented yet")
}

// Status is not implemented for macOS yet.
func (Driver) Status(rt appruntime.Runtime, args []string) error {
	return fmt.Errorf("macOS status is not implemented yet")
}
