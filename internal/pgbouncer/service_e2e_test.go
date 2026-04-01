package pgbouncer

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	goruntime "runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"wslbridge/internal/config"
	"wslbridge/internal/env"
	appruntime "wslbridge/internal/runtime"
)

func TestServiceE2E_MultiServiceLifecycle(t *testing.T) {
	requireWSLSupported(t)

	rt := newE2ERuntime(t)
	svc := NewService(rt)

	chatAddr, chatStartup, stopChat := startMockPostgresTarget(t, "chatapi-upstream")
	defer stopChat()
	saturnAddr, saturnStartup, stopSaturn := startMockPostgresTarget(t, "saturn-upstream")
	defer stopSaturn()

	warden, seenQueries := startWardenStub(t, map[string]string{
		"chatapi-ng":   chatAddr,
		"bozon-saturn": saturnAddr,
	})
	defer warden.Close()

	localPort := getClosedTCPPort(t)
	initURL := warden.URL + "/endpoints?service=bootstrap.pg:bouncer"
	withStdinInput(t, fmt.Sprintf("%s\n%d\n\n", initURL, localPort), func() {
		if err := svc.Init(false); err != nil {
			t.Fatalf("Init() error: %v", err)
		}
	})

	cfg := mustLoadConfig(t, rt.Paths.ConfigPath)
	wardenURL, err := url.Parse(warden.URL)
	if err != nil {
		t.Fatalf("parse warden URL: %v", err)
	}
	if cfg.PGBouncer.WardenHost != wardenURL.Host {
		t.Fatalf("WardenHost=%q, want %q", cfg.PGBouncer.WardenHost, wardenURL.Host)
	}
	if cfg.PGBouncer.EndpointMask != "/endpoints?service=<db>.pg:bouncer" {
		t.Fatalf("EndpointMask=%q, want %q", cfg.PGBouncer.EndpointMask, "/endpoints?service=<db>.pg:bouncer")
	}
	if cfg.PGBouncer.LocalHost != defaultLocalHost {
		t.Fatalf("LocalHost=%q, want %q", cfg.PGBouncer.LocalHost, defaultLocalHost)
	}
	if cfg.PGBouncer.LocalPort != localPort {
		t.Fatalf("LocalPort=%d, want %d", cfg.PGBouncer.LocalPort, localPort)
	}

	if err := svc.AddService("chatapi-ng"); err != nil {
		t.Fatalf("AddService(chatapi-ng) error: %v", err)
	}
	if err := svc.AddService("bozon-saturn"); err != nil {
		t.Fatalf("AddService(bozon-saturn) error: %v", err)
	}

	cfg = mustLoadConfig(t, rt.Paths.ConfigPath)
	if got, want := cfg.PGBouncer.ServiceNames, []string{"chatapi-ng", "bozon-saturn"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ServiceNames=%v, want %v", got, want)
	}
	if !IsProxyRunning(rt.Paths.DBProxyPIDFile) {
		t.Fatalf("proxy process is expected to be running after add")
	}
	if err := svc.Status(); err != nil {
		t.Fatalf("Status() error: %v", err)
	}

	routes := mustLoadRoutes(t, svc.proxyRoutesPath())
	if got, want := routes.Services["chatapi-ng"].TargetAddr, chatAddr; got != want {
		t.Fatalf("chatapi route target=%q, want %q", got, want)
	}
	if got, want := routes.Services["bozon-saturn"].TargetAddr, saturnAddr; got != want {
		t.Fatalf("bozon-saturn route target=%q, want %q", got, want)
	}

	listenAddr := fmt.Sprintf("%s:%d", cfg.PGBouncer.LocalHost, cfg.PGBouncer.LocalPort)
	if msg := connectViaProxy(t, listenAddr, "chatapi-ng"); msg != "chatapi-upstream" {
		t.Fatalf("chatapi connect returned %q, want %q", msg, "chatapi-upstream")
	}
	if msg := connectViaProxy(t, listenAddr, "bozon-saturn"); msg != "saturn-upstream" {
		t.Fatalf("bozon-saturn connect returned %q, want %q", msg, "saturn-upstream")
	}
	if !chatStartup.ContainsDatabase("chatapi-ng") {
		t.Fatalf("chat upstream did not observe database chatapi-ng")
	}
	if !saturnStartup.ContainsDatabase("bozon-saturn") {
		t.Fatalf("saturn upstream did not observe database bozon-saturn")
	}

	if err := svc.Stop(); err != nil {
		t.Fatalf("Stop() error: %v", err)
	}
	if IsProxyRunning(rt.Paths.DBProxyPIDFile) {
		t.Fatalf("proxy process should be stopped")
	}

	if err := svc.Start(false); err != nil {
		t.Fatalf("Start(false) error: %v", err)
	}
	if !IsProxyRunning(rt.Paths.DBProxyPIDFile) {
		t.Fatalf("proxy process should be running after Start(false)")
	}
	if msg := connectViaProxy(t, listenAddr, "chatapi-ng"); msg != "chatapi-upstream" {
		t.Fatalf("chatapi connect after restart returned %q, want %q", msg, "chatapi-upstream")
	}

	if err := svc.RemoveService("chatapi-ng"); err != nil {
		t.Fatalf("RemoveService(chatapi-ng) error: %v", err)
	}
	cfg = mustLoadConfig(t, rt.Paths.ConfigPath)
	if got, want := cfg.PGBouncer.ServiceNames, []string{"bozon-saturn"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ServiceNames after first remove=%v, want %v", got, want)
	}
	if msg := connectViaProxy(t, listenAddr, "chatapi-ng"); !strings.Contains(msg, `database "chatapi-ng" is not configured`) {
		t.Fatalf("chatapi should be rejected after remove, got %q", msg)
	}
	if msg := connectViaProxy(t, listenAddr, "bozon-saturn"); msg != "saturn-upstream" {
		t.Fatalf("bozon-saturn connect after chatapi removal returned %q, want %q", msg, "saturn-upstream")
	}

	if err := svc.RemoveService("bozon-saturn"); err != nil {
		t.Fatalf("RemoveService(bozon-saturn) error: %v", err)
	}
	cfg = mustLoadConfig(t, rt.Paths.ConfigPath)
	if len(cfg.PGBouncer.ServiceNames) != 0 {
		t.Fatalf("ServiceNames after final remove=%v, want empty", cfg.PGBouncer.ServiceNames)
	}
	if IsProxyRunning(rt.Paths.DBProxyPIDFile) {
		t.Fatalf("proxy process should be stopped after final remove")
	}
	if _, err := os.Stat(svc.proxyRoutesPath()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("proxy routes file should be removed, stat err=%v", err)
	}

	if !seenQueries.Contains("chatapi-ng.pg:bouncer") {
		t.Fatalf("warden query for chatapi-ng was not observed")
	}
	if !seenQueries.Contains("bozon-saturn.pg:bouncer") {
		t.Fatalf("warden query for bozon-saturn was not observed")
	}
}

