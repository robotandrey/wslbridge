package command

import (
	"reflect"
	"testing"

	appruntime "wslbridge/internal/runtime"
)

type stubCmd struct {
	name string
	help string
}

func (s stubCmd) Name() string                               { return s.name }
func (s stubCmd) Help() string                               { return s.help }
func (s stubCmd) Run(_ appruntime.Runtime, _ []string) error { return nil }

func TestRegistryGet(t *testing.T) {
	c1 := stubCmd{name: "one", help: "first"}
	c2 := stubCmd{name: "two", help: "second"}

	reg := New(c1, c2)

	if c, ok := reg.Get("two"); !ok || c.Name() != "two" {
		t.Fatalf("expected to get command 'two'")
	}
	if _, ok := reg.Get("missing"); ok {
		t.Fatalf("expected missing command")
	}
}

func TestRegistryHelpLinesSorted(t *testing.T) {
	reg := New(
		stubCmd{name: "zeta", help: "zzz"},
		stubCmd{name: "alpha", help: "aaa"},
	)

	lines := reg.HelpLines()
	expected := []string{
		"  alpha            aaa",
		"  zeta             zzz",
	}

	if !reflect.DeepEqual(lines, expected) {
		t.Fatalf("unexpected help lines: %v", lines)
	}
}
