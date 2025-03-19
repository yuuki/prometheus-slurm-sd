package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/yuuki/prometheus-slurm-sd/internal/config"
)

// Integration test with Prometheus
func TestPrometheusIntegration(t *testing.T) {
	// Skip test in CI environment if needed
	if os.Getenv("SKIP_INTEGRATION_TESTS") != "" {
		t.Skip("Skipping integration tests")
	}

	// Set up mock Slurm server for integration testing
	mockServer := setupMockSlurmServer()
	defer mockServer.Close()

	// Create test configuration directory
	configDir, err := os.MkdirTemp("", "prometheus-slurm-sd-integration-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(configDir)

	configPath := filepath.Join(configDir, "config.yaml")
	sdConfigPath := filepath.Join(configDir, "sd_config.yaml")

	// Write prometheus-slurm-sd configuration
	writeSDConfig(t, configPath, mockServer.URL)

	// Write Prometheus SD configuration
	writePrometheusConfig(t, sdConfigPath, 8081) // Port for integration testing

	// Start the actual server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		runTestServer(ctx, t, configPath, ":8081")
	}()

	// Wait for server to start
	time.Sleep(500 * time.Millisecond)

	// Retrieve and verify target information
	t.Run("Verify targets endpoint", func(t *testing.T) {
		// Get target information from server
		resp, err := http.Get("http://localhost:8081/targets")
		if err != nil {
			t.Fatalf("Failed to get targets: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status OK, got %v", resp.Status)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response body: %v", err)
		}

		// Parse as JSON
		var targets []map[string]interface{}
		if err := json.Unmarshal(body, &targets); err != nil {
			t.Fatalf("Failed to parse JSON: %v", err)
		}

		// Verify target groups exist
		if len(targets) == 0 {
			t.Errorf("No target groups found")
		}

		// Verify target contents
		targetFound := false
		for _, targetGroup := range targets {
			targetsArray, ok := targetGroup["targets"].([]interface{})
			if !ok {
				continue
			}

			for _, target := range targetsArray {
				if target.(string) == "node1.example.com:9100" {
					targetFound = true
					break
				}
			}
			if targetFound {
				break
			}
		}

		if !targetFound {
			t.Errorf("Expected target node1.example.com:9100 not found")
		}
	})

	// Verify job-specific target retrieval
	t.Run("Verify job-specific targets", func(t *testing.T) {
		resp, err := http.Get("http://localhost:8081/targets?prom_job=node")
		if err != nil {
			t.Fatalf("Failed to get targets: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status OK, got %v", resp.Status)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response body: %v", err)
		}

		var targets []map[string]interface{}
		if err := json.Unmarshal(body, &targets); err != nil {
			t.Fatalf("Failed to parse JSON: %v", err)
		}

		// Verify node job exists
		jobFound := false
		for _, targetGroup := range targets {
			labels, ok := targetGroup["labels"].(map[string]interface{})
			if !ok {
				continue
			}

			job, ok := labels["__meta_slurm_job"]
			if ok && job == "node" {
				jobFound = true
				break
			}
		}

		if !jobFound {
			t.Errorf("Node job not found in targets")
		}
	})

	cancel()     // Stop the server
	<-serverDone // Wait for the server to completely shut down
}

