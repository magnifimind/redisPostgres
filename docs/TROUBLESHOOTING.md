# Troubleshooting Guide

## Common Issues and Solutions

### Deployment Issues

#### Pods stuck in Pending state

**Symptom**:
```bash
$ kubectl get pods
NAME                        READY   STATUS    RESTARTS   AGE
postgres-xxx                0/1     Pending   0          5m
```

**Possible Causes**:
1. Insufficient resources
2. PersistentVolume not available
3. Storage class not found

**Solutions**:

Check pod events:
```bash
kubectl describe pod postgres-xxx
```

Check PV/PVC status:
```bash
kubectl get pv
kubectl get pvc
```

For local clusters (minikube, kind), ensure PV directory exists:
```bash
# On minikube
minikube ssh
sudo mkdir -p /mnt/data/postgres
exit
```

Check node resources:
```bash
kubectl describe nodes
```

---

#### Pods stuck in ImagePullBackOff

**Symptom**:
```bash
$ kubectl get pods
NAME                        READY   STATUS             RESTARTS   AGE
backend-xxx                 0/1     ImagePullBackOff   0          2m
```

**Possible Causes**:
1. Image doesn't exist in registry
2. Registry authentication failed
3. Wrong image name/tag

**Solutions**:

Check pod events:
```bash
kubectl describe pod backend-xxx
```

Verify images exist:
```bash
docker images | grep bitcoin-cache
```

Verify registry access:
```bash
docker login t5810.webcentricds.net
```

Check image names in deployments:
```bash
kubectl get deployment backend -o yaml | grep image:
```

Push images if missing:
```bash
./scripts/push.sh
```

---

#### Pods in CrashLoopBackOff

**Symptom**:
```bash
$ kubectl get pods
NAME                        READY   STATUS             RESTARTS   AGE
backend-xxx                 0/1     CrashLoopBackOff   5          5m
```

**Possible Causes**:
1. Application error on startup
2. Cannot connect to dependencies
3. Configuration error

**Solutions**:

Check logs:
```bash
kubectl logs backend-xxx
kubectl logs backend-xxx --previous  # Previous crash
```

Common backend errors:

**"Failed to connect to database"**:
```bash
# Check if postgres is running
kubectl get pods -l app=postgres

# Check postgres logs
kubectl logs -l app=postgres

# Verify service exists
kubectl get svc postgres
```

**"Failed to connect to Redis"**:
```bash
# Check if redis is running
kubectl get pods -l app=redis

# Check redis logs
kubectl logs -l app=redis

# Verify service exists
kubectl get svc redis
```

**Configuration issue**:
```bash
# Check configmap
kubectl get configmap backend-config -o yaml

# Check secret
kubectl get secret backend-secret -o yaml
```

---

### Connection Issues

#### Cannot access frontend via port-forward

**Symptom**:
```bash
$ kubectl port-forward svc/frontend 8080:80
Forwarding from 127.0.0.1:8080 -> 80
# Browser shows connection refused
```

**Solutions**:

Verify frontend pods are running:
```bash
kubectl get pods -l app=frontend
```

Check if pods are ready:
```bash
kubectl get pods -l app=frontend
# READY should show 1/1
```

Check frontend logs:
```bash
kubectl logs -l app=frontend
```

Try accessing the service directly:
```bash
kubectl get svc frontend
```

Test with a different port:
```bash
kubectl port-forward svc/frontend 8081:80
```

---

#### Backend returns 500 errors

**Symptom**:
API requests return Internal Server Error

**Solutions**:

Check backend logs:
```bash
kubectl logs -f -l app=backend
```

Test database connection:
```bash
kubectl port-forward svc/postgres 5432:5432

# In another terminal
psql -h localhost -p 5432 -U postgres -d bitcoin_db
```

Test Redis connection:
```bash
kubectl port-forward svc/redis 6379:6379

# In another terminal
redis-cli -h localhost -p 6379
> PING
```

Check service DNS:
```bash
kubectl run -it --rm debug --image=busybox --restart=Never -- sh
# Inside pod:
nslookup postgres
nslookup redis
```

---

#### Frontend can't connect to backend

**Symptom**:
Frontend shows "Failed to fetch bitcoins"

**Solutions**:

Check browser console for CORS errors

Verify backend service is accessible:
```bash
kubectl get svc backend
kubectl port-forward svc/backend 3000:3000

# In another terminal
curl http://localhost:3000/health
```

