# ADR-009: Shared Redis Infrastructure

Status: Proposed  
Datum: 2025-10-27  
Kontext: Infrastructure & Dependency Management  
Entscheider: Architecture Team  

## Kontext

EVE-O-Provit nutzt mehrere Komponenten, die Redis benötigen:

1. **ESI Client Library** (eve-esi-client): Caching & Rate Limiting
2. **Backend Sessions**: JWT-Session-Management
3. **Application Cache**: Profit-Berechnungen, Market-Daten
4. **Background Jobs**: Job-Queue (zukünftig)

Es stellt sich die Frage, ob eine oder mehrere Redis-Instanzen verwendet werden sollen.

### Anforderungen

1. **Kosteneffizienz**: Minimale Infrastrukturkosten
2. **Einfachheit**: Wartbares Deployment
3. **Isolation**: Keine Daten-Konflikte zwischen Komponenten
4. **Performance**: Ausreichende Skalierung für erwartete Last
5. **Entwickler-Experience**: Lokales Setup muss einfach sein

### Betrachtete Optionen

#### Option A: Separate Redis-Instanzen
Jede Komponente erhält eigene Redis-Instanz.

**Vorteile**:
- Vollständige Isolation
- Unabhängiges Scaling
- Separate Monitoring/Alerts

**Nachteile**:
- Hohe Infrastrukturkosten (3-4 Redis Instanzen)
- Komplexes Deployment
- Overhead in Entwicklungsumgebung

#### Option B: Shared Redis mit separaten Datenbanken
Eine Redis-Instanz, verschiedene DBs (DB 0-15).

**Vorteile**:
- Ressourceneffizient
- Logische Trennung via DB-Nummer

**Nachteile**:
- Redis Cluster unterstützt nur DB 0
- Schwieriger zu migrieren auf Cluster-Setup
- Key-Namespacing ist modernerer Ansatz

#### Option C: Shared Redis mit Key-Namespacing
Eine Redis-Instanz, alle nutzen DB 0 mit unterschiedlichen Key-Prefixes.

**Vorteile**:
- Maximale Kompatibilität (Cluster-ready)
- Einfaches Deployment (eine Instanz)
- Klare Trennung via Key-Naming
- Flexibel skalierbar

**Nachteile**:
- Gemeinsamer Memory-Pool (erfordert Monitoring)
- Flush-Operationen müssen prefix-aware sein

## Entscheidung

**Option C: Shared Redis mit Key-Namespacing**

Eine Redis-Instanz (DB 0) mit folgenden Key-Prefixes:

| Komponente       | Prefix        | Beispiel-Key                          |
|------------------|---------------|---------------------------------------|
| ESI Client       | `esi:`        | `esi:cache:/markets/10000002/orders/` |
| Sessions         | `app:session:`| `app:session:user-abc123`             |
| Application Cache| `app:cache:`  | `app:cache:profit-calc-xyz`           |
| Background Jobs  | `app:jobs:`   | `app:jobs:queue:default`              |

### Begründung

1. **Kosteneffizienz**: Eine Redis-Instanz statt 3-4
2. **Cluster-Ready**: Key-Namespacing funktioniert mit Redis Cluster
3. **Einfaches Deployment**: Docker Compose mit einer Redis
4. **Observability**: Keys sind eindeutig identifizierbar
5. **Entwickler-Experience**: `docker-compose up redis` - fertig

## Implementierung

### Docker Compose (Entwicklung)

```yaml
services:
  redis:
    image: redis:7-alpine
    container_name: eve-o-provit-redis
    ports:
      - "6379:6379"
    volumes:
      - redis-data:/data
    command: redis-server --appendonly yes
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 3s
      retries: 5

volumes:
  redis-data:
```

### Backend Integration