func TestServiceE2E_AddServiceRejectsUnreachableEndpoint(t *testing.T) {
	requireWSLSupported(t)

	rt := newE2ERuntime(t)
	svc := NewService(rt)

	unreachableAddr := getClosedTCPAddress(t)
	warden, _ := startWardenStub(t, map[string]string{
		"*": unreachableAddr,
	})
	defer warden.Close()

	localPort := getClosedTCPPort(t)
	initURL := warden.URL + "/endpoints?service=bootstrap.pg:bouncer"
	withStdinInput(t, fmt.Sprintf("%s\n%d\n\n", initURL, localPort), func() {
		if err := svc.Init(false); err != nil {
			t.Fatalf("Init() error: %v", err)
		}
	})

	err := svc.AddService("chatapi-ng")
	if err == nil {
		t.Fatalf("AddService(chatapi-ng) expected connectivity error")
	}
	if !strings.Contains(err.Error(), "endpoint is unreachable") {
		t.Fatalf("AddService(chatapi-ng) error=%q, want contains %q", err.Error(), "endpoint is unreachable")
	}

	cfg := mustLoadConfig(t, rt.Paths.ConfigPath)
	if len(cfg.PGBouncer.ServiceNames) != 0 {
		t.Fatalf("ServiceNames=%v, want empty after failed add", cfg.PGBouncer.ServiceNames)
	}
	if IsProxyRunning(rt.Paths.DBProxyPIDFile) {
		t.Fatalf("proxy process must not start on failed add")
	}
	if _, statErr := os.Stat(svc.proxyRoutesPath()); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("proxy routes file should not be created, stat err=%v", statErr)
	}
}

