package db

import (
	"fmt"
	"net"
	"strings"
	"time"
)

const defaultConnectivityTimeout = 3 * time.Second

// CheckTCPConnectivity verifies that TCP endpoint is reachable.
func CheckTCPConnectivity(addr string, timeout time.Duration) error {
	target := strings.TrimSpace(addr)
	if target == "" {
		return fmt.Errorf("endpoint address must not be empty")
	}
	if timeout <= 0 {
		timeout = defaultConnectivityTimeout
	}

	conn, err := net.DialTimeout("tcp", target, timeout)
	if err != nil {
		return fmt.Errorf("tcp connectivity check failed for %s: %w", target, err)
	}
	_ = conn.Close()
	return nil
}
