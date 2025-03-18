package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/yuuki/prometheus-slurm-sd/internal/config"
	"github.com/yuuki/prometheus-slurm-sd/internal/slurm"
)

// PrometheusTarget represents a Prometheus service discovery target
type PrometheusTarget struct {
	Targets []string          `json:"targets"`
	Labels  map[string]string `json:"labels"`
}

// Service is the Prometheus service discovery service
type Service struct {
	slurmClient *slurm.Client
	config      *config.Config
	logger      *slog.Logger

	targetsCache      map[string][]PrometheusTarget
	targetsCacheMutex sync.RWMutex
	updateInterval    time.Duration
}

// NewService creates a new service discovery service
func NewService(slurmClient *slurm.Client, cfg *config.Config, logger *slog.Logger) (*Service, error) {
	updateInterval, err := time.ParseDuration(cfg.UpdateInterval)
	if err != nil {
		return nil, fmt.Errorf("invalid update interval: %w", err)
	}

	return &Service{
		slurmClient:    slurmClient,
		config:         cfg,
		logger:         logger,
		targetsCache:   make(map[string][]PrometheusTarget),
		updateInterval: updateInterval,
	}, nil
}

// Start initiates the service discovery service
func (s *Service) Start(ctx context.Context) error {
	// Initial fetch
	if err := s.updateTargets(ctx); err != nil {
		s.logger.Error("Failed to update targets on startup", "error", err)
	}

	// Periodic update process
	ticker := time.NewTicker(s.updateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := s.updateTargets(ctx); err != nil {
				s.logger.Error("Failed to update targets", "error", err)
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// updateTargets fetches node information from Slurm and updates the target cache
func (s *Service) updateTargets(ctx context.Context) error {
	// Call Slurm API to get node information
	nodeInfo, err := s.slurmClient.GetNodes(ctx)
	if err != nil {
		return fmt.Errorf("failed to get nodes from Slurm: %w", err)
	}

	// Group nodes by partition
	partitionNodes := make(map[string][]slurm.Node)
	for _, node := range nodeInfo.Nodes {
		for _, partition := range node.Partitions {
			partitionNodes[partition] = append(partitionNodes[partition], node)
		}
	}

	// Generate targets for each job
	jobTargets := make(map[string][]PrometheusTarget)
	for _, job := range s.config.Jobs {
		var targets []PrometheusTarget

		for partition, nodes := range partitionNodes {
			var nodeTargets []string

			for _, node := range nodes {
				// Check node status (include only active nodes)
				isActive := false
				for _, state := range node.State {
					if state == "IDLE" || state == "ALLOCATED" || state == "MIXED" {
						isActive = true
						break
					}
				}

				if isActive {
					nodeAddress := node.Address
					if nodeAddress == "" {
						nodeAddress = node.Hostname
					}

					// Add port to the target
					nodeTargets = append(nodeTargets, fmt.Sprintf("%s:%d", nodeAddress, job.Port))
				}
			}

			if len(nodeTargets) > 0 {
				// Create PrometheusTarget
				target := PrometheusTarget{
					Targets: nodeTargets,
					Labels: map[string]string{
						"__meta_slurm_partition": partition,
						"__meta_slurm_job":       job.Name,
					},
				}
				targets = append(targets, target)
			}
		}

		jobTargets[job.Name] = targets
	}

	// Update cache
	s.targetsCacheMutex.Lock()
	s.targetsCache = jobTargets
	s.targetsCacheMutex.Unlock()

	s.logger.Info("Updated targets cache", "jobs", len(s.config.Jobs))
	return nil
}

// GetTargets returns targets for the specified job
func (s *Service) GetTargets(jobName string) ([]PrometheusTarget, bool) {
	s.targetsCacheMutex.RLock()
	defer s.targetsCacheMutex.RUnlock()

	targets, ok := s.targetsCache[jobName]
	return targets, ok
}

// GetAllTargets returns all job targets
func (s *Service) GetAllTargets() []PrometheusTarget {
	s.targetsCacheMutex.RLock()
	defer s.targetsCacheMutex.RUnlock()

	var allTargets []PrometheusTarget
	for _, targets := range s.targetsCache {
		allTargets = append(allTargets, targets...)
	}

	return allTargets
}

// HTTPHandler is the handler for Prometheus HTTP Service Discovery requests
func (s *Service) HTTPHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Log refresh interval header
		if refreshInterval := r.Header.Get("X-Prometheus-Refresh-Interval-Seconds"); refreshInterval != "" {
			s.logger.Debug("Received Prometheus refresh interval", "seconds", refreshInterval)
		}

		// Return targets for a specific job if job parameter exists
		jobName := r.URL.Query().Get("prom_job")

		var targets []PrometheusTarget
		if jobName != "" {
			if jobTargets, ok := s.GetTargets(jobName); ok {
				targets = jobTargets
			} else {
				// Return empty list if job doesn't exist
				targets = []PrometheusTarget{}
			}
		} else {
			// Return all targets if no job specified
			targets = s.GetAllTargets()
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(targets); err != nil {
			s.logger.Error("Failed to encode targets", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}
}