func TestServiceE2E_CancelRequestIsForwardedToMatchedUpstream(t *testing.T) {
	requireWSLSupported(t)

	rt := newE2ERuntime(t)
	svc := NewService(rt)

	targetAddr, cancelRecorder, stopTarget := startCancelableMockPostgresTarget(t, 41001, 77123)
	defer stopTarget()

	warden, _ := startWardenStub(t, map[string]string{
		"chatapi-ng": targetAddr,
	})
	defer warden.Close()

	localPort := getClosedTCPPort(t)
	initURL := warden.URL + "/endpoints?service=bootstrap.pg:bouncer"
	withStdinInput(t, fmt.Sprintf("%s\n%d\n\n", initURL, localPort), func() {
		if err := svc.Init(false); err != nil {
			t.Fatalf("Init() error: %v", err)
		}
	})
	if err := svc.AddService("chatapi-ng"); err != nil {
		t.Fatalf("AddService(chatapi-ng) error: %v", err)
	}

	listenAddr := fmt.Sprintf("%s:%d", defaultLocalHost, localPort)
	sessionConn, processID, secretKey := startProxySessionAndReadBackendKey(t, listenAddr, "chatapi-ng")
	defer sessionConn.Close()

	sendCancelRequestViaProxy(t, listenAddr, processID, secretKey)

	if !cancelRecorder.WaitFor(processID, secretKey, 3*time.Second) {
		t.Fatalf("cancel request with pid=%d secret=%d was not forwarded to upstream", processID, secretKey)
	}
}

func requireWSLSupported(t *testing.T) {
	t.Helper()
	if goruntime.GOOS != "linux" || !env.IsWSL() {
		t.Skip("e2e tests require linux WSL environment")
	}
}

func newE2ERuntime(t *testing.T) appruntime.Runtime {
	t.Helper()

	root := t.TempDir()
	configPath := filepath.Join(root, "config", "config.yaml")
	stateDir := filepath.Join(root, "state")

	return appruntime.Runtime{
		Paths: appruntime.Paths{
			ConfigPath:      configPath,
			StateDir:        stateDir,
			DBProxyPIDFile:  filepath.Join(stateDir, "db-proxy.pid"),
			DBProxyMetaFile: filepath.Join(stateDir, "db-proxy.json"),
			DBProxyLogFile:  filepath.Join(stateDir, "db-proxy.log"),
		},
	}
}

func withStdinInput(t *testing.T, input string, fn func()) {
	t.Helper()

	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error: %v", err)
	}
	if _, err := w.WriteString(input); err != nil {
		_ = r.Close()
		_ = w.Close()
		t.Fatalf("write stdin input: %v", err)
	}
	_ = w.Close()

	os.Stdin = r
	defer func() {
		os.Stdin = oldStdin
		_ = r.Close()
	}()

	fn()
}

func mustLoadConfig(t *testing.T, path string) config.Config {
	t.Helper()

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("config.Load(%q) error: %v", path, err)
	}
	return cfg
}

func mustLoadRoutes(t *testing.T, path string) proxyRoutesFile {
	t.Helper()

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read routes file: %v", err)
	}
	var routes proxyRoutesFile
	if err := json.Unmarshal(b, &routes); err != nil {
		t.Fatalf("unmarshal routes file: %v", err)
	}
	return routes
}

func getClosedTCPAddress(t *testing.T) string {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error: %v", err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()
	return addr
}

func getClosedTCPPort(t *testing.T) int {
	t.Helper()

	_, portText, err := net.SplitHostPort(getClosedTCPAddress(t))
	if err != nil {
		t.Fatalf("SplitHostPort() error: %v", err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		t.Fatalf("Atoi(%q) error: %v", portText, err)
	}
	return port
}

type startupRecord struct {
	Database string
	User     string
}

type startupRecorder struct {
	mu      sync.Mutex
	records []startupRecord
}

type cancelRecord struct {
	ProcessID int32
	SecretKey int32
}

type cancelRecorder struct {
	mu      sync.Mutex
	records []cancelRecord
}

func (r *cancelRecorder) Record(processID, secretKey int32) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records = append(r.records, cancelRecord{ProcessID: processID, SecretKey: secretKey})
}

func (r *cancelRecorder) WaitFor(processID, secretKey int32, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		r.mu.Lock()
		for _, record := range r.records {
			if record.ProcessID == processID && record.SecretKey == secretKey {
				r.mu.Unlock()
				return true
			}
		}
		r.mu.Unlock()
		time.Sleep(25 * time.Millisecond)
	}
	return false
}

func (r *startupRecorder) Record(database, user string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records = append(r.records, startupRecord{Database: database, User: user})
}

func (r *startupRecorder) ContainsDatabase(database string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, record := range r.records {
		if record.Database == database {
			return true
		}
	}
	return false
}

