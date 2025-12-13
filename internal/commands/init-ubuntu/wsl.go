package init_ubuntu

import (
	"fmt"
	"os"
	"strings"

	appruntime "wslbridge/internal/runtime"
)

const defaultNameServer = "8.8.8.8"

func IsWSL() bool {
	b, err := os.ReadFile("/proc/version")
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(b)), "microsoft")
}

func configureWSLConf(rt appruntime.Runtime) error {
	const content = `[network]
generateHosts = false
generateResolvConf = false
`
	tmp := "/tmp/wsl.conf"
	if err := os.WriteFile(tmp, []byte(content), 0o644); err != nil {
		return err
	}
	if err := rt.Runner.Run("sudo", "mv", tmp, "/etc/wsl.conf"); err != nil {
		return fmt.Errorf("write /etc/wsl.conf: %w", err)
	}
	return rt.Runner.Run("sudo", "chmod", "644", "/etc/wsl.conf")
}

func writeResolvConf(rt appruntime.Runtime, nameserver string) error {
	content := fmt.Sprintf("nameserver %s\n", strings.TrimSpace(nameserver)) + fmt.Sprintf("nameserver %s\n", defaultNameServer)

	tmp := "/tmp/resolv.conf"
	if err := os.WriteFile(tmp, []byte(content), 0o644); err != nil {
		return err
	}
	if err := rt.Runner.Run("sudo", "mv", tmp, "/etc/resolv.conf"); err != nil {
		return fmt.Errorf("write /etc/resolv.conf: %w", err)
	}
	return rt.Runner.Run("sudo", "chmod", "644", "/etc/resolv.conf")
}
