package init_ubuntu

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"wslbridge/internal/config"
	appruntime "wslbridge/internal/runtime"
)

type fakeRunner struct {
	out string
	err error
}

func (f fakeRunner) Run(name string, args ...string) error                  { return f.err }
func (f fakeRunner) RunCapture(name string, args ...string) (string, error) { return f.out, f.err }

func TestParseViaIP(t *testing.T) {
	if ip, ok := parseViaIP("default via 10.0.0.1 dev eth0"); !ok || ip != "10.0.0.1" {
		t.Fatalf("expected ip parsed, got %s %v", ip, ok)
	}
	if _, ok := parseViaIP("default dev eth0"); ok {
		t.Fatalf("expected no via ip")
	}
}

func TestInferWSLGatewayFromEth0(t *testing.T) {
	rt := appruntime.Runtime{Runner: fakeRunner{out: "172.16.0.2/24\n"}}
	ip, err := inferWSLGatewayFromEth0(rt)
	if err != nil {
		t.Fatalf("infer gateway: %v", err)
	}
	if ip != "172.16.0.1" {
		t.Fatalf("expected network+1 ip, got %s", ip)
	}
}

func TestInferWSLGatewayFromEth0Error(t *testing.T) {
	rt := appruntime.Runtime{Runner: fakeRunner{err: errors.New("boom")}}
	if _, err := inferWSLGatewayFromEth0(rt); err == nil {
		t.Fatalf("expected error")
	}
}

func TestDetectSocksGatewayOrder(t *testing.T) {
	// 1) route line
	rt := appruntime.Runtime{Runner: fakeRunner{}}
	cfg := config.Config{}
	gw, err := detectSocksGateway(rt, cfg, "default via 1.2.3.4 dev eth0")
	if err != nil || gw != "1.2.3.4" {
		t.Fatalf("expected gateway from route line, got %s err %v", gw, err)
	}

	// 2) default route file
	rt.Paths.DefaultRouteFile = filepath.Join(t.TempDir(), "route.txt")
	if err := os.WriteFile(rt.Paths.DefaultRouteFile, []byte("default via 5.6.7.8 dev eth0\n"), 0o644); err != nil {
		t.Fatalf("write route file: %v", err)
	}
	gw, err = detectSocksGateway(rt, cfg, "")
	if err != nil || gw != "5.6.7.8" {
		t.Fatalf("expected gateway from file, got %s err %v", gw, err)
	}

	// 3) config fallback
	cfg.Socks.Host = "9.9.9.9"
	_ = os.Remove(rt.Paths.DefaultRouteFile)
	gw, err = detectSocksGateway(rt, cfg, "")
	if err != nil || gw != "9.9.9.9" {
		t.Fatalf("expected gateway from config, got %s err %v", gw, err)
	}

	// 4) infer
	rt.Runner = fakeRunner{out: "10.0.0.2/24\n"}
	cfg.Socks.Host = ""
	gw, err = detectSocksGateway(rt, cfg, "")
	if err != nil || gw != "10.0.0.1" {
		t.Fatalf("expected inferred gateway, got %s err %v", gw, err)
	}
}
