package commands

import "testing"

// TestAllCommandsMetadata validates exported top-level CLI command metadata.
func TestAllCommandsMetadata(t *testing.T) {
	cmds := All()
	if len(cmds) != 4 {
		t.Fatalf("All() returned %d commands, want 4", len(cmds))
	}

	want := map[string]string{
		"init":   "Initialize wslbridge for the current OS/environment",
		"status": "Show wslbridge status (current OS/environment)",
		"stop":   "Stop wslbridge and restore routes (current OS/environment)",
		"db":     "Manage Warden-driven local DB proxy (init|start|status|stop|add|remove)",
	}

	for _, c := range cmds {
		help, ok := want[c.Name()]
		if !ok {
			t.Fatalf("unexpected command %q", c.Name())
		}
		if c.Help() != help {
			t.Fatalf("command %q help mismatch: got %q, want %q", c.Name(), c.Help(), help)
		}
		delete(want, c.Name())
	}

	if len(want) != 0 {
		t.Fatalf("missing commands: %v", want)
	}
}
