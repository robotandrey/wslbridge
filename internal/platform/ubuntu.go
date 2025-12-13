package platform

import (
	"fmt"
	"os/exec"

	"wslbridge/internal/execx"
)

type Ubuntu struct{}

func (Ubuntu) Name() string { return "ubuntu" }

func (Ubuntu) EnsureDeps(r execx.Runner) error {
	if _, err := exec.LookPath("apt"); err != nil {
		return fmt.Errorf("apt not found: expected ubuntu/debian")
	}

	// Команда -> пакет
	need := map[string]string{
		"curl":  "curl",
		"dig":   "dnsutils",
		"nc":    "netcat-openbsd",
		"socat": "socat",
		"psql":  "postgresql-client",
		"ip":    "iproute2",
		"go":    "golang-go",
	}

	var pkgs []string
	for bin, pkg := range need {
		if _, err := exec.LookPath(bin); err != nil {
			pkgs = append(pkgs, pkg)
		}
	}

	// Всё есть — вообще ничего не делаем.
	if len(pkgs) == 0 {
		return nil
	}

	if err := r.Run("sudo", "apt", "update"); err != nil {
		return err
	}
	args := append([]string{"apt", "install", "-y"}, pkgs...)
	return r.Run("sudo", args...)
}
