# Instructions for Linux Claude

## Problem
Minikube cannot pull images from the private registry at `t5810.webcentricds.net` due to SSL certificate verification issues.

## Solution
Restart minikube with the `--insecure-registry` flag to trust the registry.

## Commands to Run

```bash
# Stop minikube
minikube stop

# Start minikube with insecure registry configured
minikube start --insecure-registry="t5810.webcentricds.net"

# Verify the configuration
minikube ssh -- cat /etc/docker/daemon.json

# Check that minikube is ready
kubectl get nodes
```

## What to Report Back
1. Confirm minikube restarted successfully
2. Show the output of `minikube ssh -- cat /etc/docker/daemon.json` to verify the insecure registry is configured
3. Show `kubectl get nodes` to confirm the cluster is healthy

## After This is Done
The Mac Claude will delete and recreate the backend pods to trigger image pulls with the new configuration.
