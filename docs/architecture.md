# prometheus-slurm-sd Architecture

`prometheus-slurm-sd` is a service discovery component that connects Slurm clusters with Prometheus monitoring system.

## Basic Architecture

![Architecture Diagram](architecture.drawio.svg)

## Main Components

### Prometheus Server
- Uses HTTP Service Discovery to retrieve target information
- Sends requests to prometheus-slurm-sd at configured intervals (typically every few minutes)

### prometheus-slurm-sd
- Acts as a web server that handles responses to Prometheus and fetches information from Slurm
- Main functions:
  - Periodically retrieves node information from Slurm REST API
  - Converts data to Prometheus Service Discovery format
  - Caches JSON data in memory
  - Provides data to Prometheus through the `/targets` endpoint

### Slurm REST API (slurmrestd)
- Provides resource information about Slurm cluster nodes and partitions
- Publishes REST API conforming to OpenAPI specification
- Supports JWT authentication when required

## Operational Flow

1. **Startup and Periodic Updates**:
   - prometheus-slurm-sd starts based on configuration file
   - Fetches Slurm node information at configured intervals (default: 5 minutes)

2. **Data Retrieval and Conversion**:
   - Calls Slurm REST API (`GET /slurm/{version}/nodes/`) to retrieve node information
   - Converts data to Prometheus Service Discovery format (JSON list)
   - Groups nodes by partition
   - Assigns port numbers according to configured job types

3. **Caching and Response**:
   - Caches converted JSON data in memory
   - Returns cached data in response to Prometheus requests (`GET /targets`)
   - Can filter specific job types based on query parameters (`prom_job`)

## Design Benefits

- **Separation of Concerns**: No direct integration required between monitoring system (Prometheus) and cluster management system (Slurm)
- **Efficient Caching**: Prevents excessive requests to Slurm API
- **Flexible Configuration**: Supports different exporter types (node_exporter, DCGM, etc.)
- **Low Overhead**: Minimizes resource consumption with simple design
