#!/bin/bash
# Cleanup script for integration testing environment

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INTEGRATION_DIR="${SCRIPT_DIR}/../integration"

# Stop and remove Docker Compose environment
echo "Stopping Docker Compose environment..."
cd "${INTEGRATION_DIR}" && docker-compose down -v

echo "Integration test environment has been cleaned up."
