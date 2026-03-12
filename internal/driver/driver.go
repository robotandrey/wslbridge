package driver

import "wslbridge/internal/runtime"

// Driver encapsulates OS/environment-specific behavior.
type Driver interface {
	Name() string
	Init(rt runtime.Runtime, args []string) error
	Stop(rt runtime.Runtime, args []string) error
	Status(rt runtime.Runtime, args []string) error
}
