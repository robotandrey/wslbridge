package commands

import (
	"wslbridge/internal/command"
	init "wslbridge/internal/commands/init"
)

func All() []command.Command {
	return []command.Command{
		init.Command{},
	}
}
