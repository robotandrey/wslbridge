package cli

import (
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

var hostRe = regexp.MustCompile(`^[a-zA-Z0-9.-]+$`)

// ValidateHostOrIP validates a hostname or IP address.
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

// ValidatePort validates a TCP/UDP port number.
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

// ValidateIP validates an IP address string.
func ValidateIP(s string) error {
	s = strings.TrimSpace(s)
	if net.ParseIP(s) == nil {
		return fmt.Errorf("must be a valid IP address")
	}
	return nil
}

// ValidateURL validates a URL string.
func ValidateURL(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return fmt.Errorf("must not be empty")
	}
	u, err := url.ParseRequestURI(s)
	if err != nil {
		return fmt.Errorf("must be a valid URL")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("scheme must be http or https")
	}
	if u.Host == "" {
		return fmt.Errorf("host must not be empty")
	}
	return nil
}
