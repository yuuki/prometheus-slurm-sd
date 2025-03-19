# prometheus-slurm-sd

A web server that fetches node information from Slurm REST API and provides compatibility with Prometheus service discovery.

## Overview

`prometheus-slurm-sd` is a connector component that makes Slurm partition configurations available to Prometheus via service discovery. It periodically retrieves node information from Slurm clusters and exposes HTTP endpoints in Prometheus service discovery format.

### Architecture

![prometheus-slurm-sd architecture](docs/architecture.drawio.svg)

For detailed architecture documentation, see [here](docs/architecture.md).

### Features

- Retrieves node information using Slurm REST API
- Supports Prometheus [HTTP Service Discovery](https://prometheus.io/docs/prometheus/latest/http_sd/)
- Supports multiple exporter types
- Supports JWT authentication

## Installation

### Installing from Binary

Download the binary from [Releases](https://github.com/yuuki/prometheus-slurm-sd/releases).

### Building from Source

Clone the repository and build using Go or Make:

```bash
# Clone the repository
git clone https://github.com/yuuki/prometheus-slurm-sd.git
cd prometheus-slurm-sd

# Option 1: Using Go directly
go build

# Option 2: Using Make
make build
```

### Using Docker

The application can be built and run using Docker:

#### Building the Docker Image

```bash
# Build using the script
./scripts/build-docker.sh

# Or using Make
make docker

# Specify registry and version
./scripts/build-docker.sh --registry your-registry.com --version 1.0.0

# Or via Make with environment variables
DOCKER_REGISTRY=your-registry.com/ DOCKER_TAG=1.0.0 make docker
```

#### Running with Docker

```bash
# Run the container
docker run -p 8080:8080 -v $(pwd)/config.yaml:/app/config.yaml prometheus-slurm-sd

# Or using docker-compose
docker-compose up -d
```

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage
```

### Available Make Targets

| Target | Description |
|--------|-------------|
| `build` | Build the application |
| `test` | Run tests |
| `test-coverage` | Run tests with coverage report |
| `clean` | Remove build artifacts |
| `fmt` | Format code using gofmt |
| `lint` | Run linter |
| `run` | Build and run the application |
| `vet` | Run go vet |
| `docker` | Build Docker image |
| `docker-push` | Push Docker image to registry |

## Usage

### Configuration

Example configuration file (YAML):

```yaml
# Slurm REST API settings
slurm_api_endpoint: "http://slurm-restd:6820"
slurm_api_version: "v0.0.38"
slurm_api_username: "username"  # if needed
slurm_api_token: "token"  # if needed

# Web server settings
listen_address: ":8080"

# Update interval
update_interval: "5m"

# Exporter job settings
jobs:
  - name: node
    port: 9100
  - name: dcgm
    port: 9401
```

### Starting

```bash
./prometheus-slurm-sd --config.file=/path/to/config.yaml
```

### Command-line Options

| Option | Description | Default |
|--------|-------------|---------|
| `--config.file` | Configuration file path | `config.yaml` |
| `--log.level` | Log level (debug, info, warn, error) | `info` |
| `--web.listen-address` | Address to listen on for HTTP requests | Value from config file |
| `--slurm.api-endpoint` | Slurm REST API endpoint | Value from config file |
| `--slurm.api-version` | Slurm REST API version | Value from config file |
| `--slurm.api-username` | Slurm REST API username | Value from config file |
| `--slurm.api-token` | Slurm REST API token | Value from config file |
| `--update.interval` | Update interval for fetching Slurm data | Value from config file |

### Prometheus Configuration

Add HTTP Service Discovery configuration to your Prometheus configuration file (prometheus.yml):

```yaml
scrape_configs:
  - job_name: 'slurm-node-exporter'
    http_sd_configs:
      - url: http://prometheus-slurm-sd:8080/targets?prom_job=node
        refresh_interval: 5m
```

## API

### GET /targets

Returns targets in Prometheus service discovery format.

Query parameters:
- `prom_job`: Specific job name (optional, returns all jobs if omitted)

Example response:

```json
[
  {
    "targets": ["node1:9100", "node2:9100"],
    "labels": {
      "__meta_slurm_partition": "partition1",
      "__meta_slurm_job": "node"
    }
  }
]
```

### GET /health

Health check endpoint. Returns `OK` if the server is running.

## License

MIT
