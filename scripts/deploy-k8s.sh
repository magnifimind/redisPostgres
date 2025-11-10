#!/bin/bash

set -e

echo "=== Deploying to Kubernetes ==="

# Deploy PostgreSQL
echo "Deploying PostgreSQL..."
kubectl apply -f k8s/postgres/

# Wait for PostgreSQL to be ready
echo "Waiting for PostgreSQL to be ready..."
kubectl wait --for=condition=ready pod -l app=postgres --timeout=120s

# Deploy Redis
echo "Deploying Redis..."
kubectl apply -f k8s/redis/

# Wait for Redis to be ready
echo "Waiting for Redis to be ready..."
kubectl wait --for=condition=ready pod -l app=redis --timeout=60s

# Deploy Backend
echo "Deploying Backend..."
kubectl apply -f k8s/backend/

# Wait for Backend to be ready
echo "Waiting for Backend to be ready..."
kubectl wait --for=condition=ready pod -l app=backend --timeout=120s

# Deploy Frontend
echo "Deploying Frontend..."
kubectl apply -f k8s/frontend/

echo "=== Deployment Complete ==="
echo ""
echo "To check the status of your deployments:"
echo "  kubectl get pods"
echo "  kubectl get services"
echo ""
echo "To access RedisInsight UI:"
echo "  kubectl get service redisinsight"
echo ""
echo "To port-forward PostgreSQL:"
echo "  kubectl port-forward svc/postgres 5432:5432"
echo ""
echo "To port-forward Backend API:"
echo "  kubectl port-forward svc/backend 3000:3000"
echo ""
echo "To port-forward Frontend:"
echo "  kubectl port-forward svc/frontend 8080:80"
