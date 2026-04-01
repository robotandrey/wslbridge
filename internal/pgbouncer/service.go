package pgbouncer

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"wslbridge/internal/cli"
	"wslbridge/internal/config"
	"wslbridge/internal/env"
	appruntime "wslbridge/internal/runtime"
)

const (
	defaultWardenScheme = "http"
	defaultEndpointMask = "/endpoints?service=<db>.pg:bouncer"
	defaultLocalHost    = "127.0.0.1"
	defaultLocalPort    = 15432
	defaultPreferRole   = "master"
)

// Service manages Warden-driven local DB proxy flow.
type Service struct {
	rt appruntime.Runtime
}

// NewService builds a Service.
func NewService(rt appruntime.Runtime) Service {
	return Service{rt: rt}
}

// Init configures Warden base URL/mask and local proxy settings.
func (s Service) Init(force bool) error {
	if err := s.checkSupported(); err != nil {
		return err
	}

	cfg, hasCfg, err := s.loadConfig()
	if err != nil {
		return err
	}
	s.applyDefaults(&cfg)

	if !force && hasCfg && strings.TrimSpace(cfg.PGBouncer.WardenHost) != "" {
		fmt.Println("db warden is already configured:", wardenCurrent(&cfg))
		fmt.Println("db endpoint mask:", cfg.PGBouncer.EndpointMask)
		fmt.Printf("db local address: %s:%d\n", cfg.PGBouncer.LocalHost, cfg.PGBouncer.LocalPort)
		fmt.Println("use `db init --force` to reconfigure")
		return nil
	}

	pr := cli.NewPrompter(os.Stdin, os.Stdout)
	wardenInput, err := pr.AskString("Warden URL (host or full endpoint URL)", "", wardenCurrent(&cfg), validateWardenInput)
	if err != nil {
		return err
	}

	scheme, host, err := NormalizeWardenInput(wardenInput)
	if err != nil {
		return err
	}
	cfg.PGBouncer.WardenScheme = scheme
	cfg.PGBouncer.WardenHost = host

	if mask, ok, err := ExtractEndpointMaskFromWardenInput(wardenInput); err != nil {
		return err
	} else if ok {
		cfg.PGBouncer.EndpointMask = mask
		fmt.Println("db endpoint mask derived:", mask)
	} else {
		maskInput, err := pr.AskString(
			"Warden endpoint mask",
			defaultEndpointMask,
			cfg.PGBouncer.EndpointMask,
			validateEndpointMask,
		)
		if err != nil {
			return err
		}
		normalizedMask, err := NormalizeEndpointMask(maskInput)
		if err != nil {
			return err
		}
		cfg.PGBouncer.EndpointMask = normalizedMask
	}

	portStr, err := pr.AskString(
		"Local proxy port",
		strconv.Itoa(defaultLocalPort),
		strconv.Itoa(cfg.PGBouncer.LocalPort),
		cli.ValidatePort,
	)
	if err != nil {
		return err
	}
	port, _ := strconv.Atoi(strings.TrimSpace(portStr))
	cfg.PGBouncer.LocalPort = port

	role, err := pr.AskString(
		"Preferred endpoint role (master/sync/async/any)",
		defaultPreferRole,
		cfg.PGBouncer.PreferRole,
		validateRole,
	)
	if err != nil {
		return err
	}
	cfg.PGBouncer.PreferRole = strings.ToLower(strings.TrimSpace(role))

	cfg.PGBouncer.WardenURL = ""
	if err := config.Save(s.rt.Paths.ConfigPath, cfg); err != nil {
		return err
	}

	fmt.Println("db warden configured:", wardenCurrent(&cfg))
	fmt.Println("db endpoint mask:", cfg.PGBouncer.EndpointMask)
	fmt.Printf("db local address: %s:%d\n", cfg.PGBouncer.LocalHost, cfg.PGBouncer.LocalPort)
	return nil
}

