package platform

import (
	"runtime"
	"testing"
)

func TestDetect(t *testing.T) {
	p, err := Detect()
	if runtime.GOOS == "linux" {
		if err != nil {
			t.Fatalf("expected ubuntu platform, got error %v", err)
		}
		if p.Name() != "ubuntu" {
			t.Fatalf("unexpected platform name: %s", p.Name())
		}
	} else {
		if err == nil {
			t.Fatalf("expected error on unsupported GOOS")
		}
	}
}
