package initc

import (
	"fmt"
	goruntime "runtime"

	initubuntu "wslbridge/internal/commands/init-ubuntu"
	appruntime "wslbridge/internal/runtime"
)

type Command struct{}

func (Command) Name() string { return "init" }
func (Command) Help() string { return "Initialize wslbridge for the current OS/environment" }

func (Command) Run(rt appruntime.Runtime, args []string) error {
	switch goruntime.GOOS {
	case "linux":
		// пока поддерживаем только WSL/Ubuntu init
		if initubuntu.IsWSL() {
			return initubuntu.Command{}.Run(rt, args)
		}
		return fmt.Errorf("linux is supported only in WSL for now")
	case "darwin":
		return fmt.Errorf("macOS init is not implemented yet")
	case "windows":
		return fmt.Errorf("windows init is not implemented yet")
	default:
		return fmt.Errorf("unsupported OS: %s", goruntime.GOOS)
	}
}
