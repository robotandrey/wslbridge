package stopc

import (
	"wslbridge/internal/driver"
	appruntime "wslbridge/internal/runtime"
)

// Command implements the stop CLI command.
type Command struct{}

// Name returns the command name.
func (Command) Name() string { return "stop" }

// Help returns the command description.
func (Command) Help() string { return "Stop wslbridge and restore routes (current OS/environment)" }

// Run executes the stop workflow.
func (Command) Run(rt appruntime.Runtime, args []string) error {
	d, err := driver.Detect()
	if err != nil {
		return err
	}
	return d.Stop(rt, args)
}
