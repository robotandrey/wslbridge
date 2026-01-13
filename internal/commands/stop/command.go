package stopc

import (
	"fmt"
	goruntime "runtime"

	initubuntu "wslbridge/internal/commands/init-ubuntu"
	appruntime "wslbridge/internal/runtime"
)

type Command struct{}

func (Command) Name() string { return "stop" }
func (Command) Help() string { return "Stop wslbridge and restore routes (current OS/environment)" }

func (Command) Run(rt appruntime.Runtime, args []string) error {
	switch goruntime.GOOS {
	case "linux":
		if initubuntu.IsWSL() {
			return initubuntu.StopCommand{}.Run(rt, args)
		}
		return fmt.Errorf("linux is supported only in WSL for now")
	case "darwin":
		return fmt.Errorf("macOS stop is not implemented yet")
	case "windows":
		return fmt.Errorf("windows stop is not implemented yet")
	default:
		return fmt.Errorf("unsupported OS: %s", goruntime.GOOS)
	}
}