// Start resolves all configured services and starts local proxy for them.
func (s Service) Start(force bool) error {
	if err := s.checkSupported(); err != nil {
		return err
	}

	cfg, hasCfg, err := s.loadConfig()
	if err != nil {
		return err
	}
	s.applyDefaults(&cfg)
	if err := ensureWardenConfigured(cfg); err != nil {
		return fmt.Errorf("%w (run `db init` first)", err)
	}
	if len(cfg.PGBouncer.ServiceNames) == 0 {
		return fmt.Errorf("no database services configured (use `db add <service>`)")
	}

	if force {
		if err := s.promptRuntimeConfig(&cfg, hasCfg); err != nil {
			return err
		}
	}

	for _, service := range cfg.PGBouncer.ServiceNames {
		endpointURL, ep, err := s.resolveEndpoint(cfg, service)
		if err != nil {
			return fmt.Errorf("service %q validation via warden failed: %w", service, err)
		}
		if err := CheckTCPConnectivity(ep.Address, defaultConnectivityTimeout); err != nil {
			return fmt.Errorf("service %q endpoint is unreachable (%s): %w", service, ep.Address, err)
		}

		setServiceValue(&cfg.PGBouncer.ServiceTargets, service, ep.Address)
		setServiceValue(&cfg.PGBouncer.ServiceInstances, service, ep.InstanceName)
		fmt.Printf("service %s -> %s (%s)\n", service, ep.Address, endpointURL)
	}

	if strings.TrimSpace(cfg.PGBouncer.ServiceName) == "" {
		cfg.PGBouncer.ServiceName = cfg.PGBouncer.ServiceNames[0]
	}
	cfg.PGBouncer.TargetAddress = getServiceValue(cfg.PGBouncer.ServiceTargets, cfg.PGBouncer.ServiceName)
	cfg.PGBouncer.TargetInstance = getServiceValue(cfg.PGBouncer.ServiceInstances, cfg.PGBouncer.ServiceName)
	cfg.PGBouncer.WardenURL = ""

	if err := config.Save(s.rt.Paths.ConfigPath, cfg); err != nil {
		return err
	}
	if err := s.writeProxyRoutesFile(cfg); err != nil {
		return err
	}
	if err := s.ensureProxyRunning(cfg); err != nil {
		return err
	}

	listenAddr := fmt.Sprintf("%s:%d", cfg.PGBouncer.LocalHost, cfg.PGBouncer.LocalPort)
	fmt.Println("db local address:", listenAddr)
	fmt.Println("db services:", servicesLabel(cfg.PGBouncer.ServiceNames))
	fmt.Printf("jdbc url template: jdbc:postgresql://%s/%s\n", listenAddr, "<database>")
	fmt.Println("ssl mode note: local proxy answers `N` to PostgreSQL SSLRequest, use disable/prefer instead of require")
	return nil
}

// AddService appends a database service name, validates endpoint, and makes it available immediately.
func (s Service) AddService(serviceArg string) error {
	if err := s.checkSupported(); err != nil {
		return err
	}

	cfg, _, err := s.loadConfig()
	if err != nil {
		return err
	}
	s.applyDefaults(&cfg)
	if err := ensureWardenConfigured(cfg); err != nil {
		return fmt.Errorf("%w (run `db init` first)", err)
	}

	service := strings.TrimSpace(serviceArg)
	if service == "" {
		pr := cli.NewPrompter(os.Stdin, os.Stdout)
		prompted, err := pr.AskString("Database service name", "", cfg.PGBouncer.ServiceName, validateServiceName)
		if err != nil {
			return err
		}
		service = strings.TrimSpace(prompted)
	}
	if err := validateServiceName(service); err != nil {
		return err
	}

	endpointURL, ep, err := s.resolveEndpoint(cfg, service)
	if err != nil {
		return fmt.Errorf("service %q validation via warden failed: %w", service, err)
	}
	if err := CheckTCPConnectivity(ep.Address, defaultConnectivityTimeout); err != nil {
		return fmt.Errorf("service %q endpoint is unreachable (%s): %w", service, ep.Address, err)
	}

	cfg.PGBouncer.ServiceName = service
	cfg.PGBouncer.ServiceNames = upsertServiceName(cfg.PGBouncer.ServiceNames, service)
	setServiceValue(&cfg.PGBouncer.ServiceTargets, service, ep.Address)
	setServiceValue(&cfg.PGBouncer.ServiceInstances, service, ep.InstanceName)
	cfg.PGBouncer.TargetAddress = ep.Address
	cfg.PGBouncer.TargetInstance = ep.InstanceName
	cfg.PGBouncer.WardenURL = ""

	if err := config.Save(s.rt.Paths.ConfigPath, cfg); err != nil {
		return err
	}
	if err := s.writeProxyRoutesFile(cfg); err != nil {
		return err
	}
	if err := s.ensureProxyRunning(cfg); err != nil {
		return err
	}

	listenAddr := fmt.Sprintf("%s:%d", cfg.PGBouncer.LocalHost, cfg.PGBouncer.LocalPort)
	fmt.Println("db service added:", service)
	fmt.Println("db endpoint url:", endpointURL)
	fmt.Println("db selected endpoint:", ep.Address)
	if ep.InstanceName != "" {
		fmt.Println("db selected instance:", ep.InstanceName)
	}
	fmt.Println("db endpoint connectivity: ok")
	fmt.Println("db local address:", listenAddr)
	fmt.Printf("jdbc url: jdbc:postgresql://%s/%s\n", listenAddr, service)
	fmt.Println("db services:", servicesLabel(cfg.PGBouncer.ServiceNames))
	return nil
}

