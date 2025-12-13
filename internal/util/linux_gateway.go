package util

import (
	"fmt"
	"strings"
	appruntime "wslbridge/internal/runtime"
)

// DetectDefaultGateway возвращает IP из "ip route show default"
func DetectDefaultGateway(rt appruntime.Runtime) (string, error) {
	out, err := rt.Runner.RunCapture("ip", "route", "show", "default")
	if err != nil {
		return "", fmt.Errorf("ip route show default failed: %w", err)
	}
	
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		for i := 0; i < len(fields)-1; i++ {
			if fields[i] == "via" {
				return fields[i+1], nil
			}
		}
	}

	return "", fmt.Errorf("could not detect default gateway from: %q", out)
}
