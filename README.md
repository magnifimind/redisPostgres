# Bitcoin Cache System

A command-line utility for common development tasks with smart ignore pattern support.

*David Alexander English IS magnifimind!*

![David Alexander English](images/daveEnglish.png)

![David Alexander English](images/magnifiMind.jpg)

A production-ready caching system with Redis read-through/write-through capabilities backed by PostgreSQL, featuring a React frontend for managing Bitcoin entities.

## Code implementation

This covers a few more edge cases.
- Ranks bitcoin with same price as shared ranking
- Correctly sorts in TreeSet with record

```java
public class Symbols {
    record Bitcoin(String symbol, int value) implements Comparable<Bitcoin> {
        @Override
        public int compareTo(Bitcoin o) {
            int cmp = Integer.compare(o.value, this.value); // Descending by value
            return cmp != 0 ? cmp : this.symbol.compareTo(o.symbol); // Tie-break by symbol
        }
    }

    TreeSet<Bitcoin> bitcoins = new TreeSet<>();
    Map<String, Integer> symbolValueMap = new HashMap<>();
    Map<String, Integer> symbolRankMap = new HashMap<>();

    public void addSymbol(String symbol, int value) {
        var oldValue = symbolValueMap.get(symbol);
        if (oldValue != null) {
            bitcoins.remove(new Bitcoin(symbol, oldValue));
        }
        bitcoins.add(new Bitcoin(symbol, value));
        symbolValueMap.put(symbol, value);
        rebuildRank();
    }

    public int getValue(String symbol) {
        return symbolValueMap.getOrDefault(symbol, -1);
    }

    public int getRank(String symbol) {
        return symbolRankMap.getOrDefault(symbol, -1);
    }

    public List<String> getTopK(int k) {
        return bitcoins.stream().map(Bitcoin::symbol).limit(k).toList();
    }

    public void remove(String symbol) {
        var value = symbolValueMap.remove(symbol);
        if (value != null) {
            bitcoins.remove(new Bitcoin(symbol, value));
            symbolRankMap.remove(symbol);
            rebuildRank();
        }
    }

    private void rebuildRank() {
        symbolRankMap.clear();
        int rank = 0;
        Integer prevValue = Integer.MIN_VALUE;
        for (var bc : bitcoins) {
            if (prevValue != bc.value)
                rank++;
            symbolRankMap.put(bc.symbol, rank);
            prevValue = bc.value;
        }
    }
}
```

## Architecture Overview

![UI To add them](images/bitcoinCacheManager.png)

* CDN Caching here is an option, for additional overhead.
However, opted for redis only cache due to lower latency
for changes in price

```
┌─────────────┐     ┌──────────────┐     ┌─────────┐     ┌────────────┐
│   React     │────▶│  Go Backend  │────▶│  Redis  │────▶│ PostgreSQL │
│  Frontend   │     │  (Cache Svc) │     │  Cache  │     │   (SoT)    │
└─────────────┘     └──────────────┘     └─────────┘     └────────────┘
                            │
                            │
                    ┌───────▼──────────┐
                    │  RedisInsight UI │
                    └──────────────────┘
```

### Features

- **Cache Priming**: Automatically loads all data from PostgreSQL into Redis at startup
- **Read-Through Cache**: On cache miss, automatically fetches from PostgreSQL and caches the result
- **Write-Through Cache**: All writes go to both PostgreSQL and Redis simultaneously
- **Automatic Rankings**: Bitcoin entities are ranked by price in real-time
- **Persistent Storage**: PostgreSQL data persists across pod restarts using Kubernetes PV/PVC
- **Redis UI**: RedisInsight for direct Redis query and monitoring
- **Scalable**: Containerized and deployable via Kubernetes/Helm

## Quick Start

### Prerequisites

- Docker
- Kubernetes cluster (minikube, kind, or production cluster)
- kubectl
- Helm 3 (optional, for Helm deployment)
- Access to docker registry at `t5810.webcentricds.net`

### 1. Build Docker Images

```bash
./scripts/build.sh
```

This builds:
- `t5810.webcentricds.net/bitcoin-cache-backend:latest` (Go backend)
- `t5810.webcentricds.net/bitcoin-cache-frontend:latest` (React frontend)

### 2. Push to Registry