// RemoveService removes a database service and refreshes/stops local proxy.
func (s Service) RemoveService(serviceArg string) error {
	if err := s.checkSupported(); err != nil {
		return err
	}

	service := strings.TrimSpace(serviceArg)
	if service == "" {
		return fmt.Errorf("service name is required (use: db remove <service>)")
	}
	if err := validateServiceName(service); err != nil {
		return err
	}

	cfg, _, err := s.loadConfig()
	if err != nil {
		return err
	}
	s.applyDefaults(&cfg)

	updated, removed := removeServiceName(cfg.PGBouncer.ServiceNames, service)
	if !removed {
		fmt.Println("db service not found:", service)
		return nil
	}
	cfg.PGBouncer.ServiceNames = updated
	deleteServiceValue(cfg.PGBouncer.ServiceTargets, service)
	deleteServiceValue(cfg.PGBouncer.ServiceInstances, service)

	if strings.EqualFold(cfg.PGBouncer.ServiceName, service) {
		cfg.PGBouncer.ServiceName = ""
		cfg.PGBouncer.TargetAddress = ""
		cfg.PGBouncer.TargetInstance = ""
		if len(cfg.PGBouncer.ServiceNames) > 0 {
			cfg.PGBouncer.ServiceName = cfg.PGBouncer.ServiceNames[0]
			cfg.PGBouncer.TargetAddress = getServiceValue(cfg.PGBouncer.ServiceTargets, cfg.PGBouncer.ServiceName)
			cfg.PGBouncer.TargetInstance = getServiceValue(cfg.PGBouncer.ServiceInstances, cfg.PGBouncer.ServiceName)
		}
	}

	if err := config.Save(s.rt.Paths.ConfigPath, cfg); err != nil {
		return err
	}

	if len(cfg.PGBouncer.ServiceNames) == 0 {
		if err := StopProxyDaemon(DefaultProxyFiles(s.rt)); err != nil {
			return err
		}
		_ = os.Remove(s.proxyRoutesPath())
	} else {
		if err := s.writeProxyRoutesFile(cfg); err != nil {
			return err
		}
		if IsProxyRunning(s.rt.Paths.DBProxyPIDFile) {
			if err := s.ensureProxyRunning(cfg); err != nil {
				return err
			}
		}
	}

	fmt.Println("db service removed:", service)
	fmt.Println("db services:", servicesLabel(cfg.PGBouncer.ServiceNames))
	return nil
}

// Stop stops local proxy daemon.
func (s Service) Stop() error {
	if err := s.checkSupported(); err != nil {
		return err
	}
	if !IsProxyRunning(s.rt.Paths.DBProxyPIDFile) {
		fmt.Println("db proxy is not running")
		_ = os.Remove(s.rt.Paths.DBProxyPIDFile)
		_ = os.Remove(s.rt.Paths.DBProxyMetaFile)
		return nil
	}
	if err := StopProxyDaemon(DefaultProxyFiles(s.rt)); err != nil {
		return err
	}
	fmt.Println("db proxy stopped")
	return nil
}

