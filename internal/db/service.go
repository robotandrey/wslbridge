package db

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
	defaultServiceDiscoveryScheme = "http"
	defaultEndpointMask           = "/endpoints?service=<db>.pg:bouncer"
	defaultLocalHost              = "127.0.0.1"
	defaultLocalPort              = 15432
	defaultPreferRole             = "master"
)

// Service manages service-discovery-driven local DB proxy flow.
type Service struct {
	rt appruntime.Runtime
}

// NewService builds a Service.
func NewService(rt appruntime.Runtime) Service {
	return Service{rt: rt}
}

// Init configures Service discovery base URL/mask and local proxy settings.
func (s Service) Init(force bool) error {
	if err := s.checkSupported(); err != nil {
		return err
	}

	cfg, hasCfg, err := s.loadConfig()
	if err != nil {
		return err
	}
	s.applyDefaults(&cfg)

	if !force && hasCfg && strings.TrimSpace(cfg.DB.ServiceDiscoveryHost) != "" {
		fmt.Println("db service discovery is already configured:", serviceDiscoveryCurrent(&cfg))
		fmt.Println("db endpoint mask:", cfg.DB.EndpointMask)
		fmt.Printf("db local address: %s:%d\n", cfg.DB.LocalHost, cfg.DB.LocalPort)
		fmt.Println("use `db init --force` to reconfigure")
		return nil
	}

	pr := cli.NewPrompter(os.Stdin, os.Stdout)
	serviceDiscoveryInput, err := pr.AskString("Service discovery URL (host or full endpoint URL)", "", serviceDiscoveryCurrent(&cfg), validateServiceDiscoveryInput)
	if err != nil {
		return err
	}

	scheme, host, err := NormalizeServiceDiscoveryInput(serviceDiscoveryInput)
	if err != nil {
		return err
	}
	cfg.DB.ServiceDiscoveryScheme = scheme
	cfg.DB.ServiceDiscoveryHost = host

	if mask, ok, err := ExtractEndpointMaskFromServiceDiscoveryInput(serviceDiscoveryInput); err != nil {
		return err
	} else if ok {
		cfg.DB.EndpointMask = mask
		fmt.Println("db endpoint mask derived:", mask)
	} else {
		maskInput, err := pr.AskString(
			"Service discovery endpoint mask",
			defaultEndpointMask,
			cfg.DB.EndpointMask,
			validateEndpointMask,
		)
		if err != nil {
			return err
		}
		normalizedMask, err := NormalizeEndpointMask(maskInput)
		if err != nil {
			return err
		}
		cfg.DB.EndpointMask = normalizedMask
	}

	portStr, err := pr.AskString(
		"Local proxy port",
		strconv.Itoa(defaultLocalPort),
		strconv.Itoa(cfg.DB.LocalPort),
		cli.ValidatePort,
	)
	if err != nil {
		return err
	}
	port, _ := strconv.Atoi(strings.TrimSpace(portStr))
	cfg.DB.LocalPort = port

	role, err := pr.AskString(
		"Preferred endpoint role (master/sync/async/any)",
		defaultPreferRole,
		cfg.DB.PreferRole,
		validateRole,
	)
	if err != nil {
		return err
	}
	cfg.DB.PreferRole = strings.ToLower(strings.TrimSpace(role))

	cfg.DB.ServiceDiscoveryURL = ""
	if err := config.Save(s.rt.Paths.ConfigPath, cfg); err != nil {
		return err
	}

	fmt.Println("db service discovery configured:", serviceDiscoveryCurrent(&cfg))
	fmt.Println("db endpoint mask:", cfg.DB.EndpointMask)
	fmt.Printf("db local address: %s:%d\n", cfg.DB.LocalHost, cfg.DB.LocalPort)
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
	if err := ensureServiceDiscoveryConfigured(cfg); err != nil {
		return fmt.Errorf("%w (run `db init` first)", err)
	}
	if len(cfg.DB.ServiceNames) == 0 {
		return fmt.Errorf("no database services configured (use `db add <service>`)")
	}

	if force {
		if err := s.promptRuntimeConfig(&cfg, hasCfg); err != nil {
			return err
		}
	}

	for _, service := range cfg.DB.ServiceNames {
		endpointURL, ep, err := s.resolveEndpoint(cfg, service)
		if err != nil {
			return fmt.Errorf("service %q validation via service discovery failed: %w", service, err)
		}
		if err := CheckTCPConnectivity(ep.Address, defaultConnectivityTimeout); err != nil {
			return fmt.Errorf("service %q endpoint is unreachable (%s): %w", service, ep.Address, err)
		}

		setServiceValue(&cfg.DB.ServiceTargets, service, ep.Address)
		setServiceValue(&cfg.DB.ServiceInstances, service, ep.InstanceName)
		fmt.Printf("service %s -> %s (%s)\n", service, ep.Address, endpointURL)
	}

	if strings.TrimSpace(cfg.DB.ServiceName) == "" {
		cfg.DB.ServiceName = cfg.DB.ServiceNames[0]
	}
	cfg.DB.TargetAddress = getServiceValue(cfg.DB.ServiceTargets, cfg.DB.ServiceName)
	cfg.DB.TargetInstance = getServiceValue(cfg.DB.ServiceInstances, cfg.DB.ServiceName)
	cfg.DB.ServiceDiscoveryURL = ""

	if err := config.Save(s.rt.Paths.ConfigPath, cfg); err != nil {
		return err
	}
	if err := s.writeProxyRoutesFile(cfg); err != nil {
		return err
	}
	if err := s.ensureProxyRunning(cfg); err != nil {
		return err
	}

	listenAddr := fmt.Sprintf("%s:%d", cfg.DB.LocalHost, cfg.DB.LocalPort)
	fmt.Println("db local address:", listenAddr)
	fmt.Println("db services:", servicesLabel(cfg.DB.ServiceNames))
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
	if err := ensureServiceDiscoveryConfigured(cfg); err != nil {
		return fmt.Errorf("%w (run `db init` first)", err)
	}

	service := strings.TrimSpace(serviceArg)
	if service == "" {
		pr := cli.NewPrompter(os.Stdin, os.Stdout)
		prompted, err := pr.AskString("Database service name", "", cfg.DB.ServiceName, validateServiceName)
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
		return fmt.Errorf("service %q validation via service discovery failed: %w", service, err)
	}
	if err := CheckTCPConnectivity(ep.Address, defaultConnectivityTimeout); err != nil {
		return fmt.Errorf("service %q endpoint is unreachable (%s): %w", service, ep.Address, err)
	}

	cfg.DB.ServiceName = service
	cfg.DB.ServiceNames = upsertServiceName(cfg.DB.ServiceNames, service)
	setServiceValue(&cfg.DB.ServiceTargets, service, ep.Address)
	setServiceValue(&cfg.DB.ServiceInstances, service, ep.InstanceName)
	cfg.DB.TargetAddress = ep.Address
	cfg.DB.TargetInstance = ep.InstanceName
	cfg.DB.ServiceDiscoveryURL = ""

	if err := config.Save(s.rt.Paths.ConfigPath, cfg); err != nil {
		return err
	}
	if err := s.writeProxyRoutesFile(cfg); err != nil {
		return err
	}
	if err := s.ensureProxyRunning(cfg); err != nil {
		return err
	}

	listenAddr := fmt.Sprintf("%s:%d", cfg.DB.LocalHost, cfg.DB.LocalPort)
	fmt.Println("db service added:", service)
	fmt.Println("db endpoint url:", endpointURL)
	fmt.Println("db selected endpoint:", ep.Address)
	if ep.InstanceName != "" {
		fmt.Println("db selected instance:", ep.InstanceName)
	}
	fmt.Println("db endpoint connectivity: ok")
	fmt.Println("db local address:", listenAddr)
	fmt.Printf("jdbc url: jdbc:postgresql://%s/%s\n", listenAddr, service)
	fmt.Println("db services:", servicesLabel(cfg.DB.ServiceNames))
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

	updated, removed := removeServiceName(cfg.DB.ServiceNames, service)
	if !removed {
		fmt.Println("db service not found:", service)
		return nil
	}
	cfg.DB.ServiceNames = updated
	deleteServiceValue(cfg.DB.ServiceTargets, service)
	deleteServiceValue(cfg.DB.ServiceInstances, service)

	if strings.EqualFold(cfg.DB.ServiceName, service) {
		cfg.DB.ServiceName = ""
		cfg.DB.TargetAddress = ""
		cfg.DB.TargetInstance = ""
		if len(cfg.DB.ServiceNames) > 0 {
			cfg.DB.ServiceName = cfg.DB.ServiceNames[0]
			cfg.DB.TargetAddress = getServiceValue(cfg.DB.ServiceTargets, cfg.DB.ServiceName)
			cfg.DB.TargetInstance = getServiceValue(cfg.DB.ServiceInstances, cfg.DB.ServiceName)
		}
	}

	if err := config.Save(s.rt.Paths.ConfigPath, cfg); err != nil {
		return err
	}

	if len(cfg.DB.ServiceNames) == 0 {
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
	fmt.Println("db services:", servicesLabel(cfg.DB.ServiceNames))
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
	fmt.Println("Service discovery scheme:", emptyIf(cfg.DB.ServiceDiscoveryScheme))
	fmt.Println("Service discovery host:", emptyIf(cfg.DB.ServiceDiscoveryHost))
	fmt.Println("Endpoint mask:", emptyIf(cfg.DB.EndpointMask))
	fmt.Println("Preferred role:", emptyIf(cfg.DB.PreferRole))
	fmt.Printf("Local address: %s:%d\n", cfg.DB.LocalHost, cfg.DB.LocalPort)
	fmt.Println("Active service:", emptyIf(cfg.DB.ServiceName))
	fmt.Println("Services:", servicesLabel(cfg.DB.ServiceNames))

	if len(cfg.DB.ServiceNames) > 0 {
		fmt.Println("Service endpoints:")
		for _, service := range cfg.DB.ServiceNames {
			target := getServiceValue(cfg.DB.ServiceTargets, service)
			instance := getServiceValue(cfg.DB.ServiceInstances, service)
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
		fmt.Println("Proxy listen addr:", fmt.Sprintf("%s:%d", cfg.DB.LocalHost, cfg.DB.LocalPort))
		fmt.Println("Proxy routes file:", s.proxyRoutesPath())
	}

	fmt.Println("Proxy pid file:", s.rt.Paths.DBProxyPIDFile)
	fmt.Println("Proxy meta file:", s.rt.Paths.DBProxyMetaFile)
	fmt.Println("Proxy log:", s.rt.Paths.DBProxyLogFile)
	fmt.Printf("JDBC template: jdbc:postgresql://%s:%d/%s\n", cfg.DB.LocalHost, cfg.DB.LocalPort, "<database>")
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

	cfg.DB.ServiceName = strings.TrimSpace(cfg.DB.ServiceName)
	cfg.DB.ServiceNames = normalizeServiceNames(cfg.DB.ServiceNames)
	if cfg.DB.ServiceName != "" {
		cfg.DB.ServiceNames = upsertServiceName(cfg.DB.ServiceNames, cfg.DB.ServiceName)
	}
	if cfg.DB.ServiceName == "" && len(cfg.DB.ServiceNames) > 0 {
		cfg.DB.ServiceName = cfg.DB.ServiceNames[0]
	}

	cfg.DB.ServiceTargets = normalizeServiceValues(cfg.DB.ServiceNames, cfg.DB.ServiceTargets)
	cfg.DB.ServiceInstances = normalizeServiceValues(cfg.DB.ServiceNames, cfg.DB.ServiceInstances)

	if cfg.DB.ServiceName != "" {
		if strings.TrimSpace(cfg.DB.TargetAddress) != "" && getServiceValue(cfg.DB.ServiceTargets, cfg.DB.ServiceName) == "" {
			setServiceValue(&cfg.DB.ServiceTargets, cfg.DB.ServiceName, cfg.DB.TargetAddress)
		}
		if strings.TrimSpace(cfg.DB.TargetInstance) != "" && getServiceValue(cfg.DB.ServiceInstances, cfg.DB.ServiceName) == "" {
			setServiceValue(&cfg.DB.ServiceInstances, cfg.DB.ServiceName, cfg.DB.TargetInstance)
		}
		if cfg.DB.TargetAddress == "" {
			cfg.DB.TargetAddress = getServiceValue(cfg.DB.ServiceTargets, cfg.DB.ServiceName)
		}
		if cfg.DB.TargetInstance == "" {
			cfg.DB.TargetInstance = getServiceValue(cfg.DB.ServiceInstances, cfg.DB.ServiceName)
		}
	}

	if strings.TrimSpace(cfg.DB.ServiceDiscoveryScheme) == "" {
		cfg.DB.ServiceDiscoveryScheme = defaultServiceDiscoveryScheme
	}
	if strings.TrimSpace(cfg.DB.EndpointMask) == "" {
		cfg.DB.EndpointMask = defaultEndpointMask
	}
	if strings.TrimSpace(cfg.DB.LocalHost) == "" {
		cfg.DB.LocalHost = defaultLocalHost
	}
	if cfg.DB.LocalPort == 0 {
		cfg.DB.LocalPort = defaultLocalPort
	}
	if strings.TrimSpace(cfg.DB.PreferRole) == "" {
		cfg.DB.PreferRole = defaultPreferRole
	}
}

func (s Service) promptRuntimeConfig(cfg *config.Config, hasCfg bool) error {
	pr := cli.NewPrompter(os.Stdin, os.Stdout)

	curPort := ""
	if cfg.DB.LocalPort > 0 {
		curPort = strconv.Itoa(cfg.DB.LocalPort)
	}
	portStr, err := pr.AskString("Local proxy port", strconv.Itoa(defaultLocalPort), curPort, cli.ValidatePort)
	if err != nil {
		return err
	}
	port, _ := strconv.Atoi(strings.TrimSpace(portStr))
	cfg.DB.LocalPort = port

	role, err := pr.AskString(
		"Preferred endpoint role (master/sync/async/any)",
		defaultPreferRole,
		cfg.DB.PreferRole,
		validateRole,
	)
	if err != nil {
		return err
	}
	cfg.DB.PreferRole = strings.ToLower(strings.TrimSpace(role))

	_ = hasCfg
	return nil
}

func (s Service) resolveEndpoint(cfg config.Config, serviceName string) (string, Endpoint, error) {
	endpointURL, err := BuildEndpointURL(
		cfg.DB.ServiceDiscoveryScheme,
		cfg.DB.ServiceDiscoveryHost,
		cfg.DB.EndpointMask,
		serviceName,
	)
	if err != nil {
		return "", Endpoint{}, err
	}

	endpoints, err := FetchEndpoints(endpointURL)
	if err != nil {
		return "", Endpoint{}, err
	}
	ep, err := ChooseEndpoint(endpoints, cfg.DB.PreferRole)
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

func validateServiceDiscoveryInput(s string) error {
	_, _, err := NormalizeServiceDiscoveryInput(s)
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

func ensureServiceDiscoveryConfigured(cfg config.Config) error {
	if strings.TrimSpace(cfg.DB.ServiceDiscoveryHost) == "" {
		return fmt.Errorf("service discovery host is not configured")
	}
	if strings.TrimSpace(cfg.DB.EndpointMask) == "" {
		return fmt.Errorf("Service discovery endpoint mask is not configured")
	}
	return nil
}

func (s Service) backfillFromLegacy(cfg *config.Config) {
	legacy := strings.TrimSpace(cfg.DB.ServiceDiscoveryURL)
	if legacy == "" {
		return
	}
	if strings.TrimSpace(cfg.DB.ServiceDiscoveryHost) != "" && strings.TrimSpace(cfg.DB.ServiceDiscoveryScheme) != "" {
		return
	}

	scheme, host, err := NormalizeServiceDiscoveryInput(legacy)
	if err != nil {
		return
	}
	if strings.TrimSpace(cfg.DB.ServiceDiscoveryScheme) == "" {
		cfg.DB.ServiceDiscoveryScheme = scheme
	}
	if strings.TrimSpace(cfg.DB.ServiceDiscoveryHost) == "" {
		cfg.DB.ServiceDiscoveryHost = host
	}
}

func serviceDiscoveryCurrent(cfg *config.Config) string {
	host := strings.TrimSpace(cfg.DB.ServiceDiscoveryHost)
	if host == "" {
		return strings.TrimSpace(cfg.DB.ServiceDiscoveryURL)
	}
	scheme := strings.TrimSpace(cfg.DB.ServiceDiscoveryScheme)
	if scheme == "" {
		scheme = defaultServiceDiscoveryScheme
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
	if len(cfg.DB.ServiceNames) == 0 {
		return fmt.Errorf("no services configured")
	}
	if err := os.MkdirAll(s.rt.Paths.StateDir, 0o755); err != nil {
		return err
	}

	routes := proxyRoutesFile{
		Services: make(map[string]proxyRoute, len(cfg.DB.ServiceNames)),
	}
	for _, service := range cfg.DB.ServiceNames {
		target := getServiceValue(cfg.DB.ServiceTargets, service)
		if target == "" {
			return fmt.Errorf("target endpoint for service %q is not set", service)
		}
		routes.Services[serviceKey(service)] = proxyRoute{
			Service:    service,
			TargetAddr: target,
			Instance:   getServiceValue(cfg.DB.ServiceInstances, service),
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
	listenAddr := fmt.Sprintf("%s:%d", cfg.DB.LocalHost, cfg.DB.LocalPort)
	routesPath := s.proxyRoutesPath()
	meta := proxyMeta{
		ListenAddr: listenAddr,
		RoutesFile: routesPath,
		Services:   normalizeServiceNames(cfg.DB.ServiceNames),
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
