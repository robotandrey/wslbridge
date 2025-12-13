package cli

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestPrompterUsesDefaultAndValidation(t *testing.T) {
	input := bytes.NewBufferString("\ninvalid\ncustom\n")
	output := &bytes.Buffer{}

	p := NewPrompter(input, output)
	called := 0
	value, err := p.AskString("Question", "invalid-default", "", func(s string) error {
		called++
		if strings.HasPrefix(s, "invalid") {
			return fmt.Errorf("bad")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("ask string: %v", err)
	}
	if value != "custom" {
		t.Fatalf("expected custom value, got %s", value)
	}
	if called != 3 { // default(invalid) -> invalid -> custom
		t.Fatalf("expected validation called three times, got %d", called)
	}
}

func TestPrompterUsesCurrentWhenEmpty(t *testing.T) {
	input := bytes.NewBufferString("\n")
	output := &bytes.Buffer{}

	p := NewPrompter(input, output)
	value, err := p.AskString("Question", "def", "current", nil)
	if err != nil {
		t.Fatalf("ask string: %v", err)
	}
	if value != "current" {
		t.Fatalf("expected current value, got %s", value)
	}
}
