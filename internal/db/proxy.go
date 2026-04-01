package db

import (
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	appruntime "wslbridge/internal/runtime"
)

const (
	pgProtocolVersion3  = 196608
	pgCancelRequestCode = 80877102
	pgSSLRequestCode    = 80877103
	pgGSSENCRequestCode = 80877104
	maxStartupPacketLen = 64 * 1024
)

// HiddenProxyRunCommand is an internal command name for running proxy daemon.
const HiddenProxyRunCommand = "_db_proxy_run"

var cancelRegistry = newCancelRegistry()

func init() {
	if len(os.Args) > 1 && os.Args[1] == HiddenProxyRunCommand {
		if err := RunProxyProcess(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		os.Exit(0)
	}
}

type proxyMeta struct {
	ListenAddr string   `json:"listen_addr"`
	RoutesFile string   `json:"routes_file"`
	Services   []string `json:"services,omitempty"`
	StartedAt  string   `json:"started_at"`
}

type proxyRoute struct {
	Service    string `json:"service"`
	TargetAddr string `json:"target_addr"`
	Instance   string `json:"instance,omitempty"`
}

type proxyRoutesFile struct {
	Services map[string]proxyRoute `json:"services"`
}

type startupRequest struct {
	Packet   []byte
	Database string
	User     string
}

type cancelRequest struct {
	Packet    []byte
	ProcessID int32
	SecretKey int32
}

type cancelRegistryKey struct {
	ProcessID int32
	SecretKey int32
}

type cancelRegistryEntry struct {
	TargetAddr string
}

type cancelRegistryStore struct {
	mu      sync.RWMutex
	entries map[cancelRegistryKey]cancelRegistryEntry
}

// ProxyFiles groups runtime state files for a proxy instance.
type ProxyFiles struct {
	PIDFile  string
	MetaFile string
	LogFile  string
}

// DefaultProxyFiles returns legacy singleton proxy files.
func DefaultProxyFiles(rt appruntime.Runtime) ProxyFiles {
	return ProxyFiles{
		PIDFile:  rt.Paths.DBProxyPIDFile,
		MetaFile: rt.Paths.DBProxyMetaFile,
		LogFile:  rt.Paths.DBProxyLogFile,
	}
}

// RunProxyProcess starts a foreground TCP proxy process.
func RunProxyProcess(args []string) error {
	fs := flag.NewFlagSet(HiddenProxyRunCommand, flag.ContinueOnError)
	listenAddr := fs.String("listen", "", "listen address")
	routesFile := fs.String("routes-file", "", "routes file")
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*listenAddr) == "" || strings.TrimSpace(*routesFile) == "" {
		return fmt.Errorf("both --listen and --routes-file are required")
	}
	return runTCPProxy(*listenAddr, *routesFile)
}

// StartProxyDaemon starts a detached proxy process and returns its pid.
func StartProxyDaemon(listenAddr, routesFile string, files ProxyFiles) (int, error) {
	exe, err := os.Executable()
	if err != nil {
		return 0, fmt.Errorf("resolve executable: %w", err)
	}

	if err := ensureProxyFiles(files); err != nil {
		return 0, err
	}
	logf, err := os.OpenFile(files.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return 0, fmt.Errorf("open proxy log: %w", err)
	}
	defer logf.Close()

	cmd := exec.Command("nohup", exe, HiddenProxyRunCommand, "--listen="+listenAddr, "--routes-file="+routesFile)
	cmd.Stdout = logf
	cmd.Stderr = logf
	cmd.Stdin = nil

	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("start proxy daemon: %w", err)
	}
	pid := cmd.Process.Pid
	if !waitPID(pid, 2*time.Second) {
		return 0, fmt.Errorf("proxy daemon did not stay alive")
	}
	if !waitListenReady(listenAddr, 2*time.Second) {
		return 0, fmt.Errorf("proxy daemon did not start listening on %s", listenAddr)
	}
	return pid, nil
}

// IsProxyRunning returns whether proxy process from pid-file is running.
func IsProxyRunning(pidFile string) bool {
	pid, ok := readPID(pidFile)
	if !ok {
		return false
	}
	return isPIDRunning(pid)
}

