#!/bin/bash
# Setup script for integration testing environment

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INTEGRATION_DIR="${SCRIPT_DIR}/../integration"

# Build Docker images first
echo "Building Docker images..."
cd "${INTEGRATION_DIR}" && docker-compose build

# Start Docker Compose environment
echo "Starting Docker Compose environment..."
cd "${INTEGRATION_DIR}" && docker-compose up -d

# Wait for all services to be healthy
echo "Waiting for services to be ready..."
for service in slurmctld slurmd slurmrestd prometheus prometheus-slurm-sd; do
  echo "Waiting for ${service}..."
  while true; do
    health_status=$(docker inspect --format='{{.State.Health.Status}}' "${service}" 2>/dev/null || echo "container not found")
    if [ "${health_status}" = "healthy" ]; then
      echo "${service} is healthy."
      break
    elif [ "${health_status}" = "container not found" ]; then
      echo "Container ${service} not found. Exiting."
      exit 1
    fi
    echo "Waiting for ${service} to be healthy. Current status: ${health_status}"
    sleep 5
  done
done

echo "Integration test environment is ready!"
echo "Prometheus UI: http://localhost:9090"
echo "Prometheus-Slurm-SD: http://localhost:8080/targets"
