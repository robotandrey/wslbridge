package cli

import (
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
)

var hostRe = regexp.MustCompile(`^[a-zA-Z0-9.-]+$`)

func ValidateHostOrIP(s string) error {
	s = strings.TrimSpace(s)
	if ip := net.ParseIP(s); ip != nil {
		return nil
	}
	if !hostRe.MatchString(s) {
		return fmt.Errorf("must be an IP or hostname")
	}
	if strings.HasPrefix(s, ".") || strings.HasSuffix(s, ".") || strings.Contains(s, "..") {
		return fmt.Errorf("invalid hostname format")
	}
	for _, label := range strings.Split(s, ".") {
		if label == "" || strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return fmt.Errorf("invalid hostname format")
		}
	}
	return nil
}

func ValidatePort(s string) error {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return fmt.Errorf("must be integer")
	}
	if n < 1 || n > 65535 {
		return fmt.Errorf("must be in range 1..65535")
	}
	return nil
}

func ValidateIP(s string) error {
	s = strings.TrimSpace(s)
	if net.ParseIP(s) == nil {
		return fmt.Errorf("must be a valid IP address")
	}
	return nil
}
