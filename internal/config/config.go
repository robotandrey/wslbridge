package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds wslbridge configuration.
type Config struct {
	Socks struct {
		Host string `yaml:"host"`
		Port int    `yaml:"port"`
	} `yaml:"socks"`

	Tun struct {
		Dev  string `yaml:"dev"`
		CIDR string `yaml:"cidr"`
	} `yaml:"tun"`

	DNS struct {
		Nameserver string `yaml:"nameserver"`
	} `yaml:"dns"`

	PGBouncer struct {
		WardenScheme     string            `yaml:"warden_scheme"`
		WardenHost       string            `yaml:"warden_host"`
		EndpointMask     string            `yaml:"endpoint_mask"`
		ServiceName      string            `yaml:"service_name"`
		ServiceNames     []string          `yaml:"service_names,omitempty"`
		ServicePorts     map[string]int    `yaml:"service_ports,omitempty"`
		ServiceTargets   map[string]string `yaml:"service_targets,omitempty"`
		ServiceInstances map[string]string `yaml:"service_instances,omitempty"`
		WardenURL        string            `yaml:"warden_url,omitempty"` // legacy, kept for backward compatibility
		LocalHost        string            `yaml:"local_host"`
		LocalPort        int               `yaml:"local_port"`
		PreferRole       string            `yaml:"prefer_role"`
		TargetAddress    string            `yaml:"target_address"`
		TargetInstance   string            `yaml:"target_instance"`
	} `yaml:"pgbouncer"`
}

// Load reads config from the given path.
func Load(path string) (Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return Config{}, err
	}
	return c, nil
}

// Save writes config to the given path.
func Save(path string, c Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}
