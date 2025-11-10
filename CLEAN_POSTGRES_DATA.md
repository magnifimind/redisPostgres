# Instructions for Linux Claude: Clean Postgres Data

The Postgres pod needs a clean data directory before we can redeploy with Helm.

## Please run these commands:

```bash
# Clean the postgres data directory
minikube ssh "sudo rm -rf /tmp/postgres-data && sudo mkdir -p /tmp/postgres-data && sudo chmod 777 /tmp/postgres-data"

# Verify it's empty
minikube ssh "ls -la /tmp/postgres-data"
```

## After running, report back:

Confirm that:
1. The directory was cleaned
2. The `ls -la` shows an empty directory (only . and .. entries)