```bash
# Login to registry (if not already logged in)
docker login t5810.webcentricds.net

# Push images
./scripts/push.sh
```

### 3. Deploy to Kubernetes

#### Option A: Using kubectl

```bash
./scripts/deploy-k8s.sh
```

#### Option B: Using Helm

```bash
./scripts/deploy-helm.sh
```

### 4. Access the Application

#### Port Forward Backend API

```bash
kubectl port-forward svc/backend 3000:3000
```

Access API at: http://localhost:3000

#### Port Forward Frontend

```bash
kubectl port-forward svc/frontend 8080:80
```

Access UI at: http://localhost:8080

#### Access RedisInsight UI

```bash
# Get the service
kubectl get svc redisinsight

# If LoadBalancer, use external IP
# Otherwise, port forward:
kubectl port-forward svc/redisinsight 5540:5540
```

Access RedisInsight at: http://localhost:5540

To connect to Redis in RedisInsight:
- Host: `redis`
- Port: `6379`
- Database: `0`

#### Access PostgreSQL

```bash
kubectl port-forward svc/postgres 5432:5432
```

Connect using:
- Host: `localhost`
- Port: `5432`
- Database: `bitcoin_db`
- Username: `postgres`
- Password: `postgres`

```bash
psql -h localhost -p 5432 -U postgres -d bitcoin_db
```

## Project Structure

```
.
├── backend/                 # Go backend application
│   ├── main.go             # Main application with cache service
│   ├── go.mod              # Go dependencies
│   └── Dockerfile          # Multi-stage Docker build
│
├── frontend/               # React frontend application
│   ├── src/
│   │   ├── App.js         # Main React component
│   │   ├── App.css        # Styles
│   │   └── index.js       # Entry point
│   ├── public/
│   ├── package.json       # Node dependencies
│   ├── Dockerfile         # Multi-stage Docker build
│   └── nginx.conf         # Nginx configuration
│
├── k8s/                   # Kubernetes manifests
│   ├── postgres/          # PostgreSQL with PV/PVC
│   ├── redis/             # Redis + RedisInsight
│   ├── backend/           # Backend deployment
│   └── frontend/          # Frontend deployment
│
├── helm/                  # Helm chart
│   └── bitcoin-cache/
│       ├── Chart.yaml
│       ├── values.yaml
│       └── templates/
│
├── scripts/               # Deployment scripts
│   ├── build.sh          # Build Docker images
│   ├── push.sh           # Push to registry
│   ├── deploy-k8s.sh     # Deploy with kubectl
│   ├── deploy-helm.sh    # Deploy with Helm
│   └── cleanup.sh        # Cleanup resources
│
└── docs/                  # Additional documentation
    ├── API.md             # API documentation
    ├── ARCHITECTURE.md    # Architecture details
    └── TROUBLESHOOTING.md # Common issues
```

## Cache Strategy Details

### Cache Priming

On startup, the backend:
1. Connects to PostgreSQL
2. Queries all Bitcoin entries
3. Loads each entry into Redis with TTL
4. Logs the number of entries cached

**Code location**: `backend/main.go:PrimeCache()`

### Read-Through

When reading a Bitcoin:
1. Check Redis cache first
2. If **HIT**: Return cached data (fast path)
3. If **MISS**: Query PostgreSQL, cache result, return data

**Code location**: `backend/main.go:GetBitcoin()`

### Write-Through

When creating/updating a Bitcoin:
1. Write to PostgreSQL first (source of truth)
2. Write to Redis cache (synchronously)
3. Invalidate rankings cache
4. Return the saved data

**Code location**: `backend/main.go:SetBitcoin()`

### Cache Invalidation

- **Individual entries**: Invalidated on update/delete
- **Rankings**: Invalidated whenever any price changes
- **TTL**: All cache entries expire after 1 hour

## API Endpoints

### Health Check
```
GET /health
```

### Get All Bitcoins (Ranked)
```
GET /api/bitcoins
```

Response:
```json
[
  {
    "symbol": "BTC",
    "price": 65000,
    "rank": 1,
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
]
```

### Get Single Bitcoin
```
GET /api/bitcoins/:symbol
```

### Create/Update Bitcoin
```
POST /api/bitcoins
Content-Type: application/json

{
  "symbol": "BTC",
  "price": 65000
}
```