// StopProxyDaemon stops proxy process if running.
func StopProxyDaemon(files ProxyFiles) error {
	pid, ok := readPID(files.PIDFile)
	if !ok {
		cleanupProxyState(files)
		return nil
	}

	_ = exec.Command("kill", strconv.Itoa(pid)).Run()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !isPIDRunning(pid) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if isPIDRunning(pid) {
		_ = exec.Command("kill", "-9", strconv.Itoa(pid)).Run()
	}

	cleanupProxyState(files)
	return nil
}

func writeProxyState(files ProxyFiles, pid int, meta proxyMeta) error {
	if err := ensureProxyFiles(files); err != nil {
		return err
	}
	if err := os.WriteFile(files.PIDFile, []byte(fmt.Sprintf("%d\n", pid)), 0o644); err != nil {
		return err
	}
	b, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(files.MetaFile, b, 0o644)
}

func readProxyMeta(path string) (proxyMeta, bool) {
	b, err := os.ReadFile(path)
	if err != nil {
		return proxyMeta{}, false
	}
	var m proxyMeta
	if err := json.Unmarshal(b, &m); err != nil {
		return proxyMeta{}, false
	}
	return m, true
}

func runTCPProxy(listenAddr, routesFile string) error {
	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", listenAddr, err)
	}
	defer ln.Close()

	for {
		clientConn, err := ln.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				time.Sleep(50 * time.Millisecond)
				continue
			}
			return err
		}
		go proxyConn(clientConn, routesFile)
	}
}

func proxyConn(clientConn net.Conn, routesFile string) {
	defer clientConn.Close()

	routes, err := loadProxyRoutes(routesFile)
	if err != nil {
		writeErrorResponse(clientConn, "wslbridge proxy routes are not available")
		return
	}

	req, route, cancelReq, isCancel, err := readClientRequest(clientConn, routes)
	if err != nil {
		if isClientDisconnectError(err) {
			return
		}
		writeErrorResponse(clientConn, err.Error())
		return
	}
	if isCancel {
		relayCancelRequest(cancelReq)
		return
	}

	serverConn, err := net.Dial("tcp", route.TargetAddr)
	if err != nil {
		writeErrorResponse(clientConn, fmt.Sprintf("upstream %s is unreachable", route.TargetAddr))
		return
	}
	defer serverConn.Close()

	if _, err := serverConn.Write(req.Packet); err != nil {
		writeErrorResponse(clientConn, fmt.Sprintf("failed to reach upstream for database %s", req.Database))
		return
	}

	done := make(chan struct{}, 2)
	var cancelKey *cancelRegistryKey

	go func() {
		_, _ = io.Copy(serverConn, clientConn)
		if tcp, ok := serverConn.(*net.TCPConn); ok {
			_ = tcp.CloseWrite()
		}
		done <- struct{}{}
	}()
	go func() {
		_, _ = relayServerToClient(clientConn, serverConn, route.TargetAddr, &cancelKey)
		if tcp, ok := clientConn.(*net.TCPConn); ok {
			_ = tcp.CloseWrite()
		}
		done <- struct{}{}
	}()

	<-done
	<-done
	if cancelKey != nil {
		cancelRegistry.Delete(*cancelKey)
	}
}

func loadProxyRoutes(path string) (proxyRoutesFile, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return proxyRoutesFile{}, err
	}
	var routes proxyRoutesFile
	if err := json.Unmarshal(b, &routes); err != nil {
		return proxyRoutesFile{}, err
	}
	if len(routes.Services) == 0 {
		return proxyRoutesFile{}, fmt.Errorf("proxy route map is empty")
	}
	return routes, nil
}

