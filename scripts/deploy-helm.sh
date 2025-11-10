#!/bin/bash

set -e

RELEASE_NAME="bitcoin-cache"
NAMESPACE="default"

echo "=== Deploying with Helm ==="

# Install or upgrade the Helm chart
helm upgrade --install ${RELEASE_NAME} ./helm/bitcoin-cache \
  --namespace ${NAMESPACE} \
  --create-namespace \
  --wait \
  --timeout 5m

echo "=== Deployment Complete ==="
echo ""
echo "Release: ${RELEASE_NAME}"
echo "Namespace: ${NAMESPACE}"
echo ""
echo "To check the status:"
echo "  helm status ${RELEASE_NAME}"
echo "  kubectl get pods -n ${NAMESPACE}"
echo ""
echo "To access RedisInsight UI:"
echo "  kubectl get service redisinsight -n ${NAMESPACE}"
echo ""
echo "To port-forward PostgreSQL:"
echo "  kubectl port-forward svc/postgres 5432:5432 -n ${NAMESPACE}"
echo ""
echo "To port-forward Backend API:"
echo "  kubectl port-forward svc/backend 3000:3000 -n ${NAMESPACE}"
echo ""
echo "To access Frontend (if LoadBalancer):"
echo "  kubectl get service frontend -n ${NAMESPACE}"
