package command

import (
	"strings"
	"testing"

	appruntime "wslbridge/internal/runtime"
)

type stubCommand struct {
	name string
	help string
}

func (s stubCommand) Name() string { return s.name }
func (s stubCommand) Help() string { return s.help }
func (s stubCommand) Run(rt appruntime.Runtime, args []string) error {
	return nil
}

// TestRegistryGet validates registry lookup by name.
func TestRegistryGet(t *testing.T) {
	reg := New(stubCommand{name: "a", help: "A"})
	if _, ok := reg.Get("a"); !ok {
		t.Fatalf("expected to find command")
	}
	if _, ok := reg.Get("missing"); ok {
		t.Fatalf("expected missing command")
	}
}

// TestRegistryHelpLines validates help line ordering.
func TestRegistryHelpLines(t *testing.T) {
	reg := New(
		stubCommand{name: "b", help: "B"},
		stubCommand{name: "a", help: "A"},
	)
	lines := reg.HelpLines()
	if len(lines) != 2 {
		t.Fatalf("expected 2 help lines, got %d", len(lines))
	}
	if lines[0] == lines[1] || lines[0] == "" || lines[1] == "" {
		t.Fatalf("unexpected help lines: %v", lines)
	}
	if !strings.Contains(lines[0], "a") || !strings.Contains(lines[1], "b") {
		t.Fatalf("expected sorted help lines, got %v", lines)
	}
}
