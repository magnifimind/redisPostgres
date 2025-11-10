#!/bin/bash

set -e

REGISTRY="t5810.webcentricds.net"
BACKEND_IMAGE="${REGISTRY}/bitcoin-cache-backend:latest"
FRONTEND_IMAGE="${REGISTRY}/bitcoin-cache-frontend:latest"

echo "=== Pushing Docker Images to Registry ==="

# Push backend
echo "Pushing backend image..."
docker push ${BACKEND_IMAGE}

# Push frontend
echo "Pushing frontend image..."
docker push ${FRONTEND_IMAGE}

echo "=== Push Complete ==="
