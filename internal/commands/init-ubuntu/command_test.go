package init_ubuntu

import "testing"

func TestParseFlags(t *testing.T) {
	f, err := parseFlags([]string{"--skip-deps", "--force", "--socks-port=1234"})
	if err != nil {
		t.Fatalf("parse flags: %v", err)
	}
	if !f.skipDeps || !f.force || f.socksPortOverride != 1234 {
		t.Fatalf("unexpected flags: %+v", f)
	}

	if _, err := parseFlags([]string{"--socks-port=bad"}); err == nil {
		t.Fatalf("expected error for bad port")
	}
	if _, err := parseFlags([]string{"--unknown"}); err == nil {
		t.Fatalf("expected error for unknown flag")
	}
}