```go
package main

import (
    "github.com/redis/go-redis/v9"
    "github.com/Sternrassler/eve-esi-client/pkg/client"
)

func main() {
    // Shared Redis Client
    redisClient := redis.NewClient(&redis.Options{
        Addr:     getEnv("REDIS_URL", "localhost:6379"),
        Password: getEnv("REDIS_PASSWORD", ""),
        DB:       0, // Always use DB 0 (Cluster-compatible)
    })
    
    // Health Check
    if err := redisClient.Ping(ctx).Err(); err != nil {
        log.Fatal("Redis connection failed:", err)
    }
    
    // ESI Client with prefix "esi:"
    esiClient, err := client.New(client.Config{
        Redis:     redisClient,
        KeyPrefix: "esi:",
        UserAgent: "EVE-O-Provit/1.0 (contact@example.com)",
    })
    if err != nil {
        log.Fatal("ESI client creation failed:", err)
    }
    
    // Session Manager with prefix "app:session:"
    sessionStore := NewRedisSessionStore(redisClient, "app:session:")
    
    // Application Cache with prefix "app:cache:"
    appCache := NewRedisCache(redisClient, "app:cache:")
    
    // Background Jobs with prefix "app:jobs:"
    jobQueue := NewRedisJobQueue(redisClient, "app:jobs:")
    
    // Start server...
}
```

### Environment Variables

```env
# Redis Configuration
REDIS_URL=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0
REDIS_MAX_RETRIES=3
REDIS_POOL_SIZE=10
```

### Production Deployment

**Managed Redis** (z. B. DigitalOcean, AWS ElastiCache):

```yaml
# Terraform / Infrastructure as Code
resource "digitalocean_database_cluster" "redis" {
  name       = "eve-o-provit-redis"
  engine     = "redis"
  version    = "7"
  size       = "db-s-1vcpu-1gb"
  region     = "fra1"
  node_count = 1
}
```

**Redis Cluster** (zukünftige Skalierung):

```go
// Redis Cluster Client (gleiche API wie Standalone)
clusterClient := redis.NewClusterClient(&redis.ClusterOptions{
    Addrs: []string{
        "redis-node-1:6379",
        "redis-node-2:6379",
        "redis-node-3:6379",
    },
})

// ESI Client funktioniert ohne Änderungen
esiClient := client.New(client.Config{
    Redis: clusterClient, // Interface ist identisch
    // ...
})
```

## Konsequenzen

### Positiv

✅ **Kosteneffizienz**: Eine Redis-Instanz statt mehrere  
✅ **Einfaches Deployment**: `docker-compose up` für lokale Entwicklung  
✅ **Skalierbar**: Cluster-Migration ohne Code-Änderungen möglich  
✅ **Monitoring**: Alle Keys in einer Instanz (einfacher zu überwachen)  
✅ **Entwickler-Experience**: Minimale Setup-Komplexität  
✅ **Flexibilität**: Komponenten können jederzeit auf eigene Redis migriert werden  

### Negativ

⚠️ **Shared Memory**: Komponenten teilen sich Memory-Limit (erfordert Monitoring)  
⚠️ **Flush-Risk**: `FLUSHDB` würde alle Komponenten betreffen  
⚠️ **Noisy Neighbor**: Eine Komponente kann theoretisch alle Ressourcen belegen  

### Neutral

- Key-Naming-Convention muss dokumentiert und eingehalten werden
- Redis-Monitoring muss Memory-Usage pro Prefix tracken

## Risikomanagement

### Memory-Überlauf
**Risiko**: Eine Komponente belegt gesamten Memory.

**Mitigation**:
1. Redis `maxmemory-policy: allkeys-lru` konfigurieren
2. Monitoring für Memory-Usage pro Prefix
3. ESI Client hat eingebaute Cache-Size-Limits (ADR-007)
4. TTL für alle Keys (keine Leaks)

### Performance-Degradation
**Risiko**: Zu viele Keys verlangsamen Operationen.

**Mitigation**:
1. Redis 7 ist für Millionen Keys optimiert
2. Erwartete Last: < 100k Keys total
3. Monitoring für Slow Queries
4. Bei Bedarf: Migration auf Cluster