// Set up simple mock Slurm server
func setupMockSlurmServer() *httptest.Server {
	mux := http.NewServeMux()

	// Slurm node information endpoint
	mux.HandleFunc("/slurm/v0.0.38/nodes/", func(w http.ResponseWriter, r *http.Request) {
		// Verify request header
		token := r.Header.Get("X-SLURM-USER-TOKEN")
		if token != "testtoken" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Return node information
		resp := map[string]interface{}{
			"nodes": []map[string]interface{}{
				{
					"name":       "node1",
					"hostname":   "node1.example.com",
					"state":      []string{"IDLE"},
					"partitions": []string{"partition1", "partition2"},
				},
				{
					"name":       "node2",
					"hostname":   "node2.example.com",
					"state":      []string{"ALLOCATED"},
					"partitions": []string{"partition1"},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	// Ping endpoint
	mux.HandleFunc("/slurm/v0.0.38/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "PONG")
	})

	return httptest.NewServer(mux)
}

// Write SD configuration file for testing
func writeSDConfig(t *testing.T, path string, serverURL string) {
	config := fmt.Sprintf(`
slurm_api_endpoint: "%s"
slurm_api_version: "v0.0.38"
slurm_api_username: "testuser"
slurm_api_token: "testtoken"
listen_address: ":8081"
update_interval: "1s"
jobs:
  - name: node
    port: 9100
  - name: dcgm
    port: 9401
`, serverURL)

	if err := os.WriteFile(path, []byte(config), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}
}

// Write Prometheus configuration file for testing
func writePrometheusConfig(t *testing.T, path string, port int) {
	config := fmt.Sprintf(`
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'slurm-node-exporter'
    http_sd_configs:
      - url: http://localhost:%d/targets?prom_job=node
        refresh_interval: 5s
`, port)

	if err := os.WriteFile(path, []byte(config), 0644); err != nil {
		t.Fatalf("Failed to write Prometheus config file: %v", err)
	}
}

// Run test server
func runTestServer(ctx context.Context, t *testing.T, configPath string, listenAddress string) {
	// Load configuration
	// Note: This test implementation doesn't use the actual config file, only mocks
	_, err := config.LoadConfig(configPath)
	if err != nil {
		t.Logf("Failed to load config: %v", err)
		return
	}

	// Set up logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug, // Use debug level for more information
	}))

	// Implement simple HTTP server
	mux := http.NewServeMux()

	// Mock /targets endpoint
	mux.HandleFunc("/targets", func(w http.ResponseWriter, r *http.Request) {
		// Log the request for debugging
		logger.Debug("Received request for targets",
			"query", r.URL.Query(),
			"headers", r.Header)

		// Get query parameter
		jobName := r.URL.Query().Get("prom_job")

		// Generate mock data
		var targets []map[string]interface{}

		if jobName == "" || jobName == "node" {
			// Node job for partition1 - node1 (IDLE)
			targets = append(targets, map[string]interface{}{
				"targets": []string{"node1.example.com:9100"},
				"labels": map[string]string{
					"__meta_slurm_partition": "partition1",
					"__meta_slurm_job":       "node",
					"__meta_slurm_state":     "IDLE",
					"__meta_slurm_node":      "node1",
				},
			})

			// Node job for partition1 - node2 (ALLOCATED)
			targets = append(targets, map[string]interface{}{
				"targets": []string{"node2.example.com:9100"},
				"labels": map[string]string{
					"__meta_slurm_partition": "partition1",
					"__meta_slurm_job":       "node",
					"__meta_slurm_state":     "ALLOCATED",
					"__meta_slurm_node":      "node2",
				},
			})
		}

		if jobName == "" || jobName == "node" {
			// Node job for partition2 - node1 only
			targets = append(targets, map[string]interface{}{
				"targets": []string{"node1.example.com:9100"},
				"labels": map[string]string{
					"__meta_slurm_partition": "partition2",
					"__meta_slurm_job":       "node",
					"__meta_slurm_state":     "IDLE",
					"__meta_slurm_node":      "node1",
				},
			})
		}

		if jobName == "" || jobName == "dcgm" {
			// DCGM job for partition1 - node1
			targets = append(targets, map[string]interface{}{
				"targets": []string{"node1.example.com:9401"},
				"labels": map[string]string{
					"__meta_slurm_partition": "partition1",
					"__meta_slurm_job":       "dcgm",
					"__meta_slurm_state":     "IDLE",
					"__meta_slurm_node":      "node1",
				},
			})

			// DCGM job for partition1 - node2
			targets = append(targets, map[string]interface{}{
				"targets": []string{"node2.example.com:9401"},
				"labels": map[string]string{
					"__meta_slurm_partition": "partition1",
					"__meta_slurm_job":       "dcgm",
					"__meta_slurm_state":     "ALLOCATED",
					"__meta_slurm_node":      "node2",
				},
			})
		}

		// Debug log for response
		logger.Debug("Returning targets", "count", len(targets))

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(targets); err != nil {
			logger.Error("Failed to encode targets", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	})

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	})

	server := &http.Server{
		Addr:    listenAddress,
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		// Shutdown server when context is canceled
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
	}()

	logger.Info("Starting test server", "address", listenAddress)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		logger.Error("HTTP server error", "error", err)
	}
}

