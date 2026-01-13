package commands

import (
	"wslbridge/internal/command"
	initcmd "wslbridge/internal/commands/init"
	stopcmd "wslbridge/internal/commands/stop"
)

func All() []command.Command {
	return []command.Command{
		initcmd.Command{},
		stopcmd.Command{},
	}
}
