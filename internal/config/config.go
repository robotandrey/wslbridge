package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type SocksConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type TunConfig struct {
	Dev  string `yaml:"dev"`
	CIDR string `yaml:"cidr"`
}

type DNSConfig struct {
	Nameserver string `yaml:"nameserver"`
}

type DBConfig struct {
	ServiceDiscoveryScheme string
	ServiceDiscoveryHost   string
	EndpointMask           string
	AuthLookupUser         string
	AuthLookupPass         string
	AuthQuery              string
	ServiceName            string
	ServiceNames           []string
	ServicePorts           map[string]int
	ServiceTargets         map[string]string
	ServiceInstances       map[string]string
	ServiceDiscoveryURL    string
	LocalHost              string
	LocalPort              int
	PreferRole             string
	TargetAddress          string
	TargetInstance         string
}

// Config holds wslbridge configuration.
type Config struct {
	Socks SocksConfig
	Tun   TunConfig
	DNS   DNSConfig
	DB    DBConfig
}

type dbDiskConfig struct {
	ServiceDiscoveryScheme string            `yaml:"service_discovery_scheme,omitempty"`
	ServiceDiscoveryHost   string            `yaml:"service_discovery_host,omitempty"`
	EndpointMask           string            `yaml:"endpoint_mask,omitempty"`
	AuthLookupUser         string            `yaml:"auth_lookup_user,omitempty"`
	AuthLookupPass         string            `yaml:"auth_lookup_password,omitempty"`
	AuthQuery              string            `yaml:"auth_query,omitempty"`
	ServiceName            string            `yaml:"service_name,omitempty"`
	ServiceNames           []string          `yaml:"service_names,omitempty"`
	ServicePorts           map[string]int    `yaml:"service_ports,omitempty"`
	ServiceTargets         map[string]string `yaml:"service_targets,omitempty"`
	ServiceInstances       map[string]string `yaml:"service_instances,omitempty"`
	ServiceDiscoveryURL    string            `yaml:"service_discovery_url,omitempty"`
	LocalHost              string            `yaml:"local_host,omitempty"`
	LocalPort              int               `yaml:"local_port,omitempty"`
	PreferRole             string            `yaml:"prefer_role,omitempty"`
	TargetAddress          string            `yaml:"target_address,omitempty"`
	TargetInstance         string            `yaml:"target_instance,omitempty"`
}

type configDisk struct {
	Socks SocksConfig  `yaml:"socks"`
	Tun   TunConfig    `yaml:"tun"`
	DNS   DNSConfig    `yaml:"dns"`
	DB    dbDiskConfig `yaml:"db"`
}

func (d dbDiskConfig) toRuntime() DBConfig {
	return DBConfig{
		ServiceDiscoveryScheme: d.ServiceDiscoveryScheme,
		ServiceDiscoveryHost:   d.ServiceDiscoveryHost,
		EndpointMask:           d.EndpointMask,
		AuthLookupUser:         d.AuthLookupUser,
		AuthLookupPass:         d.AuthLookupPass,
		AuthQuery:              d.AuthQuery,
		ServiceName:            d.ServiceName,
		ServiceNames:           d.ServiceNames,
		ServicePorts:           d.ServicePorts,
		ServiceTargets:         d.ServiceTargets,
		ServiceInstances:       d.ServiceInstances,
		ServiceDiscoveryURL:    d.ServiceDiscoveryURL,
		LocalHost:              d.LocalHost,
		LocalPort:              d.LocalPort,
		PreferRole:             d.PreferRole,
		TargetAddress:          d.TargetAddress,
		TargetInstance:         d.TargetInstance,
	}
}

func (d dbDiskConfig) isZero() bool {
	return d.ServiceDiscoveryScheme == "" &&
		d.ServiceDiscoveryHost == "" &&
		d.EndpointMask == "" &&
		d.AuthLookupUser == "" &&
		d.AuthLookupPass == "" &&
		d.AuthQuery == "" &&
		d.ServiceName == "" &&
		len(d.ServiceNames) == 0 &&
		len(d.ServicePorts) == 0 &&
		len(d.ServiceTargets) == 0 &&
		len(d.ServiceInstances) == 0 &&
		d.ServiceDiscoveryURL == "" &&
		d.LocalHost == "" &&
		d.LocalPort == 0 &&
		d.PreferRole == "" &&
		d.TargetAddress == "" &&
		d.TargetInstance == ""
}

func dbDiskFromRuntime(c DBConfig) dbDiskConfig {
	return dbDiskConfig{
		ServiceDiscoveryScheme: c.ServiceDiscoveryScheme,
		ServiceDiscoveryHost:   c.ServiceDiscoveryHost,
		EndpointMask:           c.EndpointMask,
		AuthLookupUser:         c.AuthLookupUser,
		AuthLookupPass:         c.AuthLookupPass,
		AuthQuery:              c.AuthQuery,
		ServiceName:            c.ServiceName,
		ServiceNames:           c.ServiceNames,
		ServicePorts:           c.ServicePorts,
		ServiceTargets:         c.ServiceTargets,
		ServiceInstances:       c.ServiceInstances,
		ServiceDiscoveryURL:    c.ServiceDiscoveryURL,
		LocalHost:              c.LocalHost,
		LocalPort:              c.LocalPort,
		PreferRole:             c.PreferRole,
		TargetAddress:          c.TargetAddress,
		TargetInstance:         c.TargetInstance,
	}
}

// Load reads config from the given path.
func Load(path string) (Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	var disk configDisk
	if err := yaml.Unmarshal(b, &disk); err != nil {
		return Config{}, err
	}

	return Config{
		Socks: disk.Socks,
		Tun:   disk.Tun,
		DNS:   disk.DNS,
		DB:    disk.DB.toRuntime(),
	}, nil
}

// Save writes config to the given path.
func Save(path string, c Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	disk := configDisk{
		Socks: c.Socks,
		Tun:   c.Tun,
		DNS:   c.DNS,
		DB:    dbDiskFromRuntime(c.DB),
	}

	b, err := yaml.Marshal(disk)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}
