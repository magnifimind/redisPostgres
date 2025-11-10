# Architecture Documentation

## System Overview

The Bitcoin Cache System is a three-tier application demonstrating production-ready caching patterns with Redis and PostgreSQL.

## Components

### 1. Frontend (React)

**Technology**: React 18 with Axios

**Responsibilities**:
- Display Bitcoin rankings by price
- Allow CRUD operations on Bitcoin entities
- Real-time updates via polling
- Responsive UI with gradient design

**Key Files**:
- `frontend/src/App.js` - Main component with state management
- `frontend/src/App.css` - Styling and responsive design
- `frontend/nginx.conf` - Production web server config

### 2. Backend (Go)

**Technology**: Go 1.21 with Gin framework

**Responsibilities**:
- RESTful API server
- Cache management (prime, read-through, write-through)
- Database operations
- Business logic for rankings

**Key Components**:

#### CacheService

The core caching logic:

```go
type CacheService struct {
    db          *sql.DB
    redisClient *redis.Client
    ctx         context.Context
    cacheTTL    time.Duration
}
```

**Methods**:
- `PrimeCache()` - Loads all data from DB to cache on startup
- `GetBitcoin()` - Read-through cache implementation
- `SetBitcoin()` - Write-through cache implementation
- `GetBitcoinsRanked()` - Ranked list with caching
- `DeleteBitcoin()` - Delete with cache invalidation

#### Cache Keys

- Individual: `bitcoin:<SYMBOL>` (e.g., `bitcoin:BTC`)
- Rankings: `bitcoin:rankings`

#### Cache TTL

Default: 1 hour (configurable)

### 3. Redis Cache

**Technology**: Redis 7 (Alpine)

**Purpose**:
- High-speed cache layer
- Reduces database load
- Improves response times

**Configuration**:
- Appendonly persistence enabled
- No authentication (internal cluster only)

**Data Structures**:
- String values with JSON-encoded Bitcoin objects
- TTL-based expiration

### 4. PostgreSQL Database

**Technology**: PostgreSQL 15 (Alpine)

**Purpose**:
- Source of truth for all data
- Persistent storage
- ACID guarantees

**Schema**:

```sql
CREATE TABLE bitcoins (
    symbol VARCHAR(10) PRIMARY KEY,
    price INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_bitcoin_price ON bitcoins(price DESC);
```

**Features**:
- Automatic timestamp updates via trigger
- Price index for fast ranking queries
- Persistent volume for data durability

### 5. RedisInsight

**Technology**: Redis RedisInsight (latest)

**Purpose**:
- Redis monitoring and querying UI
- Cache inspection
- Performance debugging

**Access**: LoadBalancer service on port 5540

## Data Flow

### Read Operation (Cache Hit)

```
User → Frontend → Backend → Redis (HIT) → User
                                ↓
                            (fast path)
```

### Read Operation (Cache Miss)

```
User → Frontend → Backend → Redis (MISS)
                               ↓
                          PostgreSQL
                               ↓
                           Update Redis
                               ↓
                             User
```

### Write Operation

```
User → Frontend → Backend → PostgreSQL (write)
                               ↓
                          Update Redis
                               ↓
                      Invalidate rankings
                               ↓
                             User
```

## Cache Strategies

### 1. Cache Priming

**When**: On backend startup

**How**:
1. Query all records from PostgreSQL
2. Iterate through results
3. Marshal each to JSON
4. Store in Redis with TTL
5. Log count of cached items

**Benefits**:
- Warm cache on deployment
- Immediate performance
- Predictable behavior

**Code**: `backend/main.go:PrimeCache()`

### 2. Read-Through Cache

**When**: On GET requests

**How**:
1. Check Redis for key
2. If found, return (cache hit)
3. If not found, query PostgreSQL
4. Store result in Redis
5. Return data

**Benefits**:
- Lazy loading
- Automatic cache population
- Reduced database queries

**Code**: `backend/main.go:GetBitcoin()`

### 3. Write-Through Cache

**When**: On POST/PUT requests

**How**:
1. Write to PostgreSQL (source of truth)
2. On success, update Redis cache
3. Invalidate related caches (rankings)
4. Return updated data

**Benefits**:
- Cache always consistent
- No stale data
- Immediate availability

**Code**: `backend/main.go:SetBitcoin()`

### 4. Cache Invalidation

