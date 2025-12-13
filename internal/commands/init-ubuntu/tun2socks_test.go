package init_ubuntu

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"wslbridge/internal/config"
	appruntime "wslbridge/internal/runtime"
)

type fakeRunnerTun struct {
	out string
	err error
}

func (f fakeRunnerTun) Run(name string, args ...string) error                  { return f.err }
func (f fakeRunnerTun) RunCapture(name string, args ...string) (string, error) { return f.out, f.err }

func TestIsExecutable(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "bin")
	if err := os.WriteFile(bin, []byte("test"), 0o755); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if !isExecutable(bin) {
		t.Fatalf("expected executable")
	}
	non := filepath.Join(dir, "non")
	if err := os.WriteFile(non, []byte("test"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if isExecutable(non) {
		t.Fatalf("expected non executable")
	}
}

func TestStartTun2Socks(t *testing.T) {
	rt := appruntime.Runtime{Runner: fakeRunnerTun{out: "1234\n"}}
	cfg := config.Config{}
	cfg.Tun.Dev = "tun0"
	cfg.Socks.Host = "1.2.3.4"
	cfg.Socks.Port = 1080

	pid, err := startTun2Socks(rt, "/bin/tun2socks", cfg)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if pid != 1234 {
		t.Fatalf("expected pid 1234, got %d", pid)
	}
}

func TestStartTun2SocksBadOutput(t *testing.T) {
	rt := appruntime.Runtime{Runner: fakeRunnerTun{out: "not-a-number\n"}}
	cfg := config.Config{Tun: struct {
		Dev  string "yaml:\"dev\""
		CIDR string "yaml:\"cidr\""
	}{Dev: "tun0"}, Socks: struct {
		Host string "yaml:\"host\""
		Port int    "yaml:\"port\""
	}{Host: "1.2.3.4", Port: 1080}}

	if _, err := startTun2Socks(rt, "/bin/tun2socks", cfg); err == nil {
		t.Fatalf("expected error for bad pid output")
	}
}

func TestTun2SocksIsRunningWithPidFile(t *testing.T) {
	dir := t.TempDir()
	pidFile := filepath.Join(dir, "pid")
	// current process pid should exist
	pid := os.Getpid()
	if err := os.WriteFile(pidFile, []byte(fmt.Sprintf("%d\n", pid)), 0o644); err != nil {
		t.Fatalf("write pid: %v", err)
	}
	if !tun2socksIsRunning(pidFile) {
		t.Fatalf("expected running pid")
	}
}

func TestStopTun2SocksIfRunningMissingFile(t *testing.T) {
	rt := appruntime.Runtime{Runner: fakeRunnerTun{}}
	if err := stopTun2SocksIfRunning(rt, filepath.Join(t.TempDir(), "pid")); err != nil {
		t.Fatalf("expected nil on missing file, got %v", err)
	}
}
