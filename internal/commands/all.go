package commands

import (
	"wslbridge/internal/command"
	initcmd "wslbridge/internal/commands/init"
)

func All() []command.Command {
	return []command.Command{
		initcmd.Command{},
	}
}
