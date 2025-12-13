package util

import (
	"errors"
	"testing"

	appruntime "wslbridge/internal/runtime"
)

type fakeRunner struct {
	out string
	err error
}

func (f fakeRunner) Run(name string, args ...string) error                  { return f.err }
func (f fakeRunner) RunCapture(name string, args ...string) (string, error) { return f.out, f.err }

func TestDetectDefaultGateway(t *testing.T) {
	rt := appruntime.Runtime{Runner: fakeRunner{out: "default via 172.20.0.1 dev eth0"}}

	gw, err := DetectDefaultGateway(rt)
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if gw != "172.20.0.1" {
		t.Fatalf("expected gateway, got %s", gw)
	}
}

func TestDetectDefaultGatewayError(t *testing.T) {
	rt := appruntime.Runtime{Runner: fakeRunner{err: errors.New("fail")}}
	if _, err := DetectDefaultGateway(rt); err == nil {
		t.Fatalf("expected error")
	}
}
