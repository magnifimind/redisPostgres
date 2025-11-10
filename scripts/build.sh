#!/bin/bash

set -e

REGISTRY="t5810.webcentricds.net"
BACKEND_IMAGE="${REGISTRY}/bitcoin-cache-backend:latest"
FRONTEND_IMAGE="${REGISTRY}/bitcoin-cache-frontend:latest"

echo "=== Building Docker Images ==="

# Build backend
echo "Building backend image..."
docker build -t ${BACKEND_IMAGE} ./backend

# Build frontend
echo "Building frontend image..."
docker build -t ${FRONTEND_IMAGE} \
  --build-arg REACT_APP_API_URL=http://localhost:3000 \
  ./frontend

echo "=== Build Complete ==="
echo "Backend:  ${BACKEND_IMAGE}"
echo "Frontend: ${FRONTEND_IMAGE}"