// Status prints current configuration and daemon state.
func (s Service) Status() error {
	if err := s.checkSupported(); err != nil {
		return err
	}

	cfg, _, err := s.loadConfig()
	if err != nil {
		return err
	}
	s.applyDefaults(&cfg)

	fmt.Println("Config:", s.rt.Paths.ConfigPath)
	fmt.Println("Warden scheme:", emptyIf(cfg.PGBouncer.WardenScheme))
	fmt.Println("Warden host:", emptyIf(cfg.PGBouncer.WardenHost))
	fmt.Println("Endpoint mask:", emptyIf(cfg.PGBouncer.EndpointMask))
	fmt.Println("Preferred role:", emptyIf(cfg.PGBouncer.PreferRole))
	fmt.Printf("Local address: %s:%d\n", cfg.PGBouncer.LocalHost, cfg.PGBouncer.LocalPort)
	fmt.Println("Active service:", emptyIf(cfg.PGBouncer.ServiceName))
	fmt.Println("Services:", servicesLabel(cfg.PGBouncer.ServiceNames))

	if len(cfg.PGBouncer.ServiceNames) > 0 {
		fmt.Println("Service endpoints:")
		for _, service := range cfg.PGBouncer.ServiceNames {
			target := getServiceValue(cfg.PGBouncer.ServiceTargets, service)
			instance := getServiceValue(cfg.PGBouncer.ServiceInstances, service)
			fmt.Printf("- %s -> %s", service, emptyIf(target))
			if instance != "" {
				fmt.Printf(" (%s)", instance)
			}
			fmt.Println()
		}
	}

	running := IsProxyRunning(s.rt.Paths.DBProxyPIDFile)
	fmt.Println("Proxy running:", boolLabel(running))
	if pid, ok := readPID(s.rt.Paths.DBProxyPIDFile); ok {
		if running {
			fmt.Println("Proxy pid:", pid)
		} else {
			fmt.Println("Proxy pid:", pid, "(stale)")
		}
	} else {
		fmt.Println("Proxy pid:", "(not found)")
	}

	if meta, ok := readProxyMeta(s.rt.Paths.DBProxyMetaFile); ok {
		fmt.Println("Proxy listen addr:", emptyIf(meta.ListenAddr))
		fmt.Println("Proxy routes file:", emptyIf(meta.RoutesFile))
		fmt.Println("Proxy services:", servicesLabel(meta.Services))
		fmt.Println("Proxy started at:", emptyIf(meta.StartedAt))
	} else {
		fmt.Println("Proxy listen addr:", fmt.Sprintf("%s:%d", cfg.PGBouncer.LocalHost, cfg.PGBouncer.LocalPort))
		fmt.Println("Proxy routes file:", s.proxyRoutesPath())
	}

	fmt.Println("Proxy pid file:", s.rt.Paths.DBProxyPIDFile)
	fmt.Println("Proxy meta file:", s.rt.Paths.DBProxyMetaFile)
	fmt.Println("Proxy log:", s.rt.Paths.DBProxyLogFile)
	fmt.Printf("JDBC template: jdbc:postgresql://%s:%d/%s\n", cfg.PGBouncer.LocalHost, cfg.PGBouncer.LocalPort, "<database>")
	fmt.Println("SSL mode note: local proxy answers `N` to PostgreSQL SSLRequest")
	return nil
}

func (s Service) checkSupported() error {
	if runtime.GOOS != "linux" || !env.IsWSL() {
		return fmt.Errorf("db command is supported only in Ubuntu/WSL")
	}
	return nil
}

func (s Service) loadConfig() (config.Config, bool, error) {
	c, err := config.Load(s.rt.Paths.ConfigPath)
	if err == nil {
		return c, true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return config.Config{}, false, nil
	}
	return config.Config{}, false, err
}

