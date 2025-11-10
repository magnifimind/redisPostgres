# API Documentation

## Base URL

```
http://localhost:3000
```

## Endpoints

### Health Check

Check if the API is running.

**Endpoint**: `GET /health`

**Response**:
```json
{
  "status": "healthy"
}
```

**Status Codes**:
- `200 OK`: Service is healthy

---

### Get All Bitcoins (Ranked)

Retrieve all Bitcoin entities ranked by price (highest to lowest).

**Endpoint**: `GET /api/bitcoins`

**Query Parameters**: None

**Response**:
```json
[
  {
    "symbol": "BTC",
    "price": 65000,
    "rank": 1,
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T12:00:00Z"
  },
  {
    "symbol": "ETH",
    "price": 3500,
    "rank": 2,
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  },
  {
    "symbol": "BNB",
    "price": 450,
    "rank": 3,
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
]
```

**Status Codes**:
- `200 OK`: Success
- `500 Internal Server Error`: Database or cache error

**Caching Behavior**:
- First request: Cache MISS → Query database → Cache result
- Subsequent requests: Cache HIT → Return from Redis
- Cache invalidation: On any price update or delete

**Example**:
```bash
curl http://localhost:3000/api/bitcoins
```

---

### Get Single Bitcoin

Retrieve a specific Bitcoin by symbol.

**Endpoint**: `GET /api/bitcoins/:symbol`

**Path Parameters**:
- `symbol` (string, required): Bitcoin symbol (e.g., BTC, ETH)

**Response**:
```json
{
  "symbol": "BTC",
  "price": 65000,
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T12:00:00Z"
}
```

**Status Codes**:
- `200 OK`: Bitcoin found
- `404 Not Found`: Bitcoin doesn't exist
- `500 Internal Server Error`: Database or cache error

**Caching Behavior**:
- Cache key: `bitcoin:<SYMBOL>`
- TTL: 1 hour
- Read-through: Automatic cache population on miss

**Examples**:
```bash
# Get BTC
curl http://localhost:3000/api/bitcoins/BTC

# Get ETH
curl http://localhost:3000/api/bitcoins/ETH
```

---

### Create or Update Bitcoin

Create a new Bitcoin or update existing one.

**Endpoint**: `POST /api/bitcoins`

**Headers**:
```
Content-Type: application/json
```

**Request Body**:
```json
{
  "symbol": "BTC",
  "price": 66000
}
```

**Fields**:
- `symbol` (string, required): Bitcoin symbol (max 10 chars)
- `price` (integer, required): Price in USD (whole number)

**Response**:
```json
{
  "symbol": "BTC",
  "price": 66000,
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T13:00:00Z"
}
```

**Status Codes**:
- `201 Created`: Bitcoin created or updated successfully
- `400 Bad Request`: Invalid request body
- `500 Internal Server Error`: Database or cache error

**Behavior**:
1. Upsert to PostgreSQL (INSERT ... ON CONFLICT UPDATE)
2. Update Redis cache
3. Invalidate rankings cache
4. Return updated entity

**Examples**:
```bash
# Create new Bitcoin
curl -X POST http://localhost:3000/api/bitcoins \
  -H "Content-Type: application/json" \
  -d '{"symbol": "DOGE", "price": 15}'

# Update existing Bitcoin
curl -X POST http://localhost:3000/api/bitcoins \
  -H "Content-Type: application/json" \
  -d '{"symbol": "BTC", "price": 67000}'
```

---

### Update Bitcoin

Update an existing Bitcoin price.

**Endpoint**: `PUT /api/bitcoins/:symbol`

**Path Parameters**:
- `symbol` (string, required): Bitcoin symbol

**Headers**:
```
Content-Type: application/json
```

**Request Body**:
```json
{
  "price": 68000
}
```

**Fields**:
- `price` (integer, required): New price in USD

**Response**:
```json
{
  "symbol": "BTC",
  "price": 68000,
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T14:00:00Z"
}
```

**Status Codes**:
- `200 OK`: Updated successfully
- `400 Bad Request`: Invalid price
- `404 Not Found`: Bitcoin doesn't exist (will create it)
- `500 Internal Server Error`: Database or cache error

**Behavior**:
Same as POST - uses upsert logic

**Example**:
```bash
curl -X PUT http://localhost:3000/api/bitcoins/BTC \
  -H "Content-Type: application/json" \
  -d '{"price": 68000}'
```

---

### Delete Bitcoin

Delete a Bitcoin entity.

**Endpoint**: `DELETE /api/bitcoins/:symbol`

**Path Parameters**:
- `symbol` (string, required): Bitcoin symbol

**Response**:
```json
{
  "message": "Bitcoin deleted successfully",
  "bitcoin": {
    "symbol": "BTC",
    "price": 68000,
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T14:00:00Z"
  }
}
```

**Status Codes**:
- `200 OK`: Deleted successfully
- `404 Not Found`: Bitcoin doesn't exist
- `500 Internal Server Error`: Database or cache error

