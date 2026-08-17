> [!NOTE]
> **Archived 2026-08-17.** Documents `prover/internal/api/server.go`, a Fiber-based API
> (`/batches`, `/status/:id`, Swagger UI) bound to a legacy `BatchRegistry` contract
> binding. Verified during a docs audit: `prover/cmd/prover` (the actual binary invoked
> by `prover start` / `make run-prover`) never references this package — it isn't wired
> into the live prover at all. Kept only in case it's revived; don't rely on `/batches`
> pagination working against a running prover today.

# Prover API Pagination Guide

## Endpoint

`GET /batches`

Retrieve multiple batch metadata entries with pagination support. **Batches are returned in descending order (newest first).**

## Query Parameters

| Parameter | Type | Required | Default | Valid Values | Description |
|-----------|------|----------|---------|--------------|-------------|
| `page` | integer | No | 1 | ≥ 1 | Page number to retrieve |
| `limit` | integer | No | 25 | 10, 25, 50, 100 | Number of items per page |

## Response Format

**Note:** Batches are returned in **descending order** (newest first).

```json
{
  "page": 1,
  "limit": 25,
  "count": 25,
  "batches": [
    {
      "batch_number": 100,  // Newest batch
      "tx_hash": "0x123...",
      "tx_hashes": ["0xabc...", "0xdef..."],
      "timestamp": 1703001234
    },
    {
      "batch_number": 99,
      "tx_hash": "0x456...",
      "tx_hashes": ["0xghi...", "0xjkl..."],
      "timestamp": 1703001200
    }
    // ... 23 more batches
  ]
}
```

## Examples

### Get first page (default 25 items)
```bash
curl http://localhost:8081/batches
```

Response:
```json
{
  "page": 1,
  "limit": 25,
  "count": 25,
  "batches": [ ... 25 batches ... ]
}
```

### Get page 2 with 50 items
```bash
curl "http://localhost:8081/batches?page=2&limit=50"
```

Response:
```json
{
  "page": 2,
  "limit": 50,
  "count": 50,
  "batches": [ ... batches 51-100 ... ]
}
```

### Get first 10 batches
```bash
curl "http://localhost:8081/batches?limit=10"
```

Response:
```json
{
  "page": 1,
  "limit": 10,
  "count": 10,
  "batches": [ ... batches 1-10 ... ]
}
```

### Get 100 batches at once
```bash
curl "http://localhost:8081/batches?limit=100"
```

Response:
```json
{
  "page": 1,
  "limit": 100,
  "count": 100,
  "batches": [ ... batches 1-100 ... ]
}
```

### Navigate to page 5 with 25 items per page
```bash
curl "http://localhost:8081/batches?page=5&limit=25"
```

This returns batches 101-125.

## Batch Data Structure

Each batch in the response contains:

```json
{
  "batch_number": 1000,
  "tx_hash": "0x...",      // L1 proof submission transaction hash
  "tx_hashes": [           // Array of L2 transaction hashes (128 items)
    "0xabc...",
    "0xdef...",
    ...
  ],
  "timestamp": 1703001234  // Unix timestamp when proven
}
```

## Error Responses

### Invalid page number
```bash
curl "http://localhost:8081/batches?page=0"
```

Response: `400 Bad Request`
```
Page must be >= 1
```

### Invalid limit (auto-corrected)
```bash
curl "http://localhost:8081/batches?limit=30"
```

Response: Automatically corrected to closest valid limit (25).

```json
{
  "page": 1,
  "limit": 25,
  "count": 25,
  "batches": [ ... ]
}
```

### Server error
Response: `500 Internal Server Error`
```
Error fetching batches: <error message>
```

## Usage Patterns

### Fetch all batches (paginated loop)

**Bash:**
```bash
page=1
while true; do
  response=$(curl -s "http://localhost:8081/batches?page=$page&limit=100")
  count=$(echo "$response" | jq '.count')
  
  # Process batches
  echo "$response" | jq '.batches[]'
  
  # Stop if less than limit (last page)
  if [ "$count" -lt 100 ]; then
    break
  fi
  
  page=$((page + 1))
done
```

**Python:**
```python
import requests

page = 1
limit = 100
base_url = "http://localhost:8081/batches"

while True:
    response = requests.get(base_url, params={"page": page, "limit": limit})
    data = response.json()
    
    # Process batches
    for batch in data["batches"]:
        print(f"Batch {batch['batch_number']}: {batch['tx_hash']}")
    
    # Stop if less than limit (last page)
    if data["count"] < limit:
        break
    
    page += 1
```

**JavaScript:**
```javascript
async function fetchAllBatches() {
  let page = 1;
  const limit = 100;
  const allBatches = [];
  
  while (true) {
    const response = await fetch(
      `http://localhost:8081/batches?page=${page}&limit=${limit}`
    );
    const data = await response.json();
    
    allBatches.push(...data.batches);
    
    // Stop if less than limit (last page)
    if (data.count < limit) break;
    
    page++;
  }
  
  return allBatches;
}
```

### Get latest 50 batches
```bash
curl "http://localhost:8081/batches?limit=50"
```

### Get specific range (e.g., batches 201-250)
```bash
# Page 5 with limit 50 gives batches 201-250
curl "http://localhost:8081/batches?page=5&limit=50"
```

## Recommended Patterns

### For UI/Dashboard
- **First load:** `limit=25` (default)
- **User clicks "Next":** Increment `page`
- **User changes page size:** Update `limit` (10, 25, 50, 100)

### For Data Export
- **Use:** `limit=100` (maximum efficiency)
- **Loop:** Until `count < limit`

### For Monitoring
- **Recent batches:** `limit=10` (latest 10)
- **Poll frequency:** Every 30-60 seconds

## Performance Considerations

| Limit | Batches | Response Size | Latency |
|-------|---------|---------------|---------|
| 10 | 10 | ~40 KB | ~5 ms |
| 25 | 25 | ~100 KB | ~10 ms |
| 50 | 50 | ~200 KB | ~15 ms |
| 100 | 100 | ~400 KB | ~25 ms |

**Recommendation:** Use `limit=50` for best balance of efficiency and response time.

## Swagger UI

Access interactive API documentation:
```
http://localhost:8081/swagger/index.html
```

Navigate to **Batches → GET /batches** to try the API with different parameters.

## Related Endpoints

- `GET /batches/latest` - Get latest proven batch
- `GET /batches/:id` - Get specific batch by ID
- `GET /status/:id` - Get proof status for batch

## Notes

- Batches are returned in ascending order by batch number
- Empty pages return `count: 0, batches: []`
- Invalid `limit` values are auto-corrected to nearest valid value
- Maximum 100 batches per request (performance limit)
