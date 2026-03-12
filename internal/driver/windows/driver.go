package windows

import (
	"fmt"

	appruntime "wslbridge/internal/runtime"
)

// Driver implements Windows behavior (stub).
type Driver struct{}

// Name returns the driver name.
func (Driver) Name() string { return "windows" }

// Init is not implemented for Windows yet.
func (Driver) Init(rt appruntime.Runtime, args []string) error {
	return fmt.Errorf("windows init is not implemented yet")
}

// Stop is not implemented for Windows yet.
func (Driver) Stop(rt appruntime.Runtime, args []string) error {
	return fmt.Errorf("windows stop is not implemented yet")
}

// Status is not implemented for Windows yet.
func (Driver) Status(rt appruntime.Runtime, args []string) error {
	return fmt.Errorf("windows status is not implemented yet")
}