func (s Service) applyDefaults(cfg *config.Config) {
	s.backfillFromLegacy(cfg)

	cfg.PGBouncer.ServiceName = strings.TrimSpace(cfg.PGBouncer.ServiceName)
	cfg.PGBouncer.ServiceNames = normalizeServiceNames(cfg.PGBouncer.ServiceNames)
	if cfg.PGBouncer.ServiceName != "" {
		cfg.PGBouncer.ServiceNames = upsertServiceName(cfg.PGBouncer.ServiceNames, cfg.PGBouncer.ServiceName)
	}
	if cfg.PGBouncer.ServiceName == "" && len(cfg.PGBouncer.ServiceNames) > 0 {
		cfg.PGBouncer.ServiceName = cfg.PGBouncer.ServiceNames[0]
	}

	cfg.PGBouncer.ServiceTargets = normalizeServiceValues(cfg.PGBouncer.ServiceNames, cfg.PGBouncer.ServiceTargets)
	cfg.PGBouncer.ServiceInstances = normalizeServiceValues(cfg.PGBouncer.ServiceNames, cfg.PGBouncer.ServiceInstances)

	if cfg.PGBouncer.ServiceName != "" {
		if strings.TrimSpace(cfg.PGBouncer.TargetAddress) != "" && getServiceValue(cfg.PGBouncer.ServiceTargets, cfg.PGBouncer.ServiceName) == "" {
			setServiceValue(&cfg.PGBouncer.ServiceTargets, cfg.PGBouncer.ServiceName, cfg.PGBouncer.TargetAddress)
		}
		if strings.TrimSpace(cfg.PGBouncer.TargetInstance) != "" && getServiceValue(cfg.PGBouncer.ServiceInstances, cfg.PGBouncer.ServiceName) == "" {
			setServiceValue(&cfg.PGBouncer.ServiceInstances, cfg.PGBouncer.ServiceName, cfg.PGBouncer.TargetInstance)
		}
		if cfg.PGBouncer.TargetAddress == "" {
			cfg.PGBouncer.TargetAddress = getServiceValue(cfg.PGBouncer.ServiceTargets, cfg.PGBouncer.ServiceName)
		}
		if cfg.PGBouncer.TargetInstance == "" {
			cfg.PGBouncer.TargetInstance = getServiceValue(cfg.PGBouncer.ServiceInstances, cfg.PGBouncer.ServiceName)
		}
	}

	if strings.TrimSpace(cfg.PGBouncer.WardenScheme) == "" {
		cfg.PGBouncer.WardenScheme = defaultWardenScheme
	}
	if strings.TrimSpace(cfg.PGBouncer.EndpointMask) == "" {
		cfg.PGBouncer.EndpointMask = defaultEndpointMask
	}
	if strings.TrimSpace(cfg.PGBouncer.LocalHost) == "" {
		cfg.PGBouncer.LocalHost = defaultLocalHost
	}
	if cfg.PGBouncer.LocalPort == 0 {
		cfg.PGBouncer.LocalPort = defaultLocalPort
	}
	if strings.TrimSpace(cfg.PGBouncer.PreferRole) == "" {
		cfg.PGBouncer.PreferRole = defaultPreferRole
	}
}

func (s Service) promptRuntimeConfig(cfg *config.Config, hasCfg bool) error {
	pr := cli.NewPrompter(os.Stdin, os.Stdout)

	curPort := ""
	if cfg.PGBouncer.LocalPort > 0 {
		curPort = strconv.Itoa(cfg.PGBouncer.LocalPort)
	}
	portStr, err := pr.AskString("Local proxy port", strconv.Itoa(defaultLocalPort), curPort, cli.ValidatePort)
	if err != nil {
		return err
	}
	port, _ := strconv.Atoi(strings.TrimSpace(portStr))
	cfg.PGBouncer.LocalPort = port

	role, err := pr.AskString(
		"Preferred endpoint role (master/sync/async/any)",
		defaultPreferRole,
		cfg.PGBouncer.PreferRole,
		validateRole,
	)
	if err != nil {
		return err
	}
	cfg.PGBouncer.PreferRole = strings.ToLower(strings.TrimSpace(role))

	_ = hasCfg
	return nil
}

