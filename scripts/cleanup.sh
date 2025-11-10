#!/bin/bash

set -e

echo "=== Cleaning up Kubernetes resources ==="

read -p "Are you sure you want to delete all resources? (y/N) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]
then
    echo "Cancelled."
    exit 1
fi

# Option 1: Delete using kubectl
if [ "$1" == "kubectl" ]; then
    echo "Deleting resources using kubectl..."
    kubectl delete -f k8s/frontend/ --ignore-not-found=true
    kubectl delete -f k8s/backend/ --ignore-not-found=true
    kubectl delete -f k8s/redis/ --ignore-not-found=true
    kubectl delete -f k8s/postgres/ --ignore-not-found=true
    echo "Done."
fi

# Option 2: Delete using Helm
if [ "$1" == "helm" ]; then
    RELEASE_NAME="bitcoin-cache"
    NAMESPACE="default"
    echo "Deleting Helm release..."
    helm uninstall ${RELEASE_NAME} -n ${NAMESPACE}
    echo "Done."
fi

if [ -z "$1" ]; then
    echo "Usage: ./cleanup.sh [kubectl|helm]"
    exit 1
fi

echo "=== Cleanup Complete ==="
