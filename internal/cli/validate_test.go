package cli

import "testing"

func TestValidateHostOrIP(t *testing.T) {
	valid := []string{"127.0.0.1", "example.com", "my-host"}
	for _, v := range valid {
		if err := ValidateHostOrIP(v); err != nil {
			t.Fatalf("expected valid host %s: %v", v, err)
		}
	}

	invalid := []string{"", "..bad", "bad..host", "host!"}
	for _, v := range invalid {
		if err := ValidateHostOrIP(v); err == nil {
			t.Fatalf("expected error for %q", v)
		}
	}
}

func TestValidatePort(t *testing.T) {
	if err := ValidatePort("1080"); err != nil {
		t.Fatalf("port valid: %v", err)
	}
	cases := []string{"0", "70000", "abc"}
	for _, c := range cases {
		if err := ValidatePort(c); err == nil {
			t.Fatalf("expected error for %s", c)
		}
	}
}

func TestValidateIP(t *testing.T) {
	if err := ValidateIP("192.168.1.1"); err != nil {
		t.Fatalf("ip valid: %v", err)
	}
	if err := ValidateIP("not-an-ip"); err == nil {
		t.Fatalf("expected error for invalid ip")
	}
}
