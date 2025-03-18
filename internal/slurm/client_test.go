package slurm

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestClient_GetNodes(t *testing.T) {
	// Set up logger for tests
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError, // Only log errors during tests
	}))

	// Test cases
	tests := []struct {
		name           string
		responseStatus int
		responseBody   string
		wantErr        bool
		validateResp   func(*NodeInfoResponse) bool
	}{
		{
			name:           "successful response",
			responseStatus: http.StatusOK,
			responseBody: `{
				"nodes": [
					{
						"name": "node1",
						"address": "10.0.0.1",
						"hostname": "node1.example.com",
						"state": ["IDLE"],
						"partitions": ["compute"]
					},
					{
						"name": "node2",
						"address": "10.0.0.2",
						"hostname": "node2.example.com",
						"state": ["ALLOCATED"],
						"partitions": ["compute", "gpu"]
					}
				],
				"meta": {
					"slurm": {
						"cluster": "test-cluster",
						"release": "22.05.8",
						"version": {
							"major": "22",
							"minor": "05",
							"micro": "8"
						}
					}
				}
			}`,
			wantErr: false,
			validateResp: func(resp *NodeInfoResponse) bool {
				if len(resp.Nodes) != 2 {
					return false
				}
				if resp.Nodes[0].Name != "node1" || resp.Nodes[0].Address != "10.0.0.1" {
					return false
				}
				if len(resp.Nodes[0].State) != 1 || resp.Nodes[0].State[0] != "IDLE" {
					return false
				}
				if len(resp.Nodes[0].Partitions) != 1 || resp.Nodes[0].Partitions[0] != "compute" {
					return false
				}
				if resp.Nodes[1].Name != "node2" || resp.Nodes[1].Address != "10.0.0.2" {
					return false
				}
				if resp.Meta == nil || resp.Meta.Slurm == nil {
					return false
				}
				if resp.Meta.Slurm.Cluster != "test-cluster" {
					return false
				}
				if resp.Meta.Slurm.Version == nil {
					return false
				}
				if resp.Meta.Slurm.Version.Major != "22" {
					return false
				}
				return true
			},
		},
		{
			name:           "error response",
			responseStatus: http.StatusUnauthorized,
			responseBody: `{
				"errors": [
					{
						"description": "Authentication failed",
						"source": "slurm_rest_auth",
						"error": "Invalid token",
						"error_number": 401
					}
				]
			}`,
			wantErr: true,
			validateResp: func(resp *NodeInfoResponse) bool {
				return true // Not used in error case
			},
		},
		{
			name:           "malformed json",
			responseStatus: http.StatusOK,
			responseBody:   `{"nodes": [{"name": "node1",`,
			wantErr:        true,
			validateResp: func(resp *NodeInfoResponse) bool {
				return true // Not used in error case
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Check request path
				if r.URL.Path != "/slurm/v0.0.38/nodes/" {
					t.Errorf("Unexpected request path: %s", r.URL.Path)
				}

				// Check headers if authentication is used
				if r.Header.Get("X-SLURM-USER-NAME") != "testuser" {
					t.Errorf("Unexpected or missing username header")
				}
				if r.Header.Get("X-SLURM-USER-TOKEN") != "testtoken" {
					t.Errorf("Unexpected or missing token header")
				}

				// Set response status and body
				w.WriteHeader(tc.responseStatus)
				io.WriteString(w, tc.responseBody)
			}))
			defer server.Close()

			// Create client with test server URL
			client := NewClient(
				server.URL,
				"v0.0.38",
				"testuser",
				"testtoken",
				logger,
			)

			// Call method being tested
			resp, err := client.GetNodes(context.Background())

			// Check errors
			if (err != nil) != tc.wantErr {
				t.Errorf("GetNodes() error = %v, wantErr %v", err, tc.wantErr)
				return
			}

			// Validate response when no error
			if err == nil && !tc.validateResp(resp) {
				t.Errorf("GetNodes() got invalid response: %+v", resp)
			}
		})
	}
}

// Test with custom HTTP client timeout
func TestNewClient(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	client := NewClient(
		"http://example.com",
		"v0.0.38",
		"user",
		"token",
		logger,
	)

	if client.baseURL != "http://example.com" {
		t.Errorf("baseURL = %s, want http://example.com", client.baseURL)
	}
	if client.apiVersion != "v0.0.38" {
		t.Errorf("apiVersion = %s, want v0.0.38", client.apiVersion)
	}
	if client.username != "user" {
		t.Errorf("username = %s, want user", client.username)
	}
	if client.token != "token" {
		t.Errorf("token = %s, want token", client.token)
	}
	if client.httpClient == nil {
		t.Errorf("httpClient is nil")
	}
}