func readClientRequest(conn net.Conn, routes proxyRoutesFile) (startupRequest, proxyRoute, cancelRequest, bool, error) {
	if err := conn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return startupRequest{}, proxyRoute{}, cancelRequest{}, false, err
	}
	defer func() {
		_ = conn.SetReadDeadline(time.Time{})
	}()

	for attempts := 0; attempts < 4; attempts++ {
		packet, code, err := readStartupPacket(conn)
		if err != nil {
			return startupRequest{}, proxyRoute{}, cancelRequest{}, false, err
		}

		switch code {
		case pgSSLRequestCode, pgGSSENCRequestCode:
			if _, err := conn.Write([]byte("N")); err != nil {
				return startupRequest{}, proxyRoute{}, cancelRequest{}, false, err
			}
			continue
		case pgCancelRequestCode:
			req, err := parseCancelRequest(packet)
			return startupRequest{}, proxyRoute{}, req, true, err
		}

		req, err := parseStartupRequest(packet, code)
		if err != nil {
			return startupRequest{}, proxyRoute{}, cancelRequest{}, false, err
		}

		route, err := findProxyRoute(routes, req.Database, req.User)
		if err != nil {
			return startupRequest{}, proxyRoute{}, cancelRequest{}, false, err
		}
		return req, route, cancelRequest{}, false, nil
	}

	return startupRequest{}, proxyRoute{}, cancelRequest{}, false, fmt.Errorf("too many pre-startup negotiation packets")
}

func readStartupPacket(conn net.Conn) ([]byte, uint32, error) {
	header := make([]byte, 4)
	if _, err := io.ReadFull(conn, header); err != nil {
		return nil, 0, err
	}
	packetLen := binary.BigEndian.Uint32(header)
	if packetLen < 8 {
		return nil, 0, fmt.Errorf("invalid startup packet length: %d", packetLen)
	}
	if packetLen > maxStartupPacketLen {
		return nil, 0, fmt.Errorf("startup packet is too large: %d", packetLen)
	}

	body := make([]byte, packetLen-4)
	if _, err := io.ReadFull(conn, body); err != nil {
		return nil, 0, err
	}
	code := binary.BigEndian.Uint32(body[:4])
	packet := append(header, body...)
	return packet, code, nil
}

func parseStartupRequest(packet []byte, code uint32) (startupRequest, error) {
	if code != pgProtocolVersion3 {
		return startupRequest{}, fmt.Errorf("unsupported postgres startup version: %d", code)
	}
	params, err := parseStartupParams(packet[8:])
	if err != nil {
		return startupRequest{}, err
	}

	user := strings.TrimSpace(params["user"])
	database := strings.TrimSpace(params["database"])
	if database == "" {
		database = user
	}
	if database == "" {
		return startupRequest{}, fmt.Errorf("postgres startup packet does not contain database")
	}

	return startupRequest{
		Packet:   packet,
		Database: database,
		User:     user,
	}, nil
}

func parseCancelRequest(packet []byte) (cancelRequest, error) {
	if len(packet) != 16 {
		return cancelRequest{}, fmt.Errorf("invalid postgres cancel request length: %d", len(packet))
	}
	return cancelRequest{
		Packet:    packet,
		ProcessID: int32(binary.BigEndian.Uint32(packet[8:12])),
		SecretKey: int32(binary.BigEndian.Uint32(packet[12:16])),
	}, nil
}

func parseStartupParams(body []byte) (map[string]string, error) {
	parts := strings.Split(string(body), "\x00")
	if len(parts) < 3 || parts[len(parts)-1] != "" {
		return nil, fmt.Errorf("invalid postgres startup parameters")
	}
	out := make(map[string]string)
	for i := 0; i+1 < len(parts)-1; i += 2 {
		key := strings.TrimSpace(parts[i])
		value := parts[i+1]
		if key == "" {
			break
		}
		out[key] = value
	}
	return out, nil
}

func findProxyRoute(routes proxyRoutesFile, database, user string) (proxyRoute, error) {
	key := serviceKey(database)
	if key == "" && user != "" {
		key = serviceKey(user)
	}
	if key == "" {
		return proxyRoute{}, fmt.Errorf("database is required to select an upstream")
	}

	route, ok := routes.Services[key]
	if !ok || strings.TrimSpace(route.TargetAddr) == "" {
		return proxyRoute{}, fmt.Errorf("database %q is not configured in wslbridge proxy", database)
	}
	if strings.TrimSpace(route.Service) == "" {
		route.Service = database
	}
	return route, nil
}

