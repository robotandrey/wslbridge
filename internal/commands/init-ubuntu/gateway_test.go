package init_ubuntu

import (
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestParseViaIP(t *testing.T) {
	cases := []struct {
		line string
		want string
		ok   bool
	}{
		{"default via 172.30.112.1 dev eth0 proto dhcp metric 100", "172.30.112.1", true},
		{"default dev tun0 scope link", "", false},
		{"", "", false},
	}

	for _, tc := range cases {
		got, ok := parseViaIP(tc.line)
		if ok != tc.ok || got != tc.want {
			t.Fatalf("parseViaIP(%q) = (%q, %v), want (%q, %v)", tc.line, got, ok, tc.want, tc.ok)
		}
	}
}

func TestReadResolvConfNameserver(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "resolv.conf")
	content := "# comment\nnameserver 10.0.0.1\nnameserver 8.8.8.8\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp resolv.conf: %v", err)
	}

	got, ok := readResolvConfNameserver(path)
	if !ok || got != "10.0.0.1" {
		t.Fatalf("readResolvConfNameserver() = (%q, %v), want (%q, %v)", got, ok, "10.0.0.1", true)
	}
}

func TestIncrementIPv4(t *testing.T) {
	ip := net.IPv4(10, 0, 0, 0)
	got, err := incrementIPv4(ip)
	if err != nil {
		t.Fatalf("incrementIPv4 error: %v", err)
	}
	if got.String() != "10.0.0.1" {
		t.Fatalf("incrementIPv4 = %q, want %q", got.String(), "10.0.0.1")
	}

	_, err = incrementIPv4(net.IPv4(255, 255, 255, 255))
	if err == nil {
		t.Fatalf("incrementIPv4 overflow expected error")
	}
}
