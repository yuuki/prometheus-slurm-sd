#!/bin/bash
set -e

# Default values
IMAGE_NAME="prometheus-slurm-sd"
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
REGISTRY=""

# Argument parsing
while (( "$#" )); do
  case "$1" in
    --version)
      VERSION="$2"
      shift 2
      ;;
    --registry)
      REGISTRY="$2"
      shift 2
      ;;
    --name)
      IMAGE_NAME="$2"
      shift 2
      ;;
    *)
      echo "Unknown option: $1"
      exit 1
      ;;
  esac
done

# If a registry is specified, add it
if [ -n "$REGISTRY" ]; then
  if [[ "$REGISTRY" != */ ]]; then
    REGISTRY="${REGISTRY}/"
  fi
fi

FULL_IMAGE_NAME="${REGISTRY}${IMAGE_NAME}"

echo "Building Docker image: ${FULL_IMAGE_NAME}:${VERSION}"

# Build Docker image
docker build -t "${FULL_IMAGE_NAME}:${VERSION}" .
docker tag "${FULL_IMAGE_NAME}:${VERSION}" "${FULL_IMAGE_NAME}:latest"

echo "Successfully built images:"
echo "  - ${FULL_IMAGE_NAME}:${VERSION}"
echo "  - ${FULL_IMAGE_NAME}:latest"

echo ""
echo "To push these images, run:"
echo "  docker push ${FULL_IMAGE_NAME}:${VERSION}"
echo "  docker push ${FULL_IMAGE_NAME}:latest"
