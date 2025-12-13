package runtime

import (
	"testing"

	"wslbridge/internal/execx"
)

type stubPlatform struct{}

func (stubPlatform) Name() string                  { return "stub" }
func (stubPlatform) EnsureDeps(execx.Runner) error { return nil }

func TestNewSetsConfigPath(t *testing.T) {
	rt, err := New(execx.OSRunner{}, stubPlatform{})
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	if rt.Paths.ConfigPath == "" {
		t.Fatalf("expected config path set")
	}
}

func TestRuntimeStructHoldsPlatform(t *testing.T) {
	rt := Runtime{Platform: stubPlatform{}}
	if rt.Platform.Name() != "stub" {
		t.Fatalf("expected stub platform")
	}
}
