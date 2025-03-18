package slurm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// Client is the Slurm REST API client
type Client struct {
	baseURL    string
	apiVersion string
	username   string
	token      string
	httpClient *http.Client
	logger     *slog.Logger
}

// NodeInfoResponse represents the Slurm node information response
type NodeInfoResponse struct {
	Nodes      []Node     `json:"nodes"`
	LastUpdate *TimeValue `json:"last_update,omitempty"`
	Meta       *Meta      `json:"meta,omitempty"`
	Errors     []Error    `json:"errors,omitempty"`
	Warnings   []Warning  `json:"warnings,omitempty"`
}

// Node represents Slurm node information
type Node struct {
	Name       string   `json:"name"`
	Address    string   `json:"address"`
	Hostname   string   `json:"hostname"`
	State      []string `json:"state"`
	Partitions []string `json:"partitions"`
	// Add other necessary fields
}

// TimeValue represents a Slurm timestamp value
type TimeValue struct {
	Number   int64 `json:"number"`
	Set      bool  `json:"set"`
	Infinite bool  `json:"infinite"`
}

// Meta represents Slurm metadata information
type Meta struct {
	Slurm  *SlurmInfo  `json:"slurm,omitempty"`
	Plugin *PluginInfo `json:"plugin,omitempty"`
	Client *ClientInfo `json:"client,omitempty"`
}

// SlurmInfo represents Slurm version information
type SlurmInfo struct {
	Cluster string       `json:"cluster"`
	Release string       `json:"release"`
	Version *VersionInfo `json:"version"`
}

// VersionInfo represents detailed version information
type VersionInfo struct {
	Major string `json:"major"`
	Minor string `json:"minor"`
	Micro string `json:"micro"`
}

// PluginInfo represents plugin information
type PluginInfo struct {
	AccountingStorage string `json:"accounting_storage"`
	Name              string `json:"name"`
	Type              string `json:"type"`
	DataParser        string `json:"data_parser"`
}

// ClientInfo represents client information
type ClientInfo struct {
	Source string `json:"source"`
	User   string `json:"user"`
	Group  string `json:"group"`
}

// Error represents error information
type Error struct {
	Description string `json:"description"`
	Source      string `json:"source"`
	Error       string `json:"error"`
	ErrorNumber int    `json:"error_number"`
}

// Warning represents warning information
type Warning struct {
	Description string `json:"description"`
	Source      string `json:"source"`
}

// NewClient creates a new Slurm client
func NewClient(baseURL, apiVersion, username, token string, logger *slog.Logger) *Client {
	return &Client{
		baseURL:    baseURL,
		apiVersion: apiVersion,
		username:   username,
		token:      token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// GetNodes retrieves Slurm node information
func (c *Client) GetNodes(ctx context.Context) (*NodeInfoResponse, error) {
	endpoint := fmt.Sprintf("%s/slurm/%s/nodes/", c.baseURL, c.apiVersion)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add JWT authentication headers
	if c.username != "" {
		req.Header.Set("X-SLURM-USER-NAME", c.username)
	}
	if c.token != "" {
		req.Header.Set("X-SLURM-USER-TOKEN", c.token)
	}

	c.logger.Debug("Requesting Slurm nodes", "url", endpoint)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var nodeInfo NodeInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&nodeInfo); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &nodeInfo, nil
}
