package config

import (
	"path/filepath"
	"reflect"
	"testing"
)

// TestSaveLoad validates that configs round-trip through disk.
func TestSaveLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	var want Config
	want.Socks.Host = "127.0.0.1"
	want.Socks.Port = 1080
	want.Tun.Dev = "tun0"
	want.Tun.CIDR = "10.0.0.2/24"
	want.DNS.Nameserver = "8.8.8.8"

	if err := Save(path, want); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if !reflect.DeepEqual(want, got) {
		t.Fatalf("Load mismatch: want %+v, got %+v", want, got)
	}
}
