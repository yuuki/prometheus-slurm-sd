package config

import (
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the program configuration
type Config struct {
	SlurmAPIEndpoint string      `yaml:"slurm_api_endpoint"`
	SlurmAPIVersion  string      `yaml:"slurm_api_version"`
	SlurmAPIToken    string      `yaml:"slurm_api_token,omitempty"`
	SlurmAPIUsername string      `yaml:"slurm_api_username,omitempty"`
	ListenAddress    string      `yaml:"listen_address"`
	UpdateInterval   string      `yaml:"update_interval"`
	Jobs             []JobConfig `yaml:"jobs"`
}

// JobConfig represents the configuration for a Prometheus target job
type JobConfig struct {
	Name string `yaml:"name"`
	Port int    `yaml:"port"`
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer f.Close()

	return LoadConfigFromReader(f)
}

// LoadConfigFromReader loads configuration from an io.Reader
func LoadConfigFromReader(r io.Reader) (*Config, error) {
	var cfg Config
	decoder := yaml.NewDecoder(r)
	if err := decoder.Decode(&cfg); err != nil {
		// Check for EOF (empty file) and return default config
		if err == io.EOF {
			// Set default values for empty config
			cfg.ListenAddress = ":8080"
			cfg.SlurmAPIVersion = "v0.0.38"
			cfg.UpdateInterval = "5m"
			return &cfg, nil
		}
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}

	// Set default values
	if cfg.ListenAddress == "" {
		cfg.ListenAddress = ":8080"
	}
	if cfg.SlurmAPIVersion == "" {
		cfg.SlurmAPIVersion = "v0.0.38"
	}
	if cfg.UpdateInterval == "" {
		cfg.UpdateInterval = "5m"
	}

	return &cfg, nil
}
