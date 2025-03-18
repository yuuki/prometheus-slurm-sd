# Configuration Reference

This document explains the configuration options for prometheus-slurm-sd.

## Configuration File

prometheus-slurm-sd uses a YAML format configuration file. By default, it reads `config.yaml` from the current directory, but you can specify a different file using the `--config.file` command-line option.

### Basic Configuration

```yaml
# Slurm REST API settings
slurm_api_endpoint: "http://slurm-restd:6820"  # Required: Slurm REST API endpoint
slurm_api_version: "v0.0.38"                   # Optional: API version (default: "v0.0.38")
slurm_api_username: "username"                 # Optional: Username for JWT authentication
slurm_api_token: "token"                       # Optional: Token for JWT authentication

# Web server settings
listen_address: ":8080"                        # Optional: Web server listen address (default: ":8080")

# Update interval
update_interval: "5m"                          # Optional: Slurm data update interval (default: "5m")

# Exporter job settings
jobs:                                          # Required: At least one job configuration is required
  - name: node                                 # Required: Job name
    port: 9100                                 # Required: Exporter port number
  - name: dcgm
    port: 9401
```

### Configuration Options Detail

#### Slurm API Settings

| Option | Description | Required | Default |
|--------|-------------|----------|---------|
| `slurm_api_endpoint` | Slurm REST API URL endpoint | Yes | None |
| `slurm_api_version` | Slurm REST API version | No | `"v0.0.38"` |
| `slurm_api_username` | Username for JWT authentication | No | None |
| `slurm_api_token` | Token for JWT authentication | No | None |

#### Web Server Settings

| Option | Description | Required | Default |
|--------|-------------|----------|---------|
| `listen_address` | Web server listen address (IP address and port) | No | `":8080"` |

#### Update Settings

| Option | Description | Required | Default |
|--------|-------------|----------|---------|
| `update_interval` | Slurm data update interval (Go language Duration format) | No | `"5m"` |

#### Job Settings

The `jobs` section requires at least one job configuration with the following settings:

| Option | Description | Required | Default |
|--------|-------------|----------|---------|
| `name` | Job name (used for the `prom_job` URL parameter) | Yes | None |
| `port` | Exporter port number | Yes | None |

## Command-line Options

You can use command-line options to override values from the configuration file.

| Option | Description | Default |
|--------|-------------|---------|
| `--config.file` | Configuration file path | `config.yaml` |
| `--log.level` | Log level (debug, info, warn, error) | `info` |
| `--web.listen-address` | Address to listen on for HTTP requests | Value from config file |
| `--slurm.api-endpoint` | Slurm REST API endpoint | Value from config file |
| `--slurm.api-version` | Slurm REST API version | Value from config file |
| `--slurm.api-username` | Slurm REST API username | Value from config file |
| `--slurm.api-token` | Slurm REST API token | Value from config file |
| `--update.interval` | Slurm data fetch interval | Value from config file |

## Configuration Examples

### Basic Configuration

```yaml
slurm_api_endpoint: "http://slurm-restd:6820"
listen_address: ":8080"
update_interval: "5m"
jobs:
  - name: node
    port: 9100
```

### Configuration with JWT Authentication

```yaml
slurm_api_endpoint: "http://slurm-restd:6820"
slurm_api_version: "v0.0.38"
slurm_api_username: "prometheus"
slurm_api_token: "your-jwt-token"
listen_address: ":8080"
update_interval: "5m"
jobs:
  - name: node
    port: 9100
  - name: dcgm
    port: 9401
```

### Multiple Exporter Configuration

```yaml
slurm_api_endpoint: "http://slurm-restd:6820"
listen_address: ":8080"
update_interval: "5m"
jobs:
  - name: node
    port: 9100
  - name: dcgm
    port: 9401
  - name: process
    port: 9256
  - name: ipmi
    port: 9290
  - name: lustre
    port: 9169
```
