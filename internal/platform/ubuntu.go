package platform

import (
	"fmt"
	"os/exec"

	"wslbridge/internal/execx"
)

// Ubuntu implements dependency checks for Ubuntu/WSL.
type Ubuntu struct{}

// Name returns the platform name.
func (Ubuntu) Name() string { return "ubuntu" }

// EnsureDeps installs missing dependencies via apt.
func (Ubuntu) EnsureDeps(r execx.Runner) error {
	if _, err := exec.LookPath("apt"); err != nil {
		return fmt.Errorf("apt not found: expected ubuntu/debian")
	}

	// Команда -> пакет
	need := map[string]string{
		"curl":      "curl",
		"psql":      "postgresql-client",
		"go":        "golang-go",
		"pgbouncer": "pgbouncer",
	}

	var pkgs []string
	for bin, pkg := range need {
		if _, err := exec.LookPath(bin); err != nil {
			pkgs = append(pkgs, pkg)
		}
	}

	if len(pkgs) == 0 {
		return nil
	}

	if err := r.Run("sudo", "apt", "update"); err != nil {
		return err
	}
	args := append([]string{"apt", "install", "-y"}, pkgs...)
	return r.Run("sudo", args...)
}