func (s Service) resolveEndpoint(cfg config.Config, serviceName string) (string, Endpoint, error) {
	endpointURL, err := BuildEndpointURL(
		cfg.PGBouncer.WardenScheme,
		cfg.PGBouncer.WardenHost,
		cfg.PGBouncer.EndpointMask,
		serviceName,
	)
	if err != nil {
		return "", Endpoint{}, err
	}

	endpoints, err := FetchEndpoints(endpointURL)
	if err != nil {
		return "", Endpoint{}, err
	}
	ep, err := ChooseEndpoint(endpoints, cfg.PGBouncer.PreferRole)
	if err != nil {
		return "", Endpoint{}, err
	}

	return endpointURL, ep, nil
}

func validateRole(s string) error {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "master", "sync", "async", "any":
		return nil
	default:
		return fmt.Errorf("must be one of: master, sync, async, any")
	}
}

func validateWardenInput(s string) error {
	_, _, err := NormalizeWardenInput(s)
	return err
}

func validateEndpointMask(s string) error {
	_, err := NormalizeEndpointMask(s)
	return err
}

func validateServiceName(s string) error {
	val := strings.TrimSpace(s)
	if val == "" {
		return fmt.Errorf("must not be empty")
	}
	if strings.ContainsAny(val, " \t\r\n") {
		return fmt.Errorf("must not contain spaces")
	}
	return nil
}

func ensureWardenConfigured(cfg config.Config) error {
	if strings.TrimSpace(cfg.PGBouncer.WardenHost) == "" {
		return fmt.Errorf("warden host is not configured")
	}
	if strings.TrimSpace(cfg.PGBouncer.EndpointMask) == "" {
		return fmt.Errorf("warden endpoint mask is not configured")
	}
	return nil
}

func (s Service) backfillFromLegacy(cfg *config.Config) {
	legacy := strings.TrimSpace(cfg.PGBouncer.WardenURL)
	if legacy == "" {
		return
	}
	if strings.TrimSpace(cfg.PGBouncer.WardenHost) != "" && strings.TrimSpace(cfg.PGBouncer.WardenScheme) != "" {
		return
	}

	scheme, host, err := NormalizeWardenInput(legacy)
	if err != nil {
		return
	}
	if strings.TrimSpace(cfg.PGBouncer.WardenScheme) == "" {
		cfg.PGBouncer.WardenScheme = scheme
	}
	if strings.TrimSpace(cfg.PGBouncer.WardenHost) == "" {
		cfg.PGBouncer.WardenHost = host
	}
}

func wardenCurrent(cfg *config.Config) string {
	host := strings.TrimSpace(cfg.PGBouncer.WardenHost)
	if host == "" {
		return strings.TrimSpace(cfg.PGBouncer.WardenURL)
	}
	scheme := strings.TrimSpace(cfg.PGBouncer.WardenScheme)
	if scheme == "" {
		scheme = defaultWardenScheme
	}
	return fmt.Sprintf("%s://%s", scheme, host)
}

func boolLabel(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}

func emptyIf(v string) string {
	if strings.TrimSpace(v) == "" {
		return "(not set)"
	}
	return v
}

func servicesLabel(serviceNames []string) string {
	list := normalizeServiceNames(serviceNames)
	if len(list) == 0 {
		return "(none)"
	}
	return strings.Join(list, ", ")
}

func normalizeServiceNames(serviceNames []string) []string {
	if len(serviceNames) == 0 {
		return nil
	}
	out := make([]string, 0, len(serviceNames))
	seen := make(map[string]struct{}, len(serviceNames))
	for _, name := range serviceNames {
		val := strings.TrimSpace(name)
		if val == "" {
			continue
		}
		key := serviceKey(val)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, val)
	}
	return out
}

func upsertServiceName(serviceNames []string, serviceName string) []string {
	list := normalizeServiceNames(serviceNames)
	target := strings.TrimSpace(serviceName)
	if target == "" {
		return list
	}
	for _, existing := range list {
		if strings.EqualFold(existing, target) {
			return list
		}
	}
	return append(list, target)
}

func removeServiceName(serviceNames []string, serviceName string) ([]string, bool) {
	list := normalizeServiceNames(serviceNames)
	target := strings.TrimSpace(serviceName)
	if target == "" {
		return list, false
	}
	out := make([]string, 0, len(list))
	removed := false
	for _, existing := range list {
		if strings.EqualFold(existing, target) {
			removed = true
			continue
		}
		out = append(out, existing)
	}
	return out, removed
}

