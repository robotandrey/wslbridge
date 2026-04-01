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
	want.PGBouncer.WardenScheme = "http"
	want.PGBouncer.WardenHost = "warden.stg.s.bozon.tech"
	want.PGBouncer.EndpointMask = "/endpoints?service=<db>.pg:bouncer"
	want.PGBouncer.AuthLookupUser = "pgbouncer_auth"
	want.PGBouncer.AuthLookupPass = "secret"
	want.PGBouncer.AuthQuery = "SELECT usename, passwd FROM pg_catalog.pg_shadow WHERE usename=$1"
	want.PGBouncer.ServiceName = "bozon-saturn"
	want.PGBouncer.ServiceNames = []string{"bozon-saturn"}
	want.PGBouncer.ServiceTargets = map[string]string{"bozon-saturn": "10.0.0.1:6432"}
	want.PGBouncer.LocalHost = "127.0.0.1"
	want.PGBouncer.LocalPort = 15432
	want.PGBouncer.PreferRole = "master"

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