Check frontend environment:
```bash
kubectl get configmap frontend-config -o yaml
```

For LoadBalancer setup, verify external IP:
```bash
kubectl get svc backend
# Check EXTERNAL-IP column
```

Update frontend to point to correct backend URL

---

### Database Issues

#### PostgreSQL won't start

**Symptom**:
```bash
kubectl logs -l app=postgres
# Shows errors about data directory
```

**Solutions**:

Check PVC is bound:
```bash
kubectl get pvc postgres-pvc
# STATUS should be Bound
```

Check PV permissions:
```bash
# On minikube
minikube ssh
ls -la /mnt/data/postgres
# Should be owned by postgres (999:999)
```

Delete and recreate PV/PVC:
```bash
kubectl delete -f k8s/postgres/deployment.yaml
kubectl delete -f k8s/postgres/persistent-volume-claim.yaml
kubectl delete -f k8s/postgres/persistent-volume.yaml

# Recreate
kubectl apply -f k8s/postgres/persistent-volume.yaml
kubectl apply -f k8s/postgres/persistent-volume-claim.yaml
kubectl apply -f k8s/postgres/deployment.yaml
```

---

#### Database connection refused

**Symptom**:
Backend logs show "connection refused" for PostgreSQL

**Solutions**:

Verify PostgreSQL is listening:
```bash
kubectl exec -it postgres-xxx -- psql -U postgres -d bitcoin_db -c "SELECT version();"
```

Check PostgreSQL logs:
```bash
kubectl logs -l app=postgres
```

Verify service and endpoints:
```bash
kubectl get svc postgres
kubectl get endpoints postgres
```

Test from within cluster:
```bash
kubectl run -it --rm psql --image=postgres:15-alpine --restart=Never -- \
  psql -h postgres -U postgres -d bitcoin_db
```

---

#### Database schema not initialized

**Symptom**:
Backend logs show table doesn't exist

**Solutions**:

Check if init script ran:
```bash
kubectl logs -l app=postgres | grep init.sql
```

Manually run initialization:
```bash
kubectl exec -it postgres-xxx -- psql -U postgres -d bitcoin_db

# Inside psql:
\dt  # Should show 'bitcoins' table
```

If table doesn't exist, run init script:
```bash
kubectl cp k8s/postgres/init.sql postgres-xxx:/tmp/init.sql
kubectl exec -it postgres-xxx -- psql -U postgres -d bitcoin_db -f /tmp/init.sql
```

---

### Redis Issues

#### Redis connection timeout

**Symptom**:
Backend logs show Redis connection timeout

**Solutions**:

Check Redis is running:
```bash
kubectl get pods -l app=redis
kubectl logs -l app=redis
```

Test Redis connectivity:
```bash
kubectl port-forward svc/redis 6379:6379

# In another terminal
redis-cli -h localhost -p 6379
> PING
# Should return PONG
```

Check Redis service:
```bash
kubectl get svc redis
kubectl get endpoints redis
```

---

#### Cache not working (always MISS)

**Symptom**:
Backend logs show "Cache MISS" on every request

**Solutions**:

Verify Redis is storing data:
```bash
kubectl port-forward svc/redis 6379:6379

redis-cli -h localhost -p 6379
> KEYS bitcoin:*
> GET bitcoin:BTC
```

Check TTL settings in backend code

Restart backend to trigger cache priming:
```bash
kubectl rollout restart deployment backend
kubectl logs -f -l app=backend | grep "Cache priming"
```

---

### RedisInsight Issues

#### Cannot access RedisInsight UI

**Symptom**:
Cannot load http://localhost:5540

**Solutions**:

Check if RedisInsight is running:
```bash
kubectl get pods -l app=redisinsight
kubectl logs -l app=redisinsight
```

Check service type:
```bash
kubectl get svc redisinsight
# Should show LoadBalancer or NodePort
```

For LoadBalancer, get external IP:
```bash
kubectl get svc redisinsight -o wide
```

For local clusters, use port-forward:
```bash
kubectl port-forward svc/redisinsight 5540:5540
```

Access at: http://localhost:5540

---

#### Cannot connect to Redis from RedisInsight

**Symptom**:
RedisInsight shows connection error

**Solutions**:

In RedisInsight, use cluster DNS name:
- Host: `redis` (not localhost)
- Port: `6379`
- Name: `Bitcoin Cache`

