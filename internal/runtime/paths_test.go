package runtime

import (
	"path/filepath"
	"testing"
)

func TestDefaultPathsUseUserScopedTun2SocksLog(t *testing.T) {
	paths, err := DefaultPaths()
	if err != nil {
		t.Fatalf("DefaultPaths() error = %v", err)
	}

	want := filepath.Join(paths.StateDir, "tun2socks.log")
	if paths.Tun2SocksLogFile != want {
		t.Fatalf("Tun2SocksLogFile = %q, want %q", paths.Tun2SocksLogFile, want)
	}
}
