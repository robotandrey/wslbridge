package command

import "wslbridge/internal/runtime"

// Command defines a CLI command contract.
type Command interface {
	Name() string
	Help() string
	Run(rt runtime.Runtime, args []string) error
}
