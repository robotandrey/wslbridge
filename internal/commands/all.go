package commands

import "wslbridge/internal/command"

func All() []command.Command {
	return []command.Command{
		InitUbuntu{},
		// AddDB{},
		// Up{},
		// Down{},
	}
}
