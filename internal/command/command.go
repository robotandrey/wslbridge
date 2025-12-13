package command

import "wslbridge/internal/runtime"

type Command interface {
	Name() string
	Help() string
	Run(rt runtime.Runtime, args []string) error
}