func startMockPostgresTarget(t *testing.T, marker string) (string, *startupRecorder, func()) {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error: %v", err)
	}

	rec := &startupRecorder{}
	done := make(chan struct{})
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				select {
				case <-done:
					return
				default:
					return
				}
			}
			go func(c net.Conn) {
				defer c.Close()
				_ = c.SetDeadline(time.Now().Add(5 * time.Second))
				packet, code, err := readStartupPacket(c)
				if err != nil {
					return
				}
				req, err := parseStartupRequest(packet, code)
				if err != nil {
					writeErrorResponse(c, err.Error())
					return
				}
				rec.Record(req.Database, req.User)
				writeErrorResponse(c, marker)
			}(conn)
		}
	}()

	return ln.Addr().String(), rec, func() {
		close(done)
		_ = ln.Close()
	}
}

func startCancelableMockPostgresTarget(t *testing.T, processID, secretKey int32) (string, *cancelRecorder, func()) {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error: %v", err)
	}

	rec := &cancelRecorder{}
	done := make(chan struct{})
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				select {
				case <-done:
					return
				default:
					return
				}
			}
			go func(c net.Conn) {
				defer c.Close()
				_ = c.SetDeadline(time.Now().Add(5 * time.Second))

				packet, code, err := readStartupPacket(c)
				if err != nil {
					return
				}
				if code == pgCancelRequestCode {
					req, err := parseCancelRequest(packet)
					if err == nil {
						rec.Record(req.ProcessID, req.SecretKey)
					}
					return
				}

				if _, err := parseStartupRequest(packet, code); err != nil {
					return
				}

				if _, err := c.Write(buildBackendKeyDataMessage(processID, secretKey)); err != nil {
					return
				}

				buf := make([]byte, 1)
				for {
					if _, err := c.Read(buf); err != nil {
						return
					}
				}
			}(conn)
		}
	}()

	return ln.Addr().String(), rec, func() {
		close(done)
		_ = ln.Close()
	}
}

func connectViaProxy(t *testing.T, addr, database string) string {
	t.Helper()

	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		t.Fatalf("dial proxy %s: %v", addr, err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))

	if _, err := conn.Write(buildSSLRequestPacket()); err != nil {
		t.Fatalf("write ssl request: %v", err)
	}
	reply := make([]byte, 1)
	if _, err := conn.Read(reply); err != nil {
		t.Fatalf("read ssl reply: %v", err)
	}
	if string(reply) != "N" {
		t.Fatalf("ssl reply=%q, want %q", string(reply), "N")
	}

	if _, err := conn.Write(buildStartupPacket(database, "tester")); err != nil {
		t.Fatalf("write startup packet: %v", err)
	}

	msg, err := readBackendErrorMessage(conn)
	if err != nil {
		t.Fatalf("read backend error message: %v", err)
	}
	return msg
}

func startProxySessionAndReadBackendKey(t *testing.T, addr, database string) (net.Conn, int32, int32) {
	t.Helper()

	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		t.Fatalf("dial proxy %s: %v", addr, err)
	}
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))

	if _, err := conn.Write(buildSSLRequestPacket()); err != nil {
		_ = conn.Close()
		t.Fatalf("write ssl request: %v", err)
	}
	reply := make([]byte, 1)
	if _, err := conn.Read(reply); err != nil {
		_ = conn.Close()
		t.Fatalf("read ssl reply: %v", err)
	}
	if string(reply) != "N" {
		_ = conn.Close()
		t.Fatalf("ssl reply=%q, want %q", string(reply), "N")
	}

	if _, err := conn.Write(buildStartupPacket(database, "tester")); err != nil {
		_ = conn.Close()
		t.Fatalf("write startup packet: %v", err)
	}

	processID, secretKey, err := readBackendKeyData(conn)
	if err != nil {
		_ = conn.Close()
		t.Fatalf("read backend key data: %v", err)
	}
	return conn, processID, secretKey
}

func sendCancelRequestViaProxy(t *testing.T, addr string, processID, secretKey int32) {
	t.Helper()

	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		t.Fatalf("dial proxy for cancel %s: %v", addr, err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))

	if _, err := conn.Write(buildCancelRequestPacket(processID, secretKey)); err != nil {
		t.Fatalf("write cancel request: %v", err)
	}
}

func buildSSLRequestPacket() []byte {
	packet := make([]byte, 8)
	binary.BigEndian.PutUint32(packet[0:4], 8)
	binary.BigEndian.PutUint32(packet[4:8], pgSSLRequestCode)
	return packet
}

