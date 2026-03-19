package commands

import (
	"wslbridge/internal/command"
	pgbouncercmd "wslbridge/internal/commands/pgbouncer"
	"wslbridge/internal/driver"
	appruntime "wslbridge/internal/runtime"
)

type driverCommand struct {
	name string
	help string
	run  func(d driver.Driver, rt appruntime.Runtime, args []string) error
}

func (c driverCommand) Name() string { return c.name }
func (c driverCommand) Help() string { return c.help }

func (c driverCommand) Run(rt appruntime.Runtime, args []string) error {
	d, err := driver.Detect()
	if err != nil {
		return err
	}
	return c.run(d, rt, args)
}

// All returns all registered CLI commands.
func All() []command.Command {
	return []command.Command{
		driverCommand{
			name: "init",
			help: "Initialize wslbridge for the current OS/environment",
			run: func(d driver.Driver, rt appruntime.Runtime, args []string) error {
				return d.Init(rt, args)
			},
		},
		driverCommand{
			name: "status",
			help: "Show wslbridge status (current OS/environment)",
			run: func(d driver.Driver, rt appruntime.Runtime, args []string) error {
				return d.Status(rt, args)
			},
		},
		driverCommand{
			name: "stop",
			help: "Stop wslbridge and restore routes (current OS/environment)",
			run: func(d driver.Driver, rt appruntime.Runtime, args []string) error {
				return d.Stop(rt, args)
			},
		},
		pgbouncercmd.Command{},
	}
}
