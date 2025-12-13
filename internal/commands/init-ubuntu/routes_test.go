package init_ubuntu

import (
	"errors"
	"testing"

	appruntime "wslbridge/internal/runtime"
)

type fakeRunnerRoute struct {
	out string
	err error
}

func (f fakeRunnerRoute) Run(name string, args ...string) error                  { return f.err }
func (f fakeRunnerRoute) RunCapture(name string, args ...string) (string, error) { return f.out, f.err }

func TestGetDefaultRouteLine(t *testing.T) {
	rt := appruntime.Runtime{Runner: fakeRunnerRoute{out: "default via 10.0.0.1 dev eth0\n"}}
	line, err := getDefaultRouteLine(rt)
	if err != nil {
		t.Fatalf("get route: %v", err)
	}
	if line != "default via 10.0.0.1 dev eth0" {
		t.Fatalf("unexpected line: %q", line)
	}
}

func TestGetDefaultRouteLineError(t *testing.T) {
	rt := appruntime.Runtime{Runner: fakeRunnerRoute{err: errors.New("boom")}}
	if _, err := getDefaultRouteLine(rt); err == nil {
		t.Fatalf("expected error")
	}
}

func TestDefaultIsTun(t *testing.T) {
	if !defaultIsTun("default via 10.0.0.1 dev tun0", "tun0") {
		t.Fatalf("expected tun route detected")
	}
	if defaultIsTun("default via 10.0.0.1 dev eth0", "tun0") {
		t.Fatalf("unexpected match")
	}
}