func relayCancelRequest(req cancelRequest) {
	targetAddr, ok := cancelRegistry.Get(cancelRegistryKey{
		ProcessID: req.ProcessID,
		SecretKey: req.SecretKey,
	})
	if !ok {
		return
	}

	conn, err := net.DialTimeout("tcp", targetAddr, 3*time.Second)
	if err != nil {
		return
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
	_, _ = conn.Write(req.Packet)
}

func relayServerToClient(clientConn, serverConn net.Conn, targetAddr string, cancelKey **cancelRegistryKey) (int64, error) {
	var total int64
	header := make([]byte, 5)

	for {
		if _, err := io.ReadFull(serverConn, header); err != nil {
			if err == io.EOF {
				return total, nil
			}
			return total, err
		}

		msgType := header[0]
		msgLen := binary.BigEndian.Uint32(header[1:5])
		if msgLen < 4 {
			return total, fmt.Errorf("invalid postgres backend message length: %d", msgLen)
		}

		payload := make([]byte, msgLen-4)
		if _, err := io.ReadFull(serverConn, payload); err != nil {
			return total, err
		}

		if msgType == 'K' && len(payload) == 8 {
			key := cancelRegistryKey{
				ProcessID: int32(binary.BigEndian.Uint32(payload[0:4])),
				SecretKey: int32(binary.BigEndian.Uint32(payload[4:8])),
			}
			cancelRegistry.Put(key, cancelRegistryEntry{TargetAddr: targetAddr})
			*cancelKey = &key
		}

		if _, err := clientConn.Write(header); err != nil {
			return total, err
		}
		if _, err := clientConn.Write(payload); err != nil {
			return total, err
		}
		total += int64(len(header) + len(payload))
	}
}

func writeErrorResponse(conn net.Conn, message string) {
	message = strings.TrimSpace(message)
	if message == "" {
		message = "wslbridge proxy rejected startup packet"
	}

	payload := []byte{'S'}
	payload = append(payload, []byte("FATAL")...)
	payload = append(payload, 0)
	payload = append(payload, 'M')
	payload = append(payload, []byte(message)...)
	payload = append(payload, 0, 0)

	packet := make([]byte, 1+4+len(payload))
	packet[0] = 'E'
	binary.BigEndian.PutUint32(packet[1:5], uint32(len(payload)+4))
	copy(packet[5:], payload)
	_, _ = conn.Write(packet)
}

func readPID(path string) (int, bool) {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil || pid <= 0 {
		return 0, false
	}
	return pid, true
}

func isPIDRunning(pid int) bool {
	if pid <= 0 {
		return false
	}
	return exec.Command("kill", "-0", strconv.Itoa(pid)).Run() == nil
}

func waitPID(pid int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if isPIDRunning(pid) {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}

func waitListenReady(listenAddr string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", listenAddr, 100*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}

func ensureProxyFiles(files ProxyFiles) error {
	if strings.TrimSpace(files.PIDFile) == "" || strings.TrimSpace(files.MetaFile) == "" || strings.TrimSpace(files.LogFile) == "" {
		return fmt.Errorf("proxy state files are not configured")
	}

	dirs := []string{
		filepath.Dir(files.PIDFile),
		filepath.Dir(files.MetaFile),
		filepath.Dir(files.LogFile),
	}
	for _, d := range dirs {
		if strings.TrimSpace(d) == "" || d == "." {
			continue
		}
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}
	return nil
}

func cleanupProxyState(files ProxyFiles) {
	_ = os.Remove(files.PIDFile)
	_ = os.Remove(files.MetaFile)
}

func isClientDisconnectError(err error) bool {
	return err == io.EOF || err == io.ErrUnexpectedEOF || strings.Contains(strings.ToLower(err.Error()), "closed network connection")
}

func newCancelRegistry() *cancelRegistryStore {
	return &cancelRegistryStore{
		entries: make(map[cancelRegistryKey]cancelRegistryEntry),
	}
}

func (r *cancelRegistryStore) Put(key cancelRegistryKey, entry cancelRegistryEntry) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries[key] = entry
}

func (r *cancelRegistryStore) Get(key cancelRegistryKey) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	entry, ok := r.entries[key]
	if !ok || strings.TrimSpace(entry.TargetAddr) == "" {
		return "", false
	}
	return entry.TargetAddr, true
}

func (r *cancelRegistryStore) Delete(key cancelRegistryKey) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.entries, key)
}
