package cli

import "testing"

// TestValidateHostOrIP validates host or IP inputs.
func TestValidateHostOrIP(t *testing.T) {
	cases := []struct {
		val  string
		want bool
	}{
		{"1.2.3.4", true},
		{"example.com", true},
		{"-bad", false},
		{".bad", false},
		{"a..b", false},
		{"bad host", false},
	}

	for _, tc := range cases {
		err := ValidateHostOrIP(tc.val)
		if (err == nil) != tc.want {
			t.Fatalf("ValidateHostOrIP(%q) err=%v, want ok=%v", tc.val, err, tc.want)
		}
	}
}

// TestValidatePort validates port inputs.
func TestValidatePort(t *testing.T) {
	cases := []struct {
		val  string
		want bool
	}{
		{"80", true},
		{"1", true},
		{"0", false},
		{"70000", false},
		{"abc", false},
	}

	for _, tc := range cases {
		err := ValidatePort(tc.val)
		if (err == nil) != tc.want {
			t.Fatalf("ValidatePort(%q) err=%v, want ok=%v", tc.val, err, tc.want)
		}
	}
}

// TestValidateIP validates IP inputs.
func TestValidateIP(t *testing.T) {
	cases := []struct {
		val  string
		want bool
	}{
		{"8.8.8.8", true},
		{"999.1.1.1", false},
		{"example.com", false},
	}

	for _, tc := range cases {
		err := ValidateIP(tc.val)
		if (err == nil) != tc.want {
			t.Fatalf("ValidateIP(%q) err=%v, want ok=%v", tc.val, err, tc.want)
		}
	}
}

// TestValidateURL validates URL inputs.
func TestValidateURL(t *testing.T) {
	cases := []struct {
		val  string
		want bool
	}{
		{"http://service-discovery.example.internal/endpoints?service=example-db.pg:bouncer", true},
		{"https://example.org/path", true},
		{"ftp://example.org/path", false},
		{"not-a-url", false},
		{"", false},
	}

	for _, tc := range cases {
		err := ValidateURL(tc.val)
		if (err == nil) != tc.want {
			t.Fatalf("ValidateURL(%q) err=%v, want ok=%v", tc.val, err, tc.want)
		}
	}
}
