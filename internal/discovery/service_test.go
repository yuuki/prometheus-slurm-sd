package discovery

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/yuuki/prometheus-slurm-sd/internal/config"
	"github.com/yuuki/prometheus-slurm-sd/internal/slurm"
)

// MockSlurmClient is a mock implementation of the Slurm client for testing
type MockSlurmClient struct {
	GetNodesFunc func(ctx context.Context) (*slurm.NodeInfoResponse, error)
}

// GetNodes is the mock implementation of GetNodes
func (m *MockSlurmClient) GetNodes(ctx context.Context) (*slurm.NodeInfoResponse, error) {
	return m.GetNodesFunc(ctx)
}

func TestService_updateTargets(t *testing.T) {
	// Setup logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	// Test cases
	tests := []struct {
		name             string
		cfg              *config.Config
		mockResponse     *slurm.NodeInfoResponse
		mockError        error
		expectedJobCount int
		validateTargets  func(map[string][]PrometheusTarget) bool
	}{
		{
			name: "successful update with multiple partitions",
			cfg: &config.Config{
				UpdateInterval: "5m",
				Jobs: []config.JobConfig{
					{Name: "node", Port: 9100},
					{Name: "gpu", Port: 9400},
				},
			},
			mockResponse: &slurm.NodeInfoResponse{
				Nodes: []slurm.Node{
					{
						Name:       "node1",
						Address:    "10.0.0.1",
						Hostname:   "node1.example.com",
						State:      []string{"IDLE"},
						Partitions: []string{"compute"},
					},
					{
						Name:       "node2",
						Address:    "10.0.0.2",
						Hostname:   "node2.example.com",
						State:      []string{"ALLOCATED"},
						Partitions: []string{"compute", "gpu"},
					},
					{
						Name:       "node3",
						Address:    "",
						Hostname:   "node3.example.com",
						State:      []string{"DRAIN"},
						Partitions: []string{"compute"},
					},
				},
			},
			mockError:        nil,
			expectedJobCount: 2,
			validateTargets: func(targets map[string][]PrometheusTarget) bool {
				// Check node job targets
				nodeTargets, ok := targets["node"]
				if !ok {
					return false
				}

				// Check if there's at least one target for node job
				nodeTargetCount := 0
				for _, target := range nodeTargets {
					nodeTargetCount += len(target.Targets)
				}
				if nodeTargetCount < 2 {
					return false
				}

				// Check gpu job targets
				gpuTargets, ok := targets["gpu"]
				if !ok {
					return false
				}

				// Check if there's at least one target for gpu job
				gpuTargetCount := 0
				for _, target := range gpuTargets {
					gpuTargetCount += len(target.Targets)
				}
				if gpuTargetCount < 2 {
					return false
				}

				return true
			},
		},
		{
			name: "empty node list",
			cfg: &config.Config{
				UpdateInterval: "5m",
				Jobs: []config.JobConfig{
					{Name: "node", Port: 9100},
				},
			},
			mockResponse: &slurm.NodeInfoResponse{
				Nodes: []slurm.Node{},
			},
			mockError:        nil,
			expectedJobCount: 1,
			validateTargets: func(targets map[string][]PrometheusTarget) bool {
				nodeTargets, ok := targets["node"]
				return ok && len(nodeTargets) == 0
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock client
			mockClient := &MockSlurmClient{
				GetNodesFunc: func(ctx context.Context) (*slurm.NodeInfoResponse, error) {
					return tc.mockResponse, tc.mockError
				},
			}

			// Create service with mock client
			service, err := NewService(mockClient, tc.cfg, logger)
			if err != nil {
				t.Fatalf("Failed to create service: %v", err)
			}

			// Call the method being tested
			err = service.updateTargets(context.Background())

			// Verify error handling
			if tc.mockError != nil && err == nil {
				t.Errorf("Expected error, got nil")
			}
			if tc.mockError == nil && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Verify targets cache if no error
			if err == nil {
				service.targetsCacheMutex.RLock()
				defer service.targetsCacheMutex.RUnlock()

				if len(service.targetsCache) != tc.expectedJobCount {
					t.Errorf("Expected %d job targets, got %d", tc.expectedJobCount, len(service.targetsCache))
				}

				if !tc.validateTargets(service.targetsCache) {
					t.Errorf("Target validation failed, cache: %+v", service.targetsCache)
				}
			}
		})
	}
}

