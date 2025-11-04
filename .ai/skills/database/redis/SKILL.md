# Database Skill: Redis

**Tech Stack:** Redis v9.16.0 (go-redis/v9)

**Project:** eve-o-provit Market Data Caching

---

## Architecture Patterns

### Cache-Aside Pattern

- **Check cache first:** Read from Redis before database
- **Cache miss:** Fetch from database, write to Redis
- **TTL Strategy:** Set expiration on all cached data

### Dual-Purpose Caching

- **Response Cache:** Store complete API responses (ESI market orders)
- **Computed Results Cache:** Store calculation results (trading routes)

---

## Best Practices

1. **Always Set TTL:** No cache entry without expiration (prevent memory bloat)
2. **Key Naming Convention:** Use colons for hierarchy (`market:orders:10000002:34`)
3. **Compression:** Use gzip for large values (>1KB) to save memory (~80% reduction)
4. **Context Timeouts:** Redis operations should have timeouts (1-3 seconds)
5. **Error Handling:** Cache failures should not break application (fail-open pattern)
6. **Serialization:** Use JSON for structured data, not Protocol Buffers (debugging ease)
7. **Monitoring:** Track hit/miss ratio, eviction rate, memory usage

---

## Common Patterns

### 1. Connection Setup

```go
import "github.com/redis/go-redis/v9"

rdb := redis.NewClient(&redis.Options{
    Addr:         "localhost:6379",
    Password:     "",
    DB:           0,
    PoolSize:     10,
    MinIdleConns: 5,
})

ctx := context.Background()
if err := rdb.Ping(ctx).Err(); err != nil {
    return fmt.Errorf("redis ping failed: %w", err)
}
```

### 2. Cache-Aside Read Pattern

```go
func (c *Cache) GetOrders(ctx context.Context, regionID, typeID int) ([]Order, error) {
    key := fmt.Sprintf("market:orders:%d:%d", regionID, typeID)
    
    // Try cache first
    val, err := c.rdb.Get(ctx, key).Result()
    if err == nil {
        var orders []Order
        json.Unmarshal([]byte(val), &orders)
        return orders, nil
    }
    
    // Cache miss - fetch from database
    orders, err := c.db.GetOrders(ctx, regionID, typeID)
    if err != nil {
        return nil, err
    }
    
    // Write to cache (async, don't block on failure)
    go c.setCache(key, orders, 5*time.Minute)
    
    return orders, nil
}
```

### 3. Compressed Storage

```go
func compressJSON(data interface{}) ([]byte, error) {
    jsonData, _ := json.Marshal(data)
    
    var buf bytes.Buffer
    gw := gzip.NewWriter(&buf)
    gw.Write(jsonData)
    gw.Close()
    
    return buf.Bytes(), nil
}

func (c *Cache) SetCompressed(ctx context.Context, key string, data interface{}, ttl time.Duration) error {
    compressed, _ := compressJSON(data)
    return c.rdb.Set(ctx, key, compressed, ttl).Err()
}
```

### 4. Batch Operations

```go
func (c *Cache) SetMultiple(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
    pipe := c.rdb.Pipeline()
    
    for key, value := range items {
        jsonData, _ := json.Marshal(value)
        pipe.Set(ctx, key, jsonData, ttl)
    }
    
    _, err := pipe.Exec(ctx)
    return err
}
```

---

## Anti-Patterns

❌ **No Expiration:** Every key must have TTL (use `SETEX`, not `SET` without expiry)

❌ **Large Values Without Compression:** Values >1KB should be gzipped

❌ **Blocking on Cache Failures:** Cache should be async, failures don't block main flow

❌ **Cache as Source of Truth:** Redis is ephemeral, database is source of truth

❌ **Complex Data Structures:** Keep it simple (strings, hashes), avoid Lua scripts unless necessary

---

## Integration with Backend

### Service Layer Integration

- Services decide when to cache (after database fetch)
- Services handle cache invalidation (on updates)
- Services implement fallback to database on cache miss

### Error Handling Strategy

- **Cache Read Failure:** Log error, proceed to database
- **Cache Write Failure:** Log error, don't retry (avoid cascading failures)
- **Connection Failure:** Disable caching temporarily, retry with backoff

---

## Performance Considerations

- **Pipeline Usage:** Batch multiple `SET` operations into one pipeline (~5x faster)
- **Connection Pooling:** Configure pool size based on concurrency (10-20 connections)
- **Compression Trade-off:** CPU overhead vs. memory savings (worth it for >1KB values)
- **TTL Strategy:** Short TTL for volatile data (5min), long TTL for static data (24h)
- **Memory Limits:** Set `maxmemory` policy to `allkeys-lru` (evict least recently used)

---

## Security Guidelines

- **Authentication:** Use `requirepass` in production (AUTH command)
- **Network Isolation:** Bind to localhost or private network only
- **No Sensitive Data:** Don't cache secrets, tokens, or PII without encryption
- **ACLs:** Use Redis 6+ ACLs for fine-grained permissions

---

## Quick Reference

| Task | Command |
|------|---------|
| Set with TTL | `rdb.Set(ctx, key, value, 5*time.Minute)` |
| Get | `val, err := rdb.Get(ctx, key).Result()` |
| Delete | `rdb.Del(ctx, key)` |
| Check exists | `exists, _ := rdb.Exists(ctx, key).Result()` |
| Batch SET | `pipe := rdb.Pipeline(); pipe.Set(...); pipe.Exec(ctx)` |
| Increment | `rdb.Incr(ctx, "counter")` |
| Hash SET | `rdb.HSet(ctx, "user:1", "name", "John")` |
