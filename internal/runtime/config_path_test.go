package runtime

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindProjectRootStopsAtMarkers(t *testing.T) {
	dir := t.TempDir()

	// Create /tmp/.../a/b/.git
	nested := filepath.Join(dir, "a", "b")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	gitDir := filepath.Join(dir, "a", ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatalf("mkdir git: %v", err)
	}

	root, err := FindProjectRoot(nested)
	if err != nil {
		t.Fatalf("find root: %v", err)
	}
	if root != filepath.Join(dir, "a") {
		t.Fatalf("expected project root %s, got %s", filepath.Join(dir, "a"), root)
	}
}

func TestFindProjectRootFallsBackToStart(t *testing.T) {
	dir := t.TempDir()
	start := filepath.Join(dir, "x", "y")
	if err := os.MkdirAll(start, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	root, err := FindProjectRoot(start)
	if err != nil {
		t.Fatalf("find root: %v", err)
	}
	if root != "/" {
		t.Fatalf("expected filesystem root fallback, got %s", root)
	}
}

func TestResolveProjectLocalConfigPath(t *testing.T) {
	dir := t.TempDir()
	// simulate cwd with go.mod marker
	goModDir := filepath.Join(dir, "project")
	if err := os.MkdirAll(goModDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(goModDir, "go.mod"), []byte("module dummy\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })

	if err := os.Chdir(filepath.Join(goModDir, "nested")); err != nil {
		if err := os.MkdirAll(filepath.Join(goModDir, "nested"), 0o755); err != nil {
			t.Fatalf("mkdir nested: %v", err)
		}
		if err := os.Chdir(filepath.Join(goModDir, "nested")); err != nil {
			t.Fatalf("chdir nested: %v", err)
		}
	} else {
		// nested already existed (unlikely), ensure directory
	}

	path, err := ResolveProjectLocalConfigPath()
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	expected := filepath.Join(goModDir, LocalConfigDir, LocalConfigFile)
	if path != expected {
		t.Fatalf("expected %s, got %s", expected, path)
	}
}
