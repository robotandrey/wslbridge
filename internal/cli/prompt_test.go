package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

// TestAskString_Default validates default selection on empty input.
func TestAskString_Default(t *testing.T) {
	in := strings.NewReader("\n")
	out := &bytes.Buffer{}
	p := NewPrompter(in, out)
	got, err := p.AskString("Label", "def", "", nil)
	if err != nil {
		t.Fatalf("AskString error: %v", err)
	}
	if got != "def" {
		t.Fatalf("AskString = %q, want %q", got, "def")
	}
}

// TestAskString_Current validates current value selection.
func TestAskString_Current(t *testing.T) {
	in := strings.NewReader("\n")
	out := &bytes.Buffer{}
	p := NewPrompter(in, out)
	got, err := p.AskString("Label", "def", "cur", nil)
	if err != nil {
		t.Fatalf("AskString error: %v", err)
	}
	if got != "cur" {
		t.Fatalf("AskString = %q, want %q", got, "cur")
	}
}

// TestAskString_Validate validates retry behavior on invalid input.
func TestAskString_Validate(t *testing.T) {
	in := strings.NewReader("bad\nok\n")
	out := &bytes.Buffer{}
	p := NewPrompter(in, out)
	validate := func(s string) error {
		if s == "bad" {
			return errors.New("invalid")
		}
		return nil
	}
	got, err := p.AskString("Label", "def", "", validate)
	if err != nil {
		t.Fatalf("AskString error: %v", err)
	}
	if got != "ok" {
		t.Fatalf("AskString = %q, want %q", got, "ok")
	}
}
