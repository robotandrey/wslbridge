package init_ubuntu

import (
	"testing"

	appruntime "wslbridge/internal/runtime"
)

type captureRunner struct {
	name string
	args []string
}

func (r *captureRunner) Run(name string, args ...string) error {
	r.name = name
	r.args = append([]string(nil), args...)
	return nil
}

func (r *captureRunner) RunCapture(name string, args ...string) (string, error) {
	return "", nil
}

func TestBuildRestoreDefaultRouteArgs(t *testing.T) {
	got, err := buildRestoreDefaultRouteArgs("default via 172.27.16.1 dev eth0 proto kernel metric 100 linkdown")
	if err != nil {
		t.Fatalf("buildRestoreDefaultRouteArgs() error = %v", err)
	}

	want := []string{"ip", "route", "replace", "default", "via", "172.27.16.1", "dev", "eth0", "proto", "kernel", "metric", "100"}
	if len(got) != len(want) {
		t.Fatalf("buildRestoreDefaultRouteArgs() len = %d, want %d; got=%v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("buildRestoreDefaultRouteArgs()[%d] = %q, want %q; got=%v", i, got[i], want[i], got)
		}
	}
}

func TestBuildRestoreDefaultRouteArgsRequiresDefault(t *testing.T) {
	if _, err := buildRestoreDefaultRouteArgs("via 172.27.16.1 dev eth0"); err == nil {
		t.Fatalf("buildRestoreDefaultRouteArgs() expected error for invalid route")
	}
}

func TestRestoreDefaultRouteUsesNormalizedArgs(t *testing.T) {
	runner := &captureRunner{}
	rt := appruntime.Runtime{Runner: runner}

	if err := restoreDefaultRoute(rt, "default dev eth0 scope link linkdown"); err != nil {
		t.Fatalf("restoreDefaultRoute() error = %v", err)
	}
	if runner.name != "sudo" {
		t.Fatalf("runner name = %q, want sudo", runner.name)
	}

	want := []string{"ip", "route", "replace", "default", "dev", "eth0", "scope", "link"}
	if len(runner.args) != len(want) {
		t.Fatalf("runner args len = %d, want %d; got=%v", len(runner.args), len(want), runner.args)
	}
	for i := range want {
		if runner.args[i] != want[i] {
			t.Fatalf("runner args[%d] = %q, want %q; got=%v", i, runner.args[i], want[i], runner.args)
		}
	}
}
