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

	pkgs := []string{
		"curl",
		"ca-certificates",
		"golang-go",
		"dnsutils",
		"netcat-openbsd",
		"socat",
		"postgresql-client",
		"iproute2",
	}

	if err := r.Run("sudo", "apt", "update"); err != nil {
		return err
	}
	args := append([]string{"apt", "install", "-y"}, pkgs...)
	return r.Run("sudo", args...)
}
