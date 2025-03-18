# API Reference

This document describes the API endpoints provided by prometheus-slurm-sd.

## Overview

prometheus-slurm-sd provides API endpoints that comply with the Prometheus HTTP Service Discovery standard. These APIs return target information that allows Prometheus to monitor nodes in Slurm clusters.

## Endpoints

### GET /targets

Returns a list of targets in Prometheus Service Discovery format.

#### Request Parameters

| Parameter | Description | Required | Default |
|-----------|-------------|----------|---------|
| `prom_job` | Filter by specific job name | No | None (returns all jobs) |

#### Response

- Content-Type: `application/json`
- Status Code: 200 OK

#### Response Format

```json
[
  {
    "targets": ["<hostname>:<port>", ...],
    "labels": {
      "<label_name>": "<label_value>",
      ...
    }
  },
  ...
]
```

#### Response Example

```json
[
  {
    "targets": ["node1:9100", "node2:9100", "node3:9100", "node4:9100"],
    "labels": {
      "__meta_slurm_partition": "compute",
      "__meta_slurm_job": "node"
    }
  },
  {
    "targets": ["gpu1:9100", "gpu2:9100"],
    "labels": {
      "__meta_slurm_partition": "gpu",
      "__meta_slurm_job": "node"
    }
  }
]
```

Example of filtering by a specific job:

```
GET /targets?prom_job=node
```

#### Error Response

In case of server error:

- Status Code: 500 Internal Server Error
- Content-Type: `text/plain`
- Response Body: `Internal server error`

### GET /health

Health check endpoint. Used to verify that the server is running.

#### Request Parameters

None

#### Response

- Content-Type: `text/plain`
- Status Code: 200 OK
- Response Body: `OK`

## Integration with Prometheus

Configure the following in your Prometheus configuration file (prometheus.yml):

```yaml
scrape_configs:
  - job_name: 'slurm-node-exporter'
    http_sd_configs:
      - url: http://prometheus-slurm-sd:8080/targets?prom_job=node
        refresh_interval: 5m
```

For multiple job types, configure each as a separate job:

```yaml
scrape_configs:
  - job_name: 'slurm-node-exporter'
    http_sd_configs:
      - url: http://prometheus-slurm-sd:8080/targets?prom_job=node
        refresh_interval: 5m

  - job_name: 'slurm-dcgm-exporter'
    http_sd_configs:
      - url: http://prometheus-slurm-sd:8080/targets?prom_job=dcgm
        refresh_interval: 5m
```

## Labels

prometheus-slurm-sd provides the following labels:

| Label | Description |
|-------|-------------|
| `__meta_slurm_partition` | Slurm partition name that the node belongs to |
| `__meta_slurm_job` | Job name defined in the configuration |

These labels can be used in Prometheus relabel_configs to label and filter targets:

```yaml
scrape_configs:
  - job_name: 'slurm-node-exporter'
    http_sd_configs:
      - url: http://prometheus-slurm-sd:8080/targets?prom_job=node
    relabel_configs:
      - source_labels: [__meta_slurm_partition]
        target_label: slurm_partition
      - source_labels: [__meta_slurm_job]
        target_label: slurm_job
```
