package execx

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
)

type Runner interface {
	Run(name string, args ...string) error
	RunCapture(name string, args ...string) (string, error)
}

type OSRunner struct{}

func (OSRunner) Run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func (OSRunner) RunCapture(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s %v: %w", name, args, err)
	}
	return out.String(), nil
}
