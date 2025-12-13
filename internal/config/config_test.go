package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg := Config{}
	cfg.Socks.Host = "localhost"
	cfg.Socks.Port = 1080
	cfg.Tun.Dev = "tun0"
	cfg.Tun.CIDR = "10.0.0.2/24"
	cfg.DNS.Nameserver = "8.8.8.8"

	if err := Save(path, cfg); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if loaded != cfg {
		t.Fatalf("loaded config mismatch: %+v != %+v", loaded, cfg)
	}
}

func TestLoadMissingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "missing.yaml")

	if _, err := Load(path); err == nil {
		t.Fatalf("expected error for missing file")
	} else if !os.IsNotExist(err) {
		t.Fatalf("expected not-exist error, got %v", err)
	}
}
