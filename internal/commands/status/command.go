package statusc

import (
	"wslbridge/internal/driver"
	appruntime "wslbridge/internal/runtime"
)

// Command implements the status CLI command.
type Command struct{}

// Name returns the command name.
func (Command) Name() string { return "status" }

// Help returns the command description.
func (Command) Help() string { return "Show wslbridge status (current OS/environment)" }

// Run executes the status workflow.
func (Command) Run(rt appruntime.Runtime, args []string) error {
	d, err := driver.Detect()
	if err != nil {
		return err
	}
	return d.Status(rt, args)
}