func serviceKey(service string) string {
	return strings.ToLower(strings.TrimSpace(service))
}

func normalizeServiceValues(serviceNames []string, values map[string]string) map[string]string {
	if len(serviceNames) == 0 || len(values) == 0 {
		return nil
	}
	allowed := allowedServiceKeys(serviceNames)
	out := make(map[string]string)
	for name, value := range values {
		key := serviceKey(name)
		if key == "" {
			continue
		}
		if _, ok := allowed[key]; !ok {
			continue
		}
		val := strings.TrimSpace(value)
		if val == "" {
			continue
		}
		out[key] = val
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func allowedServiceKeys(serviceNames []string) map[string]struct{} {
	out := make(map[string]struct{}, len(serviceNames))
	for _, service := range serviceNames {
		if key := serviceKey(service); key != "" {
			out[key] = struct{}{}
		}
	}
	return out
}

func getServiceValue(values map[string]string, service string) string {
	if len(values) == 0 {
		return ""
	}
	return strings.TrimSpace(values[serviceKey(service)])
}

func setServiceValue(values *map[string]string, service, value string) {
	key := serviceKey(service)
	if key == "" {
		return
	}
	trimmed := strings.TrimSpace(value)
	if *values == nil {
		*values = make(map[string]string)
	}
	if trimmed == "" {
		delete(*values, key)
		if len(*values) == 0 {
			*values = nil
		}
		return
	}
	(*values)[key] = trimmed
}

func deleteServiceValue(values map[string]string, service string) {
	if len(values) == 0 {
		return
	}
	delete(values, serviceKey(service))
}

func (s Service) proxyRoutesPath() string {
	return filepath.Join(s.rt.Paths.StateDir, "db-routes.json")
}

func (s Service) writeProxyRoutesFile(cfg config.Config) error {
	if len(cfg.PGBouncer.ServiceNames) == 0 {
		return fmt.Errorf("no services configured")
	}
	if err := os.MkdirAll(s.rt.Paths.StateDir, 0o755); err != nil {
		return err
	}

	routes := proxyRoutesFile{
		Services: make(map[string]proxyRoute, len(cfg.PGBouncer.ServiceNames)),
	}
	for _, service := range cfg.PGBouncer.ServiceNames {
		target := getServiceValue(cfg.PGBouncer.ServiceTargets, service)
		if target == "" {
			return fmt.Errorf("target endpoint for service %q is not set", service)
		}
		routes.Services[serviceKey(service)] = proxyRoute{
			Service:    service,
			TargetAddr: target,
			Instance:   getServiceValue(cfg.PGBouncer.ServiceInstances, service),
		}
	}

	b, err := json.MarshalIndent(routes, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal proxy routes: %w", err)
	}
	if err := os.WriteFile(s.proxyRoutesPath(), b, 0o600); err != nil {
		return fmt.Errorf("write proxy routes: %w", err)
	}
	return nil
}

func (s Service) ensureProxyRunning(cfg config.Config) error {
	files := DefaultProxyFiles(s.rt)
	listenAddr := fmt.Sprintf("%s:%d", cfg.PGBouncer.LocalHost, cfg.PGBouncer.LocalPort)
	routesPath := s.proxyRoutesPath()
	meta := proxyMeta{
		ListenAddr: listenAddr,
		RoutesFile: routesPath,
		Services:   normalizeServiceNames(cfg.PGBouncer.ServiceNames),
		StartedAt:  time.Now().UTC().Format(time.RFC3339),
	}

	if IsProxyRunning(files.PIDFile) {
		pid, ok := readPID(files.PIDFile)
		if !ok {
			if err := StopProxyDaemon(files); err != nil {
				return err
			}
		} else {
			if current, ok := readProxyMeta(files.MetaFile); ok && current.ListenAddr == listenAddr && current.RoutesFile == routesPath {
				if current.StartedAt != "" {
					meta.StartedAt = current.StartedAt
				}
				return writeProxyState(files, pid, meta)
			}
			if err := StopProxyDaemon(files); err != nil {
				return err
			}
		}
	}

	pid, err := StartProxyDaemon(listenAddr, routesPath, files)
	if err != nil {
		return err
	}
	return writeProxyState(files, pid, meta)
}
