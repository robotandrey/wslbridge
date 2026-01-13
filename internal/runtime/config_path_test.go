package runtime

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindProjectRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	nested := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}

	got, err := FindProjectRoot(nested)
	if err != nil {
		t.Fatalf("FindProjectRoot error: %v", err)
	}
	if got != root {
		t.Fatalf("FindProjectRoot(%q) = %q, want %q", nested, got, root)
	}
}

func TestResolveProjectLocalConfigPath(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	nested := filepath.Join(root, "x", "y")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer func() {
		_ = os.Chdir(cwd)
	}()
	if err := os.Chdir(nested); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	got, err := ResolveProjectLocalConfigPath()
	if err != nil {
		t.Fatalf("ResolveProjectLocalConfigPath error: %v", err)
	}
	want := filepath.Join(root, LocalConfigDir, LocalConfigFile)
	if got != want {
		t.Fatalf("ResolveProjectLocalConfigPath = %q, want %q", got, want)
	}
}

func TestResolveConfigPath_UsesLocalForWslbridgeRepo(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module wslbridge"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	nested := filepath.Join(root, "x", "y")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer func() {
		_ = os.Chdir(cwd)
	}()
	if err := os.Chdir(nested); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	got, err := ResolveConfigPath()
	if err != nil {
		t.Fatalf("ResolveConfigPath error: %v", err)
	}
	want := filepath.Join(root, LocalConfigDir, LocalConfigFile)
	if got != want {
		t.Fatalf("ResolveConfigPath = %q, want %q", got, want)
	}
}

func TestResolveConfigPath_FallsBackToUserConfig(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatalf("mkdir home: %v", err)
	}
	t.Setenv("HOME", home)
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module other"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	nested := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer func() {
		_ = os.Chdir(cwd)
	}()
	if err := os.Chdir(nested); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	got, err := ResolveConfigPath()
	if err != nil {
		t.Fatalf("ResolveConfigPath error: %v", err)
	}
	paths, err := DefaultPaths()
	if err != nil {
		t.Fatalf("DefaultPaths error: %v", err)
	}
	if got != paths.ConfigPath {
		t.Fatalf("ResolveConfigPath = %q, want %q", got, paths.ConfigPath)
	}
}
