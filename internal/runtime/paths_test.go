package runtime

import "testing"

func TestDefaultPaths(t *testing.T) {
	paths, err := DefaultPaths()
	if err != nil {
		t.Fatalf("default paths: %v", err)
	}
	if paths.ConfigPath == "" || paths.ShareDir == "" || paths.StateDir == "" {
		t.Fatalf("expected all paths filled: %+v", paths)
	}
}
