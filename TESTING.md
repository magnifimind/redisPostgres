# Bitcoin Cache Manager - Testing Guide

This document provides comprehensive testing instructions for the Bitcoin Cache Manager application, which demonstrates Redis sorted sets for caching with PostgreSQL as the source of truth.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Deployment](#deployment)
3. [Testing the Application](#testing-the-application)
4. [Testing Cache Behavior](#testing-cache-behavior)
5. [Testing Data Persistence](#testing-data-persistence)
6. [Performance Testing](#performance-testing)
7. [Cleanup](#cleanup)

---

## Prerequisites

### Required Tools
- Kubernetes cluster (Minikube recommended for local testing)
- kubectl configured to access your cluster
- Helm 3.x
- Docker
- curl (for API testing)
- Optional: Redis CLI for direct cache inspection

### Environment Setup

For testing on a remote Minikube cluster (like T5810):
```bash
export KUBECONFIG=~/.kube/config-t5810
```

For local Minikube:
```bash
export KUBECONFIG=~/.kube/config
```

---

## Deployment

### 1. Deploy with Helm

```bash
# Deploy to the ranking-scaled namespace
cd /path/to/redisPostgres
helm install bitcoin-cache ./helm/bitcoin-cache

# Or upgrade if already installed
helm upgrade bitcoin-cache ./helm/bitcoin-cache
```

### 2. Verify Deployment

```bash
# Check all pods are running
kubectl get pods -n ranking-scaled

# Expected output:
# NAME                        READY   STATUS    RESTARTS   AGE
# backend-xxx-yyy            1/1     Running   0          2m
# backend-xxx-zzz            1/1     Running   0          2m
# frontend-xxx-yyy           1/1     Running   0          2m
# frontend-xxx-zzz           1/1     Running   0          2m
# postgres-xxx-yyy           1/1     Running   0          2m
# redis-xxx-yyy              1/1     Running   0          2m
# redisinsight-xxx-yyy       1/1     Running   0          2m
```

### 3. Set Up Port Forwarding

```bash
# Backend API (in one terminal)
kubectl port-forward -n ranking-scaled svc/backend 3000:3000

# Frontend UI (in another terminal)
kubectl port-forward -n ranking-scaled svc/frontend 8080:80

# Optional: RedisInsight (in another terminal)
kubectl port-forward -n ranking-scaled svc/redisinsight 5540:5540
```

---

## Testing the Application

### Test 1: Access the UI

1. Open your browser to http://localhost:8080
2. You should see the "Bitcoin Cache Manager" interface
3. Verify the UI displays:
   - Form section for adding/updating bitcoins
   - Rankings list section

### Test 2: Create New Bitcoin Entries

#### Using the UI:
1. In the form, enter:
   - Symbol: `BTC`
   - Price: `65000`
2. Click "Save"
3. Verify success message appears
4. Verify BTC appears in the rankings list with rank #1

#### Using the API:
```bash
# Create BTC
curl -X POST http://localhost:3000/api/bitcoins \
  -H 'Content-Type: application/json' \
  -d '{"symbol":"BTC","price":65000}'

# Create ETH
curl -X POST http://localhost:3000/api/bitcoins \
  -H 'Content-Type: application/json' \
  -d '{"symbol":"ETH","price":3500}'

# Create DOGE
curl -X POST http://localhost:3000/api/bitcoins \
  -H 'Content-Type: application/json' \
  -d '{"symbol":"DOGE","price":50000}'

# Create BNB
curl -X POST http://localhost:3000/api/bitcoins \
  -H 'Content-Type: application/json' \
  -d '{"symbol":"BNB","price":450}'
```

**Expected Result:**
- Each create returns HTTP 201 with the created bitcoin
- Rankings update automatically showing:
  1. BTC - $65,000
  2. DOGE - $50,000
  3. ETH - $3,500
  4. BNB - $450

### Test 3: Update Existing Bitcoin

#### Using the UI:
1. Click the "Edit" button next to BTC
2. Form populates with BTC data (symbol disabled)
3. Change price to `70000`
4. Click "Update"
5. Verify success message: "Successfully updated BTC"
6. Verify ranking list shows new price immediately

#### Using the API:
```bash
# Update BTC price
curl -X POST http://localhost:3000/api/bitcoins \
  -H 'Content-Type: application/json' \
  -d '{"symbol":"BTC","price":75000}'

# Verify update
curl -s http://localhost:3000/api/bitcoins | python3 -m json.tool
```

**Expected Result:**
- `updated_at` timestamp changes
- `created_at` timestamp remains the same
- Rankings re-sort automatically
- UI refreshes with new price

### Test 4: Verify Rankings Integrity

```bash
# Get all bitcoins ranked by price
curl -s http://localhost:3000/api/bitcoins | python3 -m json.tool
```

**Verify:**
- Each bitcoin has a sequential rank (1, 2, 3, 4...)
- Ranks are in descending order by price
- No duplicate ranks
- No gaps in rank sequence

### Test 5: Delete Bitcoin

#### Using the UI:
1. Click "Delete" next to BNB
2. Confirm deletion in the dialog
3. Verify success message
4. Verify BNB is removed from the list
5. Verify ranks re-number automatically (1, 2, 3)

#### Using the API:
```bash
# Delete BNB
curl -X DELETE http://localhost:3000/api/bitcoins/BNB

# Verify deletion
curl -s http://localhost:3000/api/bitcoins | python3 -m json.tool
```

**Expected Result:**
- HTTP 200 with deleted bitcoin data
- Rankings automatically update
- Remaining bitcoins re-rank sequentially

### Test 6: Get Single Bitcoin

```bash
# Get specific bitcoin
curl -s http://localhost:3000/api/bitcoins/BTC | python3 -m json.tool

# Try non-existent bitcoin
curl -s http://localhost:3000/api/bitcoins/XYZ
```

**Expected Results:**
- Existing bitcoin returns HTTP 200 with data
- Non-existent bitcoin returns HTTP 404

---

## Testing Cache Behavior

### Test 7: Verify Cache Priming

```bash
# Check backend logs for cache priming
kubectl logs -n ranking-scaled -l app=backend --tail=100 | grep "Cache priming"

# Expected output:
# Starting cache priming...
# Cache priming completed: X bitcoins loaded into cache and sorted set
```

### Test 8: Verify Cache Hits and Misses

```bash
# First request (should be cache HIT from priming)
curl -s http://localhost:3000/api/bitcoins/BTC | python3 -m json.tool

# Check logs for cache hit
kubectl logs -n ranking-scaled -l app=backend --tail=20 | grep "Cache"

# Expected: "Cache HIT for BTC"
```

### Test 9: Verify Redis Sorted Set Usage

```bash
# Check logs for sorted set usage
kubectl logs -n ranking-scaled -l app=backend --tail=50 | grep "sorted set"

# Expected: "Rankings served from Redis sorted set (X bitcoins)"
```

### Test 10: Verify Write-Through Caching

```bash
# Update a bitcoin
curl -X POST http://localhost:3000/api/bitcoins \
  -H 'Content-Type: application/json' \
  -d '{"symbol":"ETH","price":4000}'

# Check logs for write-through
kubectl logs -n ranking-scaled -l app=backend --tail=20 | grep "Write-through"

# Expected: "Write-through completed for ETH (price: 4000)"
```

### Test 11: Inspect Redis Directly (Optional)

If you have RedisInsight running:
1. Open http://localhost:5540
2. Connect to redis:6379
3. Browse keys:
   - `bitcoin:BTC` - Individual bitcoin JSON cache
   - `bitcoin:rankings:sorted` - Sorted set for rankings
4. Verify sorted set scores match bitcoin prices

Or using Redis CLI:
```bash
# Connect to Redis pod
kubectl exec -it -n ranking-scaled deployment/redis -- redis-cli

# Check sorted set
ZREVRANGE bitcoin:rankings:sorted 0 -1 WITHSCORES

# Check individual cache
GET bitcoin:BTC

# Exit
exit
```

---

## Testing Data Persistence

### Test 12: PostgreSQL Persistence

```bash
# Connect to PostgreSQL
kubectl exec -it -n ranking-scaled deployment/postgres -- psql -U postgres -d bitcoin_db

# Query bitcoins table
SELECT symbol, price, created_at, updated_at FROM bitcoins ORDER BY price DESC;

# Exit
\q
```

**Verify:**
- All bitcoins are persisted in PostgreSQL
- Prices match current values
- Timestamps are correct

### Test 13: Pod Restart Persistence

```bash
# Get current bitcoin data
curl -s http://localhost:3000/api/bitcoins > /tmp/before.json

# Delete backend pod to trigger restart
kubectl delete pod -n ranking-scaled -l app=backend

# Wait for new pod to be ready
kubectl wait --for=condition=ready pod -l app=backend -n ranking-scaled --timeout=60s

# Get bitcoin data after restart
curl -s http://localhost:3000/api/bitcoins > /tmp/after.json

# Compare
diff /tmp/before.json /tmp/after.json
```

**Expected Result:**
- No differences in data
- Cache priming runs automatically on startup
- All rankings maintained correctly

### Test 14: Full Stack Restart

```bash
# Delete all pods
kubectl delete pods -n ranking-scaled --all

# Wait for all pods to be ready
kubectl wait --for=condition=ready pod --all -n ranking-scaled --timeout=120s

# Verify data still intact
curl -s http://localhost:3000/api/bitcoins | python3 -m json.tool
```

**Expected Result:**
- All bitcoin data persists
- Rankings correct after full restart
- PostgreSQL data survived pod restart

---

## Performance Testing

### Test 15: Bulk Create

```bash
# Create multiple bitcoins
for i in {1..10}; do
  curl -X POST http://localhost:3000/api/bitcoins \
    -H 'Content-Type: application/json' \
    -d "{\"symbol\":\"TEST$i\",\"price\":$((RANDOM % 100000))}"
  echo ""
done

# Verify all created
curl -s http://localhost:3000/api/bitcoins | python3 -m json.tool | grep -c "symbol"
```

### Test 16: Concurrent Requests

```bash
# Make 10 concurrent requests
for i in {1..10}; do
  curl -s http://localhost:3000/api/bitcoins > /dev/null &
done
wait

# Check backend logs for any errors
kubectl logs -n ranking-scaled -l app=backend --tail=50
```

**Expected Result:**
- All requests complete successfully
- No errors in logs
- Consistent rankings across all responses

### Test 17: Ranking Performance

```bash
# Time the rankings endpoint
time curl -s http://localhost:3000/api/bitcoins > /dev/null
```

**Expected Result:**
- Response time < 100ms for typical datasets
- Consistent performance regardless of database size

---

## Testing Edge Cases

### Test 18: Duplicate Symbol Handling

```bash
# Create BTC if it doesn't exist
curl -X POST http://localhost:3000/api/bitcoins \
  -H 'Content-Type: application/json' \
  -d '{"symbol":"BTC","price":65000}'

# Try to create BTC again with different price
curl -X POST http://localhost:3000/api/bitcoins \
  -H 'Content-Type: application/json' \
  -d '{"symbol":"BTC","price":75000}'

# Verify BTC was updated, not duplicated
curl -s http://localhost:3000/api/bitcoins | python3 -m json.tool | grep -A 3 "BTC"
```

**Expected Result:**
- No duplicate BTCs
- Price updated to 75000
- Only one BTC in rankings

### Test 19: Invalid Input Handling

```bash
# Try to create without symbol
curl -X POST http://localhost:3000/api/bitcoins \
  -H 'Content-Type: application/json' \
  -d '{"price":1000}'

# Try to create without price
curl -X POST http://localhost:3000/api/bitcoins \
  -H 'Content-Type: application/json' \
  -d '{"symbol":"TEST"}'

# Try to delete non-existent bitcoin
curl -X DELETE http://localhost:3000/api/bitcoins/NOTEXIST
```

**Expected Results:**
- HTTP 400 for missing required fields
- HTTP 404 for non-existent bitcoin
- Appropriate error messages

### Test 20: Price Update Rank Change

```bash
# Create bitcoins with known prices
curl -X POST http://localhost:3000/api/bitcoins -H 'Content-Type: application/json' -d '{"symbol":"A","price":1000}'
curl -X POST http://localhost:3000/api/bitcoins -H 'Content-Type: application/json' -d '{"symbol":"B","price":2000}'
curl -X POST http://localhost:3000/api/bitcoins -H 'Content-Type: application/json' -d '{"symbol":"C","price":3000}'

# Initial rankings: C(1), B(2), A(3)
curl -s http://localhost:3000/api/bitcoins | python3 -m json.tool

# Update A to highest price
curl -X POST http://localhost:3000/api/bitcoins -H 'Content-Type: application/json' -d '{"symbol":"A","price":5000}'

# Verify new rankings: A(1), C(2), B(3)
curl -s http://localhost:3000/api/bitcoins | python3 -m json.tool
```

**Expected Result:**
- Rankings automatically adjust
- A moves from rank 3 to rank 1
- All other ranks shift accordingly

---

## Health Checks

### Test 21: Health Endpoint

```bash
# Check backend health
curl http://localhost:3000/health

# Expected response:
# {"status":"healthy"}
```

### Test 22: Cache Stats

```bash
# Get Redis cache statistics
curl -s http://localhost:3000/api/cache/stats
```

**Expected Result:**
- Returns Redis INFO stats section
- Shows cache hit/miss ratios

---

## Cleanup

### Full Cleanup

```bash
# Uninstall Helm release
helm uninstall bitcoin-cache

# Verify all resources deleted (PVC should auto-delete)
kubectl get all,pvc -n ranking-scaled

# Optional: Delete namespace
kubectl delete namespace ranking-scaled
```

### Partial Cleanup (Keep Postgres Data)

```bash
# Delete only application pods
kubectl delete deployment -n ranking-scaled backend frontend redis redisinsight

# Keep postgres and PVC for data persistence
```

---

## Common Issues and Solutions

### Issue: Pods not starting
**Solution:**
```bash
# Check pod status
kubectl get pods -n ranking-scaled

# Check pod logs
kubectl logs -n ranking-scaled <pod-name>

# Check events
kubectl describe pod -n ranking-scaled <pod-name>
```

### Issue: Port forward fails
**Solution:**
```bash
# Kill existing port forwards
pkill -f "kubectl port-forward"

# Restart port forward
kubectl port-forward -n ranking-scaled svc/backend 3000:3000
```

### Issue: Rankings not updating
**Solution:**
```bash
# Check backend logs
kubectl logs -n ranking-scaled -l app=backend --tail=50

# Verify Redis is running
kubectl get pods -n ranking-scaled -l app=redis

# Verify sorted set operations in logs
kubectl logs -n ranking-scaled -l app=backend | grep "sorted set"
```

### Issue: Data not persisting
**Solution:**
```bash
# Check PostgreSQL pod
kubectl get pods -n ranking-scaled -l app=postgres

# Check PVC
kubectl get pvc -n ranking-scaled

# Verify database connection
kubectl logs -n ranking-scaled -l app=backend | grep "Connected to PostgreSQL"
```

---

## Success Criteria

A fully passing test suite should demonstrate:

1. **CRUD Operations**: Create, Read, Update, Delete all work correctly
2. **Cache Performance**: Cache hits on repeated reads
3. **Rankings**: Always sorted correctly by price in descending order
4. **Data Flow**: Updates propagate through UI → Backend → Redis → PostgreSQL
5. **Persistence**: Data survives pod restarts and full stack restarts
6. **Sorted Sets**: Rankings served from Redis sorted sets (O(log N) performance)
7. **Auto-Update**: Redis sorted set scores update automatically on price changes
8. **UI Responsiveness**: Edit functionality loads existing data, updates propagate immediately

---

## Advanced Testing

### Load Testing with Apache Bench

```bash
# Install Apache Bench if needed
# macOS: brew install httpd
# Ubuntu: sudo apt-get install apache2-utils

# Create test
ab -n 1000 -c 10 http://localhost:3000/api/bitcoins

# POST test (requires special setup)
echo '{"symbol":"TEST","price":1000}' > /tmp/post.json
ab -n 100 -c 5 -p /tmp/post.json -T application/json http://localhost:3000/api/bitcoins
```

### Stress Testing

```bash
# Create many bitcoins
for i in {1..100}; do
  curl -X POST http://localhost:3000/api/bitcoins \
    -H 'Content-Type: application/json' \
    -d "{\"symbol\":\"STRESS$i\",\"price\":$((RANDOM % 1000000))}" &
done
wait

# Verify all rankings correct
curl -s http://localhost:3000/api/bitcoins | python3 -m json.tool | grep rank
```

---

## Monitoring

### Watch Logs in Real-Time

```bash
# Backend logs
kubectl logs -n ranking-scaled -l app=backend -f

# All pods
kubectl logs -n ranking-scaled --all-containers=true -f
```

### Resource Usage

```bash
# Check resource usage
kubectl top pods -n ranking-scaled

# Check memory/CPU limits
kubectl describe deployments -n ranking-scaled
```