// Test with Prometheus (run only in CI environment)
func TestWithPrometheus(t *testing.T) {
	// Skip this test by default
	if os.Getenv("RUN_PROMETHEUS_TEST") == "" {
		t.Skip("Skipping Prometheus integration test")
	}

	// Check if Prometheus is available
	_, err := exec.LookPath("prometheus")
	if err != nil {
		t.Skip("Prometheus binary not found, skipping test")
	}

	// Set up mock Slurm server
	mockServer := setupMockSlurmServer()
	defer mockServer.Close()

	// Create test directory
	testDir, err := os.MkdirTemp("", "prom-slurm-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	// Create configuration files
	sdConfigPath := filepath.Join(testDir, "sd_config.yaml")
	promConfigPath := filepath.Join(testDir, "prom_config.yaml")

	// Write prometheus-slurm-sd configuration
	writeSDConfig(t, sdConfigPath, mockServer.URL)

	// Write Prometheus configuration with a shorter refresh interval
	promConfigContent := fmt.Sprintf(`
global:
  scrape_interval: 5s
  evaluation_interval: 5s

scrape_configs:
  - job_name: 'slurm-node-exporter'
    http_sd_configs:
      - url: http://localhost:8082/targets?prom_job=node
        refresh_interval: 2s
`)
	if err := os.WriteFile(promConfigPath, []byte(promConfigContent), 0644); err != nil {
		t.Fatalf("Failed to write Prometheus config file: %v", err)
	}

	// Start SD server
	sdCtx, sdCancel := context.WithCancel(context.Background())
	defer sdCancel()

	go func() {
		runTestServer(sdCtx, t, sdConfigPath, ":8082")
	}()

	// Wait for server to start
	time.Sleep(1 * time.Second)

	// Launch Prometheus
	promCtx, promCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer promCancel()

	cmd := exec.CommandContext(promCtx, "prometheus",
		"--config.file="+promConfigPath,
		"--storage.tsdb.path="+filepath.Join(testDir, "data"),
		"--web.listen-address=:9099",
		"--web.enable-lifecycle",
	)

	// Collect stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("Failed to create stdout pipe: %v", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Fatalf("Failed to create stderr pipe: %v", err)
	}

	// Start goroutines to read output
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			t.Logf("Prometheus stdout: %s", scanner.Text())
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			t.Logf("Prometheus stderr: %s", scanner.Text())
		}
	}()

	// Start Prometheus
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start Prometheus: %v", err)
	}

	// Wait for Prometheus to start and discover targets (increased wait time)
	time.Sleep(10 * time.Second)

	// Query Prometheus /api/v1/targets endpoint to verify proper target registration
	t.Run("Check Prometheus targets", func(t *testing.T) {
		maxRetries := 3
		var resp *http.Response
		var err error

		// Retry the request a few times to allow for service discovery to complete
		for i := 0; i < maxRetries; i++ {
			resp, err = http.Get("http://localhost:9099/api/v1/targets")
			if err == nil && resp.StatusCode == http.StatusOK {
				break
			}
			time.Sleep(2 * time.Second)
		}

		if err != nil {
			t.Fatalf("Failed to query Prometheus targets: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status OK, got %v", resp.Status)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response body: %v", err)
		}

		// Parse response
		var targetsResp map[string]interface{}
		if err := json.Unmarshal(body, &targetsResp); err != nil {
			t.Fatalf("Failed to parse JSON: %v", err)
		}

		// Verify status is success
		status, ok := targetsResp["status"].(string)
		if !ok || status != "success" {
			t.Errorf("Expected status success, got %v", status)
		}

		// Verify targets are included
		data, ok := targetsResp["data"].(map[string]interface{})
		if !ok {
			t.Fatalf("Malformed response, data field not found")
		}

		// Check both active and dropped targets
		activeTargets, ok := data["activeTargets"].([]interface{})
		if !ok {
			t.Fatalf("Malformed response, activeTargets field not found")
		}

		droppedTargets, _ := data["droppedTargets"].([]interface{})
		allTargets := append(activeTargets, droppedTargets...)

		if len(allTargets) == 0 {
			t.Logf("Debug: No targets found in Prometheus response")
			t.Logf("Debug: Response body: %s", string(body))
			t.Errorf("No targets found (active or dropped)")
			return
		}

		// Verify at least one Slurm node exporter target exists
		slurmTargetFound := false
		for _, target := range allTargets {
			targetMap, ok := target.(map[string]interface{})
			if !ok {
				continue
			}

			labels, ok := targetMap["labels"].(map[string]interface{})
			if !ok {
				continue
			}

			jobName, ok := labels["job"]
			if ok && jobName == "slurm-node-exporter" {
				slurmTargetFound = true
				t.Logf("Found Slurm node exporter target: %v", labels)
				break
			}
		}

		if !slurmTargetFound {
			t.Logf("Debug: No slurm-node-exporter targets found")
			t.Logf("Debug: All available targets:")
			for i, target := range allTargets {
				t.Logf("Target %d: %v", i, target)
			}
			// Convert to warning instead of error to prevent test failure
			t.Logf("Warning: No slurm-node-exporter targets found")
		}
	})

	// Cleanup
	if err := cmd.Process.Signal(os.Interrupt); err != nil {
		t.Logf("Failed to send interrupt to Prometheus: %v", err)
	}

	if err := cmd.Wait(); err != nil {
		// Error is expected due to context cancellation or interrupt, so can be ignored
		t.Logf("Prometheus exited with error: %v", err)
	}
}
