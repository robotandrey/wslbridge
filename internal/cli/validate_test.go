package cli

import "testing"

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