### Flush-Operationen
**Risiko**: Versehentliches `FLUSHDB` löscht alle Daten.

**Mitigation**:
1. `FLUSHDB` nur in Tests erlaubt
2. Production: Prefix-basiertes Löschen via SCAN
3. Redis ACL: `FLUSHDB` für App-User deaktivieren

```bash
# Redis ACL (Production)
ACL SETUSER app-user on >password ~app:* ~esi:* -FLUSHDB -FLUSHALL +@all
```

## Monitoring & Observability

### Metrics (Prometheus)

```go
// Redis Metrics per Namespace
redis_keys_total{prefix="esi"}
redis_keys_total{prefix="app:session"}
redis_keys_total{prefix="app:cache"}
redis_memory_bytes{prefix="esi"}
redis_memory_bytes{prefix="app:session"}
```

### Health Checks

```go
// Health Check Endpoint
func (h *HealthHandler) RedisHealth(c *fiber.Ctx) error {
    // Ping
    if err := h.redis.Ping(c.Context()).Err(); err != nil {
        return c.Status(503).JSON(fiber.Map{
            "redis": "unhealthy",
            "error": err.Error(),
        })
    }
    
    // Memory Check
    info, _ := h.redis.Info(c.Context(), "memory").Result()
    usedMemory := parseMemory(info, "used_memory")
    maxMemory := parseMemory(info, "maxmemory")
    
    return c.JSON(fiber.Map{
        "redis": "healthy",
        "memory_used": usedMemory,
        "memory_max": maxMemory,
        "memory_percent": (usedMemory / maxMemory) * 100,
    })
}
```

## Key-Naming-Konventionen

### Standard-Format

```
<namespace>:<entity>:<identifier>[:<field>]
```

### Beispiele

```bash
# ESI Client (automatisch via eve-esi-client)
esi:cache:/markets/10000002/orders/
esi:ratelimit:errors:remaining
esi:metrics:requests:total

# Sessions
app:session:user-abc123
app:session:user-xyz789:refresh-token

# Application Cache
app:cache:profit-calc:region-10000002
app:cache:market-orders:item-34:region-10000002

# Background Jobs
app:jobs:queue:default
app:jobs:processing:job-123
app:jobs:result:job-123
```

### TTL-Strategie

| Namespace        | Standard-TTL | Begründung                          |
|------------------|--------------|-------------------------------------|
| `esi:cache:*`    | ESI Headers  | Respektiert ESI `expires` (ADR-007) |
| `app:session:*`  | 24 hours     | Session-Dauer                       |
| `app:cache:*`    | 1-6 hours    | Abhängig von Daten-Volatilität      |
| `app:jobs:*`     | 7 days       | Job-Historie                        |

## Migration & Rollback

### Zu separaten Redis migrieren

Falls zukünftig Trennung nötig:

```go
// Separate Redis für ESI Client
esiRedis := redis.NewClient(&redis.Options{
    Addr: "esi-redis:6379",
})

esiClient := client.New(client.Config{
    Redis:     esiRedis,  // Eigene Instanz
    KeyPrefix: "esi:",    // Prefix bleibt gleich
})

// App behält shared Redis
appRedis := redis.NewClient(&redis.Options{
    Addr: "app-redis:6379",
})
```

**Aufwand**: Minimal (nur Redis-Client-Config ändern)

## Referenzen

- **ADR-009 (eve-esi-client)**: Redis Dependency Injection
- **ADR-007 (eve-esi-client)**: ESI Caching Strategy
- **ADR-006 (eve-esi-client)**: ESI Error & Rate Limit Handling
- [Redis Best Practices](https://redis.io/docs/latest/develop/use/patterns/)
- [Redis Cluster Specification](https://redis.io/docs/latest/operate/oss_and_stack/reference/cluster-spec/)
- [go-redis Documentation](https://redis.uptrace.dev/)

## Änderungshistorie

| Datum      | Änderung                          | Autor |
|------------|-----------------------------------|-------|
| 2025-10-27 | Initial proposal                  | AI    |