### Update Bitcoin
```
PUT /api/bitcoins/:symbol
Content-Type: application/json

{
  "price": 66000
}
```

### Delete Bitcoin
```
DELETE /api/bitcoins/:symbol
```

### Cache Stats
```
GET /api/cache/stats
```

See [docs/API.md](docs/API.md) for detailed API documentation.

## Configuration

### Backend Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `3000` | Server port |
| `POSTGRES_HOST` | `localhost` | PostgreSQL host |
| `POSTGRES_PORT` | `5432` | PostgreSQL port |
| `POSTGRES_DB` | `bitcoin_db` | Database name |
| `POSTGRES_USER` | `postgres` | Database user |
| `POSTGRES_PASSWORD` | `postgres` | Database password |
| `REDIS_HOST` | `localhost` | Redis host |
| `REDIS_PORT` | `6379` | Redis port |

### Kubernetes Configuration

Edit `k8s/*/configmap.yaml` and `k8s/*/secret.yaml` to customize settings.

### Helm Configuration

Edit `helm/bitcoin-cache/values.yaml`:

```yaml
registry: t5810.webcentricds.net

backend:
  replicas: 2
  resources:
    requests:
      memory: "128Mi"
      cpu: "100m"

postgres:
  storage:
    size: 5Gi
    storageClassName: manual
```

## Database Schema

```sql
CREATE TABLE bitcoins (
    symbol VARCHAR(10) PRIMARY KEY,
    price INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_bitcoin_price ON bitcoins(price DESC);
```

Sample data is automatically loaded on first startup:
- BTC: $65,000
- ETH: $3,500
- BNB: $450

## Monitoring

### Check Pod Status
```bash
kubectl get pods
```

### Check Logs

Backend:
```bash
kubectl logs -f -l app=backend
```

Frontend:
```bash
kubectl logs -f -l app=frontend
```

PostgreSQL:
```bash
kubectl logs -f -l app=postgres
```

Redis:
```bash
kubectl logs -f -l app=redis
```

### Cache Hit/Miss Monitoring

Watch backend logs for cache performance:
```bash
kubectl logs -f -l app=backend | grep -E "Cache (HIT|MISS)"
```

## Cleanup

```bash
# Using kubectl
./scripts/cleanup.sh kubectl

# Using Helm
./scripts/cleanup.sh helm
```

## Troubleshooting

### Pods not starting

Check pod status and events:
```bash
kubectl get pods
kubectl describe pod <pod-name>
```

### Cannot connect to PostgreSQL

Verify service and pod are running:
```bash
kubectl get svc postgres
kubectl get pod -l app=postgres
```

### Redis connection issues

Check Redis pod logs:
```bash
kubectl logs -l app=redis
```

### Backend can't connect to dependencies

Check backend logs for connection errors:
```bash
kubectl logs -l app=backend
```

Verify network policies and service names match configuration.

See [docs/TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md) for more details.

## Development

### Local Development (without Kubernetes)

1. Start PostgreSQL:
```bash
docker run -d --name postgres \
  -e POSTGRES_DB=bitcoin_db \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=postgres \
  -p 5432:5432 \
  postgres:15-alpine
```

2. Initialize database:
```bash
psql -h localhost -p 5432 -U postgres -d bitcoin_db < k8s/postgres/init.sql
```

3. Start Redis:
```bash
docker run -d --name redis -p 6379:6379 redis:7-alpine
```

4. Start backend:
```bash
cd backend
go run main.go
```

5. Start frontend:
```bash
cd frontend
npm install
npm start
```

### Running Tests

Backend:
```bash
cd backend
go test ./...
```

Frontend:
```bash
cd frontend
npm test
```

## Production Considerations

1. **Security**:
   - Change default PostgreSQL password
   - Use Kubernetes secrets for sensitive data
   - Enable TLS for all services
   - Implement network policies

2. **Scalability**:
   - Adjust replica counts based on load
   - Configure horizontal pod autoscaling
   - Use Redis cluster for high availability

3. **Monitoring**:
   - Set up Prometheus metrics
   - Configure alerting
   - Use distributed tracing

4. **Backup**:
   - Schedule PostgreSQL backups
   - Consider using managed database services

## License

MIT

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for contribution guidelines.