**Strategy**: Selective invalidation

**Triggers**:
- Individual entry: On update or delete
- Rankings: On any price change

**Implementation**:
```go
cs.redisClient.Del(cs.ctx, cs.getBitcoinCacheKey(symbol))
cs.redisClient.Del(cs.ctx, rankCacheKey)
```

## Deployment Architecture

### Kubernetes Resources

```
Namespace: default

Deployments:
- postgres (1 replica)
- redis (1 replica)
- redisinsight (1 replica)
- backend (2 replicas)
- frontend (2 replicas)

Services:
- postgres (ClusterIP:5432)
- redis (ClusterIP:6379)
- redisinsight (LoadBalancer:5540)
- backend (ClusterIP:3000)
- frontend (LoadBalancer:80)

Storage:
- PersistentVolume (5Gi, manual storage class)
- PersistentVolumeClaim (5Gi)

ConfigMaps:
- postgres-config
- postgres-init-script
- backend-config
- frontend-config

Secrets:
- postgres-secret
- backend-secret
```

### Network Flow

```
Internet
   ↓
LoadBalancer (Frontend)
   ↓
Frontend Pods (2 replicas)
   ↓
Backend Service (ClusterIP)
   ↓
Backend Pods (2 replicas)
   ↓         ↓
Redis     PostgreSQL
(ClusterIP) (ClusterIP)
```

## Scaling Considerations

### Horizontal Scaling

**Frontend**: Stateless, can scale freely
```bash
kubectl scale deployment frontend --replicas=5
```

**Backend**: Stateless, can scale freely
```bash
kubectl scale deployment backend --replicas=5
```

**Redis**: Single instance (upgrade to Redis Cluster for HA)

**PostgreSQL**: Single instance (upgrade to StatefulSet with replicas)

### Vertical Scaling

Adjust resource limits in deployments:

```yaml
resources:
  requests:
    memory: "256Mi"
    cpu: "200m"
  limits:
    memory: "512Mi"
    cpu: "500m"
```

## High Availability

### Current Setup

- Frontend: 2 replicas (HA)
- Backend: 2 replicas (HA)
- Redis: 1 replica (single point of failure)
- PostgreSQL: 1 replica (single point of failure)

### Production Recommendations

1. **Redis Sentinel** or **Redis Cluster** for HA
2. **PostgreSQL with replicas** (primary + standby)
3. **Persistent volumes** with backup/restore
4. **Pod disruption budgets** for all deployments
5. **Health checks** on all services
6. **Autoscaling** based on CPU/memory

## Security

### Current Implementation

- Internal cluster communication only
- Basic authentication on PostgreSQL
- No TLS/SSL

### Production Recommendations

1. **Network Policies**: Restrict pod-to-pod communication
2. **TLS/SSL**: Encrypt all connections
3. **Secret Management**: Use external secret store (Vault, AWS Secrets Manager)
4. **RBAC**: Kubernetes role-based access control
5. **Image Scanning**: Scan for vulnerabilities
6. **Pod Security Policies**: Restrict container privileges

## Monitoring & Observability

### Logging

All services log to stdout/stderr:
- Backend: Structured logs with cache hit/miss
- Frontend: Nginx access logs
- PostgreSQL: Query logs (configurable)
- Redis: Command logs (configurable)

### Metrics

Potential metrics to collect:
- Cache hit rate
- Cache miss rate
- API request latency
- Database query time
- Pod CPU/memory usage

### Tracing

Future enhancement: Distributed tracing with OpenTelemetry

## Performance Characteristics

### Expected Latency

- Cache hit: < 10ms
- Cache miss: 50-100ms (includes DB query)
- Write operation: 50-100ms (DB + cache update)

### Throughput

- Backend: ~1000 req/s per replica (estimated)
- Redis: ~100k ops/s (typical)
- PostgreSQL: Depends on instance size

### Cache Efficiency

Expected hit rate: 80-95% (after priming)

## Future Enhancements

1. **Async cache warming**: Background job to refresh cache
2. **Cache stampede protection**: Lock mechanism for cache misses
3. **Metrics dashboard**: Grafana + Prometheus
4. **Rate limiting**: Protect backend from abuse
5. **API authentication**: JWT tokens
6. **GraphQL API**: Alternative to REST
7. **WebSocket support**: Real-time updates
8. **Multi-region**: Geographic distribution
