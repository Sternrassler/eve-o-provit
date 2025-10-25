# evedb Package

SQLite Access Layer für EVE SDE Daten.

## Status

🚧 **Migrationspending** - Code wird aus eve-sde Projekt übernommen

## Migration Plan

Folgende Komponenten aus `eve-sde/pkg/evedb` übernehmen:

1. **Database Connection**
   - SQLite Read-Only Connection
   - Connection Pool Management

2. **SDE Queries**
   - Blueprint Queries
   - Item/Type Queries
   - Material Queries
   - View Access (Cargo, Manufacturing Chains)

3. **Models**
   - Blueprint structs
   - Item structs
   - Material structs

## Usage (planned)

```go
import "github.com/Sternrassler/eve-o-provit/backend/pkg/evedb"

// Open SDE database
db, err := evedb.Open("data/sde/sde.sqlite")
if err != nil {
    log.Fatal(err)
}
defer db.Close()

// Query blueprints
blueprints, err := db.GetBlueprints()
```
