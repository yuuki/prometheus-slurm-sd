package main

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// TestWithDockerCompose runs integration tests using Docker Compose environment
// with actual Slurm components (slurmctld, slurmd, slurmrestd) and Prometheus.
func TestWithDockerCompose(t *testing.T) {
	// Skip test in CI environment if needed or if Docker is not available
	if os.Getenv("SKIP_DOCKER_TESTS") != "" {
		t.Skip("Skipping Docker-based integration tests")
	}

	// Check if Docker is available
	if err := exec.Command("docker", "--version").Run(); err != nil {
		t.Skip("Docker not available, skipping integration test")
	}

	// Check if Docker Compose is available
	if err := exec.Command("docker-compose", "--version").Run(); err != nil {
		t.Skip("Docker Compose not available, skipping integration test")
	}

	// Get the root directory of the project
	rootDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Setup the integration test environment
	setupCmd := exec.Command(filepath.Join(rootDir, "tests/scripts/setup-integration.sh"))
	setupCmd.Stdout = os.Stdout
	setupCmd.Stderr = os.Stderr
	if err := setupCmd.Run(); err != nil {
		t.Fatalf("Failed to setup integration test environment: %v", err)
	}

	// Cleanup the integration test environment when the test is done
	defer func() {
		cleanupCmd := exec.Command(filepath.Join(rootDir, "tests/scripts/cleanup-integration.sh"))
		cleanupCmd.Stdout = os.Stdout
		cleanupCmd.Stderr = os.Stderr
		if err := cleanupCmd.Run(); err != nil {
			t.Logf("Failed to cleanup integration test environment: %v", err)
		}
	}()

	// Wait for the services to be fully initialized
	time.Sleep(10 * time.Second)

	// Test the prometheus-slurm-sd endpoints
	t.Run("Check targets endpoint", func(t *testing.T) {
		// Get target information from our service
		resp, err := http.Get("http://localhost:8080/targets")
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

		// Verify we have targets
		if len(targets) == 0 {
			t.Logf("No targets found in response: %s", string(body))
			t.Errorf("No target groups found")
		}

		// Log the targets for debugging
		t.Logf("Received targets: %s", string(body))
	})

	// Check job-specific targets
	t.Run("Check job-specific targets", func(t *testing.T) {
		resp, err := http.Get("http://localhost:8080/targets?prom_job=node")
		if err != nil {
			t.Fatalf("Failed to get node-specific targets: %v", err)
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

		// Log the targets for debugging
		t.Logf("Received job-specific targets: %s", string(body))
	})

	// Check Prometheus metrics endpoint
	t.Run("Check Prometheus metrics", func(t *testing.T) {
		// Allow time for Prometheus to scrape targets
		time.Sleep(20 * time.Second)

		// Query Prometheus for up metrics
		resp, err := http.Get("http://localhost:9090/api/v1/query?query=up")
		if err != nil {
			t.Fatalf("Failed to query Prometheus: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status OK, got %v", resp.Status)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response body: %v", err)
		}

		// Log the Prometheus response for debugging
		t.Logf("Prometheus response: %s", string(body))
	})
}
