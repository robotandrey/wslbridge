package db

import (
	"net"
	"testing"
	"time"
)

// TestCheckTCPConnectivity_OK verifies connectivity to a live local listener.
func TestCheckTCPConnectivity_OK(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen error: %v", err)
	}
	defer ln.Close()

	done := make(chan struct{})
	defer close(done)
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			_ = conn.Close()
			select {
			case <-done:
				return
			default:
			}
		}
	}()

	if err := CheckTCPConnectivity(ln.Addr().String(), time.Second); err != nil {
		t.Fatalf("CheckTCPConnectivity error: %v", err)
	}
}

// TestCheckTCPConnectivity_Fail verifies failure for an unreachable port.
func TestCheckTCPConnectivity_Fail(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen error: %v", err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()

	if err := CheckTCPConnectivity(addr, 200*time.Millisecond); err == nil {
		t.Fatalf("CheckTCPConnectivity expected error")
	}
}
