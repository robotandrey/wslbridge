package env

import (
	"os"
	"strings"
)

// IsWSL reports whether the current linux environment looks like WSL.
func IsWSL() bool {
	b, err := os.ReadFile("/proc/version")
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(b)), "microsoft")
}