func buildStartupPacket(database, user string) []byte {
	payload := make([]byte, 0, 64)
	payload = append(payload, []byte("user")...)
	payload = append(payload, 0)
	payload = append(payload, []byte(user)...)
	payload = append(payload, 0)
	payload = append(payload, []byte("database")...)
	payload = append(payload, 0)
	payload = append(payload, []byte(database)...)
	payload = append(payload, 0, 0)

	packet := make([]byte, 8+len(payload))
	binary.BigEndian.PutUint32(packet[0:4], uint32(len(packet)))
	binary.BigEndian.PutUint32(packet[4:8], pgProtocolVersion3)
	copy(packet[8:], payload)
	return packet
}

func buildCancelRequestPacket(processID, secretKey int32) []byte {
	packet := make([]byte, 16)
	binary.BigEndian.PutUint32(packet[0:4], uint32(len(packet)))
	binary.BigEndian.PutUint32(packet[4:8], pgCancelRequestCode)
	binary.BigEndian.PutUint32(packet[8:12], uint32(processID))
	binary.BigEndian.PutUint32(packet[12:16], uint32(secretKey))
	return packet
}

func buildBackendKeyDataMessage(processID, secretKey int32) []byte {
	packet := make([]byte, 1+4+8)
	packet[0] = 'K'
	binary.BigEndian.PutUint32(packet[1:5], 12)
	binary.BigEndian.PutUint32(packet[5:9], uint32(processID))
	binary.BigEndian.PutUint32(packet[9:13], uint32(secretKey))
	return packet
}

func readBackendKeyData(conn net.Conn) (int32, int32, error) {
	header := make([]byte, 5)
	if _, err := ioReadFull(conn, header); err != nil {
		return 0, 0, err
	}
	if header[0] != 'K' {
		return 0, 0, fmt.Errorf("unexpected backend message type %q", string(header[:1]))
	}
	if got := binary.BigEndian.Uint32(header[1:5]); got != 12 {
		return 0, 0, fmt.Errorf("unexpected BackendKeyData length %d", got)
	}

	payload := make([]byte, 8)
	if _, err := ioReadFull(conn, payload); err != nil {
		return 0, 0, err
	}
	return int32(binary.BigEndian.Uint32(payload[0:4])), int32(binary.BigEndian.Uint32(payload[4:8])), nil
}

func readBackendErrorMessage(conn net.Conn) (string, error) {
	header := make([]byte, 5)
	if _, err := ioReadFull(conn, header); err != nil {
		return "", err
	}
	if header[0] != 'E' {
		return "", fmt.Errorf("unexpected backend message type %q", string(header[:1]))
	}

	payloadLen := binary.BigEndian.Uint32(header[1:5])
	if payloadLen < 4 {
		return "", fmt.Errorf("invalid backend payload length %d", payloadLen)
	}

	payload := make([]byte, payloadLen-4)
	if _, err := ioReadFull(conn, payload); err != nil {
		return "", err
	}

	i := 0
	for i < len(payload) {
		fieldType := payload[i]
		i++
		if fieldType == 0 {
			break
		}
		start := i
		for i < len(payload) && payload[i] != 0 {
			i++
		}
		if i >= len(payload) {
			return "", fmt.Errorf("unterminated backend error field")
		}
		value := string(payload[start:i])
		i++
		if fieldType == 'M' {
			return value, nil
		}
	}
	return "", fmt.Errorf("backend error message field not found")
}

func ioReadFull(conn net.Conn, buf []byte) (int, error) {
	total := 0
	for total < len(buf) {
		n, err := conn.Read(buf[total:])
		total += n
		if err != nil {
			return total, err
		}
	}
	return total, nil
}

type queryRecorder struct {
	mu      sync.Mutex
	queries []string
}

func (q *queryRecorder) Record(v string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.queries = append(q.queries, v)
}

func (q *queryRecorder) Contains(v string) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	for _, candidate := range q.queries {
		if candidate == v {
			return true
		}
	}
	return false
}

func startWardenStub(t *testing.T, addrByService map[string]string) (*httptest.Server, *queryRecorder) {
	t.Helper()

	rec := &queryRecorder{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serviceQuery := strings.TrimSpace(r.URL.Query().Get("service"))
		rec.Record(serviceQuery)

		serviceName := strings.TrimSuffix(serviceQuery, ".pg:bouncer")
		targetAddr := strings.TrimSpace(addrByService[serviceName])
		if targetAddr == "" {
			targetAddr = strings.TrimSpace(addrByService["*"])
		}
		if targetAddr == "" {
			http.Error(w, "unknown service", http.StatusNotFound)
			return
		}

		payload := []Endpoint{
			{
				InstanceName:   "db-" + serviceName,
				Address:        targetAddr,
				Role:           "master",
				IsDefaultRoute: true,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(payload)
	}))

	return srv, rec
}
