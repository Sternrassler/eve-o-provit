# ADR-010: SDE Database Path Convention

**Status:** Accepted  
**Date:** 2025-11-01  
**Deciders:** Development Team  
**Technical Story:** Inkonsistente SDE-Pfade zwischen Projekten und Umgebungen

## Kontext

Das EVE-SDE Projekt und eve-o-provit haben inkonsistente Pfade und Dateinamen für die SQLite-Datenbank:

**Aktuelles Chaos:**

- eve-sde Projekt: `data/sqlite/eve-sde.db` (405 MB) ✅
- eve-sde Release: `eve-sde.db.gz` → entpackt zu `eve-sde.db`
- eve-o-provit download-sde.sh: Lädt herunter als `sde.sqlite` ❌
- eve-o-provit main.go Default: `../eve-sde/data/sqlite/sde.sqlite` ❌
- docker-compose.yml: `SDE_PATH: /data/sde/sde.sqlite` ❌
- Dokumentation: Teils `eve-sde.db`, teils `sde.sqlite`, teils `sde.db`

**Problem:**

- Keine Single Source of Truth
- Docker Mount zeigt auf falschen Dateinamen
- Download-Script erstellt falschen Dateinamen
- Default-Pfade in Code stimmen nicht überein

## Entscheidung

**Verbindliche Konvention:**

1. **Kanonischer Dateiname:** `eve-sde.db` (konsistent mit eve-sde Projekt Release)
2. **Lokales Verzeichnis:** `backend/data/sde/eve-sde.db` (im eve-o-provit Projekt)
3. **Docker Container:** `/data/sde/eve-sde.db` (Volume Mount von backend/data/sde)
4. **Environment Variable:** `SDE_PATH` (keine Defaults im Code außer für Tests)

**Dateipfade:**

| Kontext | Pfad | Begründung |
|---------|------|------------|
| eve-sde Projekt | `data/sqlite/eve-sde.db` | Build Output |
| GitHub Release | `eve-sde.db.gz` | Komprimiert, entpackt zu `eve-sde.db` |
| eve-o-provit lokal | `backend/data/sde/eve-sde.db` | Download-Ziel (aus Release) |
| Docker Container | `/data/sde/eve-sde.db` | Volume Mount von backend/data/sde |
| Tests | Testdata mit beliebigem Namen | Isoliert |

**Environment Variables:**

```bash
# Produktion (Docker)
SDE_PATH=/data/sde/eve-sde.db

# Lokale Entwicklung (ohne Docker)
SDE_PATH=backend/data/sde/eve-sde.db

# Tests
SDE_PATH=/tmp/test-sde.db  # oder In-Memory
```

## Konsequenzen

### Positive

- ✅ Konsistente Benennung über alle Projekte
- ✅ Klare Single Source of Truth (eve-sde Projekt)
- ✅ Keine verwirrenden Aliase (sde.sqlite, sde.db)
- ✅ Docker Mounts eindeutig dokumentiert
- ✅ Download-Scripts passen zum Release-Namen

### Negative

- 🔧 Breaking Change: Alle Pfade müssen aktualisiert werden
- 🔧 Alte Dokumentation muss überarbeitet werden

### Migration

1. **eve-o-provit download-sde.sh:**
   - Ändere `TARGET_FILE="sde.sqlite"` → `TARGET_FILE="eve-sde.db"`
   - Ändere `TARGET_DIR` bleibt `backend/data/sde` (lokal im Projekt)
   - Lädt neueste Release von github.com/Sternrassler/eve-sde

2. **docker-compose.yml:**
   - Volume: `../backend/data/sde:/data/sde:ro` (relativ zum deployments/ Verzeichnis)
   - Env: `SDE_PATH=/data/sde/eve-sde.db`
   - **:ro Flag** verhindert versehentliche Änderungen

3. **backend/cmd/api/main.go:**
   - Default nur für lokale Dev: `../eve-sde/data/sqlite/eve-sde.db`
   - Docker muss **immer** SDE_PATH setzen

4. **Dokumentation:**
   - README.md: Symlink-Empfehlung dokumentieren
   - deployments/README.md: Volume Mounts erklären

5. **Cleanup:**
   - Lösche leere `/home/ix/vscode/eve-sde/data/sqlite/sde.db`
   - Prüfe auf weitere Altlasten

## Implementation Checklist

- [x] ADR akzeptiert
- [x] eve-o-provit/scripts/download-sde.sh: TARGET_FILE + TARGET_DIR (backend/data/sde)
- [x] eve-o-provit/deployments/docker-compose.yml: Volume Mount ../backend/data/sde:/data/sde:ro
- [x] eve-o-provit/backend/cmd/api/main.go: Default data/sde/eve-sde.db
- [x] eve-o-provit/backend/cmd/test-sde/main.go: Pfad ../data/sde/eve-sde.db
- [x] eve-o-provit/backend/internal/database/db.go: SQLite URI mit immutable=1
- [x] eve-o-provit/backend/pkg/evedb/cargo/cargo.go: SDE Schema (types, volume)
- [x] eve-o-provit/deployments/README.md: Lokales backend/data/sde dokumentiert
- [x] Alte sde.sqlite* Dateien gelöscht aus backend/data/sde/
- [x] SDE-Datei kopiert nach backend/data/sde/eve-sde.db
- [x] Docker Rebuild getestet - Backend startet erfolgreich
- [x] API getestet - GetItemVolume funktioniert, 26 profitable Items
- [x] Volume Mount verifiziert - Container sieht eve-sde.db (405 MB)
- [ ] eve-o-provit/README.md: Setup-Schritte inkl. download-sde.sh (TODO)
- [ ] Makefile Target `make download-sde` hinzufügen (TODO)

## References

- eve-sde/README.md: Definiert `eve-sde.db` als Hauptprodukt
- GitHub Releases: Verwendet `eve-sde.db.gz`
- ADR-009: Redis Dependency Injection (ähnliches Konfig-Thema)
