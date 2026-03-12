package commands

import (
	"wslbridge/internal/command"
	initcmd "wslbridge/internal/commands/init"
	statuscmd "wslbridge/internal/commands/status"
	stopcmd "wslbridge/internal/commands/stop"
)

// All returns all registered CLI commands.
func All() []command.Command {
	return []command.Command{
		initcmd.Command{},
		statuscmd.Command{},
		stopcmd.Command{},
	}
}