If still failing, check network policies:
```bash
kubectl get networkpolicies
```

Test Redis from RedisInsight pod:
```bash
kubectl exec -it redisinsight-xxx -- sh
# Inside pod:
apk add redis
redis-cli -h redis -p 6379
> PING
```

---

### Performance Issues

#### Slow API responses

**Possible Causes**:
1. Cache not working
2. Database slow queries
3. Network latency

**Solutions**:

Check cache hit rate:
```bash
kubectl logs -f -l app=backend | grep -E "Cache (HIT|MISS)" | uniq -c
```

Check database query performance:
```bash
kubectl exec -it postgres-xxx -- psql -U postgres -d bitcoin_db

# Enable query timing
\timing

# Run test query
SELECT * FROM bitcoins ORDER BY price DESC;
```

Check pod resource usage:
```bash
kubectl top pods
```

Scale up if needed:
```bash
kubectl scale deployment backend --replicas=4
```

---

#### High memory usage

**Symptom**:
Pods getting OOMKilled

**Solutions**:

Check memory limits:
```bash
kubectl describe deployment backend | grep -A 5 Limits
```

Increase limits:
```bash
kubectl edit deployment backend
# Increase memory limits
```

Check for memory leaks:
```bash
kubectl logs -f -l app=backend
# Look for unusual patterns
```

---

### Build Issues

#### Docker build fails

**Symptom**:
```
ERROR: failed to solve: failed to resolve source metadata
```

**Solutions**:

Check Dockerfile syntax:
```bash
cd backend
docker build -t test .
```

Ensure go.mod exists:
```bash
ls -la backend/go.mod
```

Update Go dependencies:
```bash
cd backend
go mod tidy
go mod download
```

---

#### Frontend build fails

**Symptom**:
```
npm ERR! code ELIFECYCLE
```

**Solutions**:

Check package.json:
```bash
cd frontend
cat package.json
```

Clear npm cache:
```bash
cd frontend
rm -rf node_modules package-lock.json
npm install
```

Build locally first:
```bash
cd frontend
npm run build
```

---

### Helm Issues

#### Helm install fails

**Symptom**:
```
Error: failed to create resource
```

**Solutions**:

Validate Helm chart:
```bash
helm lint ./helm/bitcoin-cache
```

Dry run to see what would be created:
```bash
helm install bitcoin-cache ./helm/bitcoin-cache --dry-run --debug
```

Check template rendering:
```bash
helm template bitcoin-cache ./helm/bitcoin-cache
```

---

## Debugging Commands Reference

### Pod Debugging

```bash
# List all pods
kubectl get pods

# Describe pod
kubectl describe pod <pod-name>

# Get logs
kubectl logs <pod-name>
kubectl logs -f <pod-name>  # Follow
kubectl logs <pod-name> --previous  # Previous instance

# Execute command in pod
kubectl exec -it <pod-name> -- sh

# Port forward
kubectl port-forward <pod-name> 8080:80
```

### Service Debugging

```bash
# List services
kubectl get svc

# Describe service
kubectl describe svc <service-name>

# Check endpoints
kubectl get endpoints <service-name>
```

### Deployment Debugging

```bash
# List deployments
kubectl get deployments

# Describe deployment
kubectl describe deployment <deployment-name>

# Check rollout status
kubectl rollout status deployment <deployment-name>

# View rollout history
kubectl rollout history deployment <deployment-name>

# Rollback
kubectl rollout undo deployment <deployment-name>
```

### Resource Debugging

```bash
# Check resource usage
kubectl top pods
kubectl top nodes

# Get all resources
kubectl get all

# Check events
kubectl get events --sort-by='.lastTimestamp'
```

---

## Getting Help

If you're still stuck:

1. **Check Logs**: Always start with pod logs
2. **Check Events**: `kubectl get events`
3. **Describe Resources**: `kubectl describe <resource>`
4. **Test Connectivity**: Use debug pods
5. **Check Documentation**: Review README and architecture docs

### Debug Pod

Run a debug pod for network testing:

```bash
kubectl run -it --rm debug \
  --image=nicolaka/netshoot \
  --restart=Never -- bash

# Inside pod, test services:
curl http://postgres:5432
curl http://redis:6379
curl http://backend:3000/health
```

### Clean Slate

If all else fails, delete and redeploy:

```bash
./scripts/cleanup.sh kubectl
./scripts/deploy-k8s.sh
```
