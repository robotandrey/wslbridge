package execx

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
)

// Runner executes external commands.
type Runner interface {
	Run(name string, args ...string) error
	RunCapture(name string, args ...string) (string, error)
}

// OSRunner runs commands via os/exec.
type OSRunner struct{}

// Run executes a command and streams stdio.
func (OSRunner) Run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

// RunCapture executes a command and returns stdout.
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
