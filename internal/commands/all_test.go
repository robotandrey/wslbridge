package commands

import "testing"

func TestAllIncludesInit(t *testing.T) {
	cmds := All()
	found := false
	for _, c := range cmds {
		if c.Name() == "init" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected init command present")
	}
}
