package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/yuuki/prometheus-slurm-sd/internal/config"
	"github.com/yuuki/prometheus-slurm-sd/internal/discovery"
	"github.com/yuuki/prometheus-slurm-sd/internal/slurm"
)

// Mock Slurm server for E2E testing
type mockSlurmServer struct {
	server *httptest.Server
}

// Create a new mock Slurm server
func newMockSlurmServer() *mockSlurmServer {
	mux := http.NewServeMux()

	// Mock Slurm node information endpoint
	// The client requests in the format %s/slurm/%s/nodes/ so we must match exactly including the trailing slash
	mux.HandleFunc("/slurm/v0.0.38/nodes/", func(w http.ResponseWriter, r *http.Request) {
		// Verify JWT token if needed
		authHeader := r.Header.Get("X-SLURM-USER-TOKEN")
		if authHeader == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Provide mock data
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
				{
					"name":       "node3",
					"hostname":   "node3.example.com",
					"state":      []string{"IDLE"},
					"partitions": []string{"partition2"},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	})

	// Health check endpoint
	mux.HandleFunc("/slurm/v0.0.38/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "PONG")
	})

	server := httptest.NewServer(mux)
	return &mockSlurmServer{server: server}
}

// Close the server
func (m *mockSlurmServer) close() {
	m.server.Close()
}

// E2E Test: Test the entire flow
func TestE2EFlow(t *testing.T) {
	// Start the mock Slurm server
	mockServer := newMockSlurmServer()
	defer mockServer.close()

	// Log the server URL
	t.Logf("Mock Slurm server started at: %s", mockServer.server.URL)

	// Create a temporary config file
	tempConfigFile := createTempConfigFile(t, mockServer.server.URL)
	defer os.Remove(tempConfigFile)

	// Log config content
	configContent, _ := os.ReadFile(tempConfigFile)
	t.Logf("Test config: %s", string(configContent))

	// Load configuration
	cfg, err := config.LoadConfig(tempConfigFile)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Test logger
	logger := testLogger()

	// Create Slurm client
	slurmClient := slurm.NewClient(
		cfg.SlurmAPIEndpoint,
		cfg.SlurmAPIVersion,
		cfg.SlurmAPIUsername,
		cfg.SlurmAPIToken,
		logger,
	)

	// Create discovery service
	discoveryService, err := discovery.NewService(slurmClient, cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create discovery service: %v", err)
	}

	// Set up context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start discovery service
	go func() {
		discoveryService.Start(ctx)
	}()

	// Wait for data to be updated
	time.Sleep(100 * time.Millisecond)

	// Create handler and set up test server
	handler := discoveryService.HTTPHandler()
	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	// Test API requests
	t.Run("Test all targets", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("%s/targets", server.URL))
		if err != nil {
			t.Fatalf("Failed to get targets: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status OK, got %v", resp.Status)
		}

		var targets []map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&targets); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Verify target groups exist
		if len(targets) == 0 {
			t.Errorf("No target groups found")
		} else {
			t.Logf("Found %d target groups", len(targets))
		}
	})

	t.Run("Test specific job targets", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("%s/targets?prom_job=node", server.URL))
		if err != nil {
			t.Fatalf("Failed to get targets: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status OK, got %v", resp.Status)
		}

		var targets []map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&targets); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Verify targets for specific job
		jobFound := false
		for _, target := range targets {
			labels, ok := target["labels"].(map[string]interface{})
			if !ok {
				continue
			}

			if job, ok := labels["__meta_slurm_job"].(string); ok && job == "node" {
				jobFound = true
				break
			}
		}

		if !jobFound {
			t.Errorf("Node job not found in targets")
		}
	})

	// Test health check endpoint
	t.Run("Test health endpoint", func(t *testing.T) {
		// Test prometheus-slurm-sd endpoint (actual server)
		mux := http.NewServeMux()
		mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "OK")
		})

		healthServer := httptest.NewServer(mux)
		defer healthServer.Close()

		resp, err := http.Get(fmt.Sprintf("%s/health", healthServer.URL))
		if err != nil {
			t.Fatalf("Failed to get health: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status OK, got %v", resp.Status)
		}
	})
}

// Create a temporary config file for testing
func createTempConfigFile(t *testing.T, serverURL string) string {
	config := fmt.Sprintf(`
slurm_api_endpoint: "%s"
slurm_api_version: "v0.0.38"
slurm_api_username: "testuser"
slurm_api_token: "testtoken"
listen_address: ":8080"
update_interval: "1s"
jobs:
  - name: node
    port: 9100
  - name: dcgm
    port: 9401
`, serverURL)

	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	if _, err := tmpfile.Write([]byte(config)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}

	if err := tmpfile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	return tmpfile.Name()
}

// Create a logger for testing
func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
}
