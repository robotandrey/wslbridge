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
