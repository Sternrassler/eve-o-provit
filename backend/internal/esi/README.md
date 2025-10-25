# ESI Client Package

EVE Swagger Interface (ESI) API Client.

## Status

🚧 **TODO** - ESI Client Implementation

## Features (planned)

- Market Orders (Buy/Sell)
- Market History
- Universe Data
- Rate Limiting (150 req/s burst, 20 req/s sustained)
- Redis Cache Integration
- Auto-Retry on 429

## Usage (planned)

```go
import "github.com/Sternrassler/eve-o-provit/backend/internal/esi"

client := esi.NewClient(esi.Config{
    UserAgent: "eve-o-provit/0.1.0",
    CacheEnabled: true,
})

// Get market orders
orders, err := client.GetMarketOrders(regionID, typeID)
```

## References

- [ESI Swagger UI](https://esi.evetech.net/ui/)
- [ESI Docs](https://docs.esi.evetech.net/)
