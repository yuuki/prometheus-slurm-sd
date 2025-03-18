package main

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/alecthomas/kingpin/v2"
)

func TestCommandLineArgs(t *testing.T) {
	// Save original args and restore after test
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	tests := []struct {
		name        string
		args        []string
		validate    func(configFile, logLevel, listenAddress, slurmApiEndpoint, slurmApiVersion, slurmApiUsername, slurmApiToken, updateInterval *string) bool
		expectError bool
		skipTest    bool // Skip validation for help/version
	}{
		{
			name: "default values",
			args: []string{"prometheus-slurm-sd"},
			validate: func(configFile, logLevel, listenAddress, slurmApiEndpoint, slurmApiVersion, slurmApiUsername, slurmApiToken, updateInterval *string) bool {
				return *configFile == "config.yaml"
			},
			expectError: false,
			skipTest:    false,
		},
		{
			name: "custom config file",
			args: []string{"prometheus-slurm-sd", "--config.file=/custom/path/config.yaml"},
			validate: func(configFile, logLevel, listenAddress, slurmApiEndpoint, slurmApiVersion, slurmApiUsername, slurmApiToken, updateInterval *string) bool {
				return *configFile == "/custom/path/config.yaml"
			},
			expectError: false,
			skipTest:    false,
		},
		{
			name: "multiple options",
			args: []string{
				"prometheus-slurm-sd",
				"--config.file=test.yaml",
				"--log.level=debug",
				"--web.listen-address=:9090",
				"--slurm.api-endpoint=http://test-slurm:6820",
				"--slurm.api-version=v0.0.38",
				"--slurm.api-username=testuser",
				"--slurm.api-token=testtoken",
				"--update.interval=10m",
			},
			validate: func(configFile, logLevel, listenAddress, slurmApiEndpoint, slurmApiVersion, slurmApiUsername, slurmApiToken, updateInterval *string) bool {
				return *configFile == "test.yaml" &&
					*logLevel == "debug" &&
					*listenAddress == ":9090" &&
					*slurmApiEndpoint == "http://test-slurm:6820" &&
					*slurmApiVersion == "v0.0.38" &&
					*slurmApiUsername == "testuser" &&
					*slurmApiToken == "testtoken" &&
					*updateInterval == "10m"
			},
			expectError: false,
			skipTest:    false,
		},
		{
			name: "invalid log level",
			args: []string{"prometheus-slurm-sd", "--log.level=invalid"},
			validate: func(configFile, logLevel, listenAddress, slurmApiEndpoint, slurmApiVersion, slurmApiUsername, slurmApiToken, updateInterval *string) bool {
				return true
			},
			expectError: true,
			skipTest:    false,
		},
		{
			name: "show help",
			args: []string{"prometheus-slurm-sd", "--help"},
			validate: func(configFile, logLevel, listenAddress, slurmApiEndpoint, slurmApiVersion, slurmApiUsername, slurmApiToken, updateInterval *string) bool {
				return true
			},
			expectError: true,
			skipTest:    true, // Skip validation for --help
		},
		{
			name: "show version",
			args: []string{"prometheus-slurm-sd", "--version"},
			validate: func(configFile, logLevel, listenAddress, slurmApiEndpoint, slurmApiVersion, slurmApiUsername, slurmApiToken, updateInterval *string) bool {
				return true
			},
			expectError: true,
			skipTest:    true, // Skip validation for --version
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Skip test completely for help and version
			if tc.skipTest {
				t.Skip("Skipping help/version test")
				return
			}

			// Recreate the application for each test
			app := kingpin.New("prometheus-slurm-sd", "Prometheus service discovery for Slurm clusters").
				Version(version)

			configFile := app.Flag("config.file", "Config file path").
				Default("config.yaml").String()
			logLevel := app.Flag("log.level", "Log level (debug, info, warn, error)").
				Default("info").Enum("debug", "info", "warn", "error")
			listenAddress := app.Flag("web.listen-address", "Address to listen on for HTTP requests").
				String()
			slurmApiEndpoint := app.Flag("slurm.api-endpoint", "Slurm REST API endpoint").
				String()
			slurmApiVersion := app.Flag("slurm.api-version", "Slurm REST API version").
				String()
			slurmApiUsername := app.Flag("slurm.api-username", "Slurm REST API username").
				String()
			slurmApiToken := app.Flag("slurm.api-token", "Slurm REST API token").
				String()
			updateInterval := app.Flag("update.interval", "Update interval for fetching Slurm data").
				String()

			// Silence the standard output and error for help/version flags
			// Set a custom terminate function that won't exit the test
			app.Terminate(func(int) {
				// Do nothing, just catch the termination
			})
			app.UsageWriter(io.Discard)
			app.ErrorWriter(io.Discard)

			// Set the test arguments
			os.Args = tc.args

			// Parse the arguments
			_, err := app.Parse(os.Args[1:])

			// Check error expectation
			if (err != nil) != tc.expectError {
				// Skip error check for help and version flags
				if !tc.expectError || (!strings.Contains(err.Error(), "--help") && !strings.Contains(err.Error(), "--version")) {
					t.Errorf("Parse() error = %v, expectError %v", err, tc.expectError)
				}
				return
			}

			// Skip validation if an error was expected
			if err != nil {
				return
			}

			// Validate the parsed values
			if !tc.validate(configFile, logLevel, listenAddress, slurmApiEndpoint, slurmApiVersion, slurmApiUsername, slurmApiToken, updateInterval) {
				t.Errorf("Validation failed for args: %v", tc.args)
				t.Logf("Values: config.file=%v, log.level=%v, web.listen-address=%v, "+
					"slurm.api-endpoint=%v, slurm.api-version=%v, slurm.api-username=%v, "+
					"slurm.api-token=%v, update.interval=%v",
					*configFile, *logLevel, *listenAddress, *slurmApiEndpoint,
					*slurmApiVersion, *slurmApiUsername, *slurmApiToken, *updateInterval)
			}
		})
	}
}
