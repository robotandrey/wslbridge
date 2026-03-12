package driver

import (
	"fmt"
	"runtime"

	driverdarwin "wslbridge/internal/driver/darwin"
	driverwindows "wslbridge/internal/driver/windows"
	driverwsl "wslbridge/internal/driver/wsl"
	"wslbridge/internal/env"
)

// Detect selects a driver for the current OS/environment.
func Detect() (Driver, error) {
	switch runtime.GOOS {
	case "linux":
		if env.IsWSL() {
			return driverwsl.Driver{}, nil
		}
		return nil, fmt.Errorf("linux is supported only in WSL for now")
	case "darwin":
		return driverdarwin.Driver{}, nil
	case "windows":
		return driverwindows.Driver{}, nil
	default:
		return nil, fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}
