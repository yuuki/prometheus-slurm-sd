package config

import (
	"os"
	"strings"
	"testing"
)

func TestLoadConfigFromReader(t *testing.T) {
	// Test cases
	tests := []struct {
		name        string
		input       string
		wantErr     bool
		validateCfg func(*Config) bool
	}{
		{
			name: "valid basic config",
			input: `
slurm_api_endpoint: "http://slurm-api:6820"
jobs:
  - name: node
    port: 9100
`,
			wantErr: false,
			validateCfg: func(cfg *Config) bool {
				if cfg.SlurmAPIEndpoint != "http://slurm-api:6820" {
					return false
				}
				if len(cfg.Jobs) != 1 {
					return false
				}
				if cfg.Jobs[0].Name != "node" || cfg.Jobs[0].Port != 9100 {
					return false
				}
				return true
			},
		},
		{
			name: "valid complete config",
			input: `
slurm_api_endpoint: "http://slurm-api:6820"
slurm_api_version: "v0.0.40"
slurm_api_username: "testuser"
slurm_api_token: "testtoken"
listen_address: ":9090"
update_interval: "10m"
jobs:
  - name: node
    port: 9100
  - name: dcgm
    port: 9400
`,
			wantErr: false,
			validateCfg: func(cfg *Config) bool {
				if cfg.SlurmAPIEndpoint != "http://slurm-api:6820" {
					return false
				}
				if cfg.SlurmAPIVersion != "v0.0.40" {
					return false
				}
				if cfg.SlurmAPIUsername != "testuser" {
					return false
				}
				if cfg.SlurmAPIToken != "testtoken" {
					return false
				}
				if cfg.ListenAddress != ":9090" {
					return false
				}
				if cfg.UpdateInterval != "10m" {
					return false
				}
				if len(cfg.Jobs) != 2 {
					return false
				}
				return true
			},
		},
		{
			name: "invalid yaml",
			input: `
slurm_api_endpoint: "http://slurm-api:6820"
jobs:
  - name: node
    port: 9100
  invalid yaml format
`,
			wantErr: true,
			validateCfg: func(cfg *Config) bool {
				return true // Not used in error case
			},
		},
		{
			name:    "empty config",
			input:   ``,
			wantErr: false,
			validateCfg: func(cfg *Config) bool {
				return cfg.SlurmAPIVersion == "v0.0.38" &&
					cfg.ListenAddress == ":8080" &&
					cfg.UpdateInterval == "5m"
			},
		},
	}

	// Run test cases
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reader := strings.NewReader(tc.input)
			cfg, err := LoadConfigFromReader(reader)
			if (err != nil) != tc.wantErr {
				t.Errorf("LoadConfigFromReader() error = %v, wantErr %v", err, tc.wantErr)
				return
			}

			if !tc.wantErr && !tc.validateCfg(cfg) {
				t.Errorf("LoadConfigFromReader() got invalid config: %+v", cfg)
			}
		})
	}
}

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file
	content := `
slurm_api_endpoint: "http://slurm-api:6820"
jobs:
  - name: test
    port: 9999
`
	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	// Test loading from file
	cfg, err := LoadConfig(tmpfile.Name())
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Validate config
	if cfg.SlurmAPIEndpoint != "http://slurm-api:6820" {
		t.Errorf("Incorrect SlurmAPIEndpoint: %s", cfg.SlurmAPIEndpoint)
	}
	if len(cfg.Jobs) != 1 || cfg.Jobs[0].Name != "test" || cfg.Jobs[0].Port != 9999 {
		t.Errorf("Incorrect Jobs configuration: %+v", cfg.Jobs)
	}

	// Test file not found
	_, err = LoadConfig("non-existent-file.yaml")
	if err == nil {
		t.Error("LoadConfig() expected error for non-existent file, got nil")
	}
}
