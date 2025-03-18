package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alecthomas/kingpin/v2"

	"github.com/yuuki/prometheus-slurm-sd/internal/config"
	"github.com/yuuki/prometheus-slurm-sd/internal/discovery"
	"github.com/yuuki/prometheus-slurm-sd/internal/slurm"
)

var (
	version = "dev"
)

func main() {
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

	kingpin.MustParse(app.Parse(os.Args[1:]))

	// Configure logger
	var level slog.Level
	switch *logLevel {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	}))

	// Load configuration file
	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		logger.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	// Override with CLI settings
	if *listenAddress != "" {
		cfg.ListenAddress = *listenAddress
	}
	if *slurmApiEndpoint != "" {
		cfg.SlurmAPIEndpoint = *slurmApiEndpoint
	}
	if *slurmApiVersion != "" {
		cfg.SlurmAPIVersion = *slurmApiVersion
	}
	if *slurmApiUsername != "" {
		cfg.SlurmAPIUsername = *slurmApiUsername
	}
	if *slurmApiToken != "" {
		cfg.SlurmAPIToken = *slurmApiToken
	}
	if *updateInterval != "" {
		cfg.UpdateInterval = *updateInterval
	}

	// Validate configuration
	if cfg.SlurmAPIEndpoint == "" {
		logger.Error("Slurm API endpoint is required")
		os.Exit(1)
	}

	// Create Slurm client
	slurmClient := slurm.NewClient(
		cfg.SlurmAPIEndpoint,
		cfg.SlurmAPIVersion,
		cfg.SlurmAPIUsername,
		cfg.SlurmAPIToken,
		logger,
	)

	// Create service discovery service
	discoveryService, err := discovery.NewService(slurmClient, cfg, logger)
	if err != nil {
		logger.Error("Failed to create discovery service", "error", err)
		os.Exit(1)
	}

	// Set up context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		logger.Info("Received signal, shutting down", "signal", sig)
		cancel()
	}()

	// Start periodic update process
	go func() {
		if err := discoveryService.Start(ctx); err != nil && err != context.Canceled {
			logger.Error("Discovery service error", "error", err)
			cancel()
		}
	}()

	// Set up HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/targets", discoveryService.HTTPHandler())

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK")
	})

	server := &http.Server{
		Addr:    cfg.ListenAddress,
		Handler: mux,
	}

	// Start HTTP server
	go func() {
		logger.Info("Starting HTTP server", "address", cfg.ListenAddress)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server error", "error", err)
			cancel()
		}
	}()

	// Wait for shutdown
	<-ctx.Done()

	// Graceful shutdown of HTTP server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown error", "error", err)
	}

	logger.Info("Server stopped")
}