func TestService_HTTPHandler(t *testing.T) {
	// Setup logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	// Test configuration
	cfg := &config.Config{
		UpdateInterval: "5m",
		Jobs: []config.JobConfig{
			{Name: "node", Port: 9100},
			{Name: "gpu", Port: 9400},
		},
	}

	// Create mock Slurm client
	mockClient := &MockSlurmClient{
		GetNodesFunc: func(ctx context.Context) (*slurm.NodeInfoResponse, error) {
			// Return a predefined response for testing
			return &slurm.NodeInfoResponse{
				Nodes: []slurm.Node{
					{
						Name:       "node1",
						Address:    "10.0.0.1",
						Hostname:   "node1.example.com",
						State:      []string{"IDLE"},
						Partitions: []string{"compute"},
					},
					{
						Name:       "node2",
						Address:    "10.0.0.2",
						Hostname:   "node2.example.com",
						State:      []string{"ALLOCATED"},
						Partitions: []string{"compute", "gpu"},
					},
				},
			}, nil
		},
	}

	// Create service with mock client
	service, err := NewService(mockClient, cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Manually update targets
	err = service.updateTargets(context.Background())
	if err != nil {
		t.Fatalf("Failed to update targets: %v", err)
	}

	// Test HTTP handler cases
	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		validateResp   func([]PrometheusTarget) bool
	}{
		{
			name:           "get all targets",
			queryParams:    "",
			expectedStatus: http.StatusOK,
			validateResp: func(targets []PrometheusTarget) bool {
				return len(targets) > 0
			},
		},
		{
			name:           "get node job targets",
			queryParams:    "?prom_job=node",
			expectedStatus: http.StatusOK,
			validateResp: func(targets []PrometheusTarget) bool {
				if len(targets) == 0 {
					return false
				}
				for _, target := range targets {
					if target.Labels["__meta_slurm_job"] != "node" {
						return false
					}
				}
				return true
			},
		},
		{
			name:           "job not found",
			queryParams:    "?prom_job=nonexistent",
			expectedStatus: http.StatusOK,
			validateResp: func(targets []PrometheusTarget) bool {
				return len(targets) == 0
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a request with test parameters
			req := httptest.NewRequest("GET", "/targets"+tc.queryParams, nil)

			// Add Prometheus header to test logging
			req.Header.Set("X-Prometheus-Refresh-Interval-Seconds", "300")

			// Create a response recorder
			rr := httptest.NewRecorder()

			// Call the handler
			handler := service.HTTPHandler()
			handler(rr, req)

			// Check status code
			if status := rr.Code; status != tc.expectedStatus {
				t.Errorf("Handler returned wrong status code: got %v want %v", status, tc.expectedStatus)
			}

			// Check content type
			if ctype := rr.Header().Get("Content-Type"); ctype != "application/json" {
				t.Errorf("Handler returned wrong content type: got %v want application/json", ctype)
			}

			// Parse response body
			var targets []PrometheusTarget
			if err := json.NewDecoder(rr.Body).Decode(&targets); err != nil {
				t.Errorf("Failed to decode response: %v", err)
				return
			}

			// Validate response
			if !tc.validateResp(targets) {
				t.Errorf("Response validation failed, targets: %+v", targets)
			}
		})
	}
}

func TestService_Start(t *testing.T) {
	// Setup logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	// Test configuration with short update interval
	cfg := &config.Config{
		UpdateInterval: "10ms", // Very short for testing
		Jobs: []config.JobConfig{
			{Name: "test", Port: 9100},
		},
	}

	// Create counter for calls to GetNodes
	callCount := 0

	// Create mock Slurm client
	mockClient := &MockSlurmClient{
		GetNodesFunc: func(ctx context.Context) (*slurm.NodeInfoResponse, error) {
			callCount++
			return &slurm.NodeInfoResponse{
				Nodes: []slurm.Node{
					{
						Name:       "node1",
						Address:    "10.0.0.1",
						State:      []string{"IDLE"},
						Partitions: []string{"compute"},
					},
				},
			}, nil
		},
	}

	// Create service with mock client
	service, err := NewService(mockClient, cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Create a context with cancel function
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the service in a goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- service.Start(ctx)
	}()

	// Wait a bit for multiple updates to happen
	time.Sleep(50 * time.Millisecond)

	// Cancel the context to stop the service
	cancel()

	// Wait for service to stop
	err = <-errCh
	if err != nil && err != context.Canceled {
		t.Errorf("Service.Start() returned unexpected error: %v", err)
	}

	// Check that GetNodes was called at least twice
	if callCount < 2 {
		t.Errorf("Expected multiple calls to GetNodes, got %d", callCount)
	}
}
