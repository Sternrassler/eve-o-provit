# Archiv-Verzeichnis

Dieses Verzeichnis enthält historische Dokumentations-Dateien, die durch neuere, konsolidierte Dokumente ersetzt wurden.

## Archivierte Dokumente

### IMPLEMENTATION_SUMMARY.md (Archiviert: 2025-11-04)

**Grund:** Beschreibt veraltete Implementierung ohne BatchFetcher-Integration  
**Ersetzt durch:** `../ARCHITECTURE.md`

**Inhalt:**

- Frühe Backend-Implementierung (Issue #16)
- Sequenzielle Market Order Fetching
- Basis-Struktur ohne Frontend

### PHASE3_IMPLEMENTATION_SUMMARY.md (Archiviert: 2025-11-04)

**Grund:** Beschreibt Worker Pool Pattern, das in eve-esi-client zentralisiert wurde  
**Ersetzt durch:**

- `../ARCHITECTURE.md` (aktuelle Architektur)
- ADR-011 (Status: Superseded)
- [eve-esi-client/pkg/pagination](https://github.com/Sternrassler/eve-esi-client)

**Inhalt:**

- MarketOrderFetcher Implementierung (veraltet, gelöscht)
- Worker Pool Details (jetzt in eve-esi-client)
- Performance-Tests (Baseline 120s → 8.7s)

## Aktuelle Dokumentation

Verwende stattdessen:

- `../ARCHITECTURE.md` - Vollständige System-Architektur (November 2025)
- `../PROJECT_STRUCTURE.md` - Code-Organisation
- `../EVE-SSO-INTEGRATION.md` - Authentication Guide
- `../README.md` - Getting Started Guide

## Historische Referenz

Diese Dateien bleiben für historische Nachvollziehbarkeit erhalten, sind aber **nicht mehr aktuell**.

Bei Fragen zur aktuellen Implementierung siehe die oben genannten aktuellen Dokumente.
