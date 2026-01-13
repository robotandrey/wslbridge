package init_ubuntu

import "testing"

func TestDefaultIsTun(t *testing.T) {
	cases := []struct {
		line string
		dev  string
		want bool
	}{
		{"default dev tun0 scope link", "tun0", true},
		{"default via 172.30.112.1 dev tun0 proto dhcp", "tun0", true},
		{"default dev eth0 scope link", "tun0", false},
		{"", "tun0", false},
	}

	for _, tc := range cases {
		got := defaultIsTun(tc.line, tc.dev)
		if got != tc.want {
			t.Fatalf("defaultIsTun(%q, %q) = %v, want %v", tc.line, tc.dev, got, tc.want)
		}
	}
}
