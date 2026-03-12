package wsl

import (
	initubuntu "wslbridge/internal/commands/init-ubuntu"
	appruntime "wslbridge/internal/runtime"
)

// Driver implements WSL-specific behavior.
type Driver struct{}

// Name returns the driver name.
func (Driver) Name() string { return "wsl" }

// Init runs WSL initialization.
func (Driver) Init(rt appruntime.Runtime, args []string) error {
	return initubuntu.Command{}.Run(rt, args)
}

// Stop shuts down WSL networking changes.
func (Driver) Stop(rt appruntime.Runtime, args []string) error {
	return initubuntu.StopCommand{}.Run(rt, args)
}

// Status prints WSL status.
func (Driver) Status(rt appruntime.Runtime, args []string) error {
	return initubuntu.StatusCommand{}.Run(rt, args)
}
