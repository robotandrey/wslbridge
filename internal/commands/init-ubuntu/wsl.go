package init_ubuntu

import (
	"fmt"
	"os"
	"path/filepath"
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
	backup := filepath.Join(rt.Paths.StateDir, "wsl.conf.bak")
	if err := backupFileOnce("/etc/wsl.conf", backup); err != nil {
		return err
	}
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
	backup := filepath.Join(rt.Paths.StateDir, "resolv.conf.bak")
	if err := backupFileOnce("/etc/resolv.conf", backup); err != nil {
		return err
	}
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

func restoreWSLConf(rt appruntime.Runtime) (bool, error) {
	backup := filepath.Join(rt.Paths.StateDir, "wsl.conf.bak")
	return restoreFileIfExists(rt, backup, "/etc/wsl.conf")
}

func restoreResolvConf(rt appruntime.Runtime) (bool, error) {
	backup := filepath.Join(rt.Paths.StateDir, "resolv.conf.bak")
	return restoreFileIfExists(rt, backup, "/etc/resolv.conf")
}

func backupFileOnce(src, backup string) error {
	if _, err := os.Stat(backup); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	b, err := os.ReadFile(src)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if err := os.MkdirAll(filepath.Dir(backup), 0o755); err != nil {
		return err
	}
	return os.WriteFile(backup, b, 0o600)
}

func restoreFileIfExists(rt appruntime.Runtime, backup, target string) (bool, error) {
	b, err := os.ReadFile(backup)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	tmp := filepath.Join(os.TempDir(), filepath.Base(target)+".wslbridge")
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return false, err
	}
	if err := rt.Runner.Run("sudo", "mv", tmp, target); err != nil {
		return false, fmt.Errorf("write %s: %w", target, err)
	}
	if err := rt.Runner.Run("sudo", "chmod", "644", target); err != nil {
		return false, err
	}
	return true, nil
}