**Behavior**:
1. Delete from PostgreSQL
2. Delete from Redis cache
3. Invalidate rankings cache
4. Return deleted entity

**Example**:
```bash
curl -X DELETE http://localhost:3000/api/bitcoins/DOGE
```

---

### Cache Statistics

Get Redis cache statistics.

**Endpoint**: `GET /api/cache/stats`

**Response**:
```json
{
  "info": "# Stats\ntotal_commands_processed:12345\n..."
}
```

**Status Codes**:
- `200 OK`: Success
- `500 Internal Server Error`: Redis connection error

**Example**:
```bash
curl http://localhost:3000/api/cache/stats
```

---

## Error Responses

All error responses follow this format:

```json
{
  "error": "Error message description"
}
```

### Common Errors

**400 Bad Request**:
```json
{
  "error": "Symbol and price are required"
}
```

**404 Not Found**:
```json
{
  "error": "Bitcoin not found"
}
```

**500 Internal Server Error**:
```json
{
  "error": "Failed to fetch bitcoins"
}
```

---

## CORS

CORS is enabled for all origins in development:

```
Access-Control-Allow-Origin: *
Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS
Access-Control-Allow-Headers: Origin, Content-Type, Accept
```

**Production**: Restrict to specific origins

---

## Rate Limiting

Currently: No rate limiting

**Production**: Implement rate limiting to prevent abuse

---

## Authentication

Currently: No authentication required

**Production**: Implement JWT or API key authentication

---

## Caching Headers

The API does not currently set cache control headers on responses.

**Production**: Add appropriate cache headers:
```
Cache-Control: max-age=60, public
ETag: "..."
```

---

## Examples Using Different Tools

### cURL

```bash
# Get all bitcoins
curl http://localhost:3000/api/bitcoins

# Get specific bitcoin
curl http://localhost:3000/api/bitcoins/BTC

# Create bitcoin
curl -X POST http://localhost:3000/api/bitcoins \
  -H "Content-Type: application/json" \
  -d '{"symbol":"SOL","price":120}'

# Update bitcoin
curl -X PUT http://localhost:3000/api/bitcoins/SOL \
  -H "Content-Type: application/json" \
  -d '{"price":125}'

# Delete bitcoin
curl -X DELETE http://localhost:3000/api/bitcoins/SOL
```

### JavaScript (Axios)

```javascript
import axios from 'axios';

const API_URL = 'http://localhost:3000';

// Get all
const bitcoins = await axios.get(`${API_URL}/api/bitcoins`);

// Get one
const btc = await axios.get(`${API_URL}/api/bitcoins/BTC`);

// Create/Update
const updated = await axios.post(`${API_URL}/api/bitcoins`, {
  symbol: 'BTC',
  price: 70000
});

// Delete
await axios.delete(`${API_URL}/api/bitcoins/BTC`);
```

### Python (requests)

```python
import requests

API_URL = 'http://localhost:3000'

# Get all
response = requests.get(f'{API_URL}/api/bitcoins')
bitcoins = response.json()

# Get one
response = requests.get(f'{API_URL}/api/bitcoins/BTC')
btc = response.json()

# Create/Update
response = requests.post(f'{API_URL}/api/bitcoins', json={
    'symbol': 'BTC',
    'price': 70000
})

# Delete
requests.delete(f'{API_URL}/api/bitcoins/BTC')
```

### HTTPie

```bash
# Get all
http GET localhost:3000/api/bitcoins

# Get one
http GET localhost:3000/api/bitcoins/BTC

# Create
http POST localhost:3000/api/bitcoins symbol=BTC price:=70000

# Delete
http DELETE localhost:3000/api/bitcoins/BTC
```

---

## Testing the API

### Test Sequence

```bash
# 1. Check health
curl http://localhost:3000/health

# 2. Get initial data
curl http://localhost:3000/api/bitcoins

# 3. Add new bitcoin
curl -X POST http://localhost:3000/api/bitcoins \
  -H "Content-Type: application/json" \
  -d '{"symbol":"TEST","price":100}'

# 4. Verify it appears in rankings
curl http://localhost:3000/api/bitcoins

# 5. Update price
curl -X PUT http://localhost:3000/api/bitcoins/TEST \
  -H "Content-Type: application/json" \
  -d '{"price":200}'

# 6. Verify ranking changed
curl http://localhost:3000/api/bitcoins

# 7. Delete
curl -X DELETE http://localhost:3000/api/bitcoins/TEST

# 8. Verify it's gone
curl http://localhost:3000/api/bitcoins
```

### Testing Cache Behavior

Watch backend logs while making requests:

```bash
# Terminal 1: Watch logs
kubectl logs -f -l app=backend | grep -E "Cache (HIT|MISS)"

# Terminal 2: Make requests
curl http://localhost:3000/api/bitcoins/BTC  # MISS
curl http://localhost:3000/api/bitcoins/BTC  # HIT
curl http://localhost:3000/api/bitcoins/BTC  # HIT
```
