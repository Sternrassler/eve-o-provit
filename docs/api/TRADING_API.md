# Trading API Endpoints

## Übersicht

Diese API stellt Endpoints für Intra-Region Trading Route Calculations und Character Context bereit.

## Endpoints

### 1. Calculate Trading Routes (Public)

Berechnet profitable Trading Routes innerhalb einer Region.

**Endpoint:** `POST /api/v1/trading/routes/calculate`

**Authentication:** Nicht erforderlich (Public)

**Request Body:**
```json
{
  "region_id": 10000002,
  "ship_type_id": 648,
  "cargo_capacity": 15000
}
```

**Request Parameter:**
- `region_id` (int, required): Region ID (z.B. 10000002 für The Forge)
- `ship_type_id` (int, required): Ship Type ID (z.B. 648 für Badger)
- `cargo_capacity` (float, optional): Cargo Capacity in m³. Wenn nicht angegeben, wird aus SDE ermittelt.

**Response:** `200 OK`
```json
{
  "region_id": 10000002,
  "region_name": "The Forge",
  "ship_type_id": 648,
  "ship_name": "Badger",
  "cargo_capacity": 15000,
  "calculation_time_ms": 1234,
  "routes": [
    {
      "item_type_id": 34,
      "item_name": "Tritanium",
      "buy_system_id": 30000142,
      "buy_system_name": "Jita",
      "buy_station_id": 60003760,
      "buy_station_name": "Jita IV - Moon 4 - Caldari Navy Assembly Plant",
      "buy_price": 5.50,
      "sell_system_id": 30002187,
      "sell_system_name": "Amarr",
      "sell_station_id": 60008494,
      "sell_station_name": "Amarr VIII (Oris) - Emperor Family Academy",
      "sell_price": 6.00,
      "quantity": 1500,
      "profit_per_unit": 0.50,
      "total_profit": 750.00,
      "spread_percent": 9.09,
      "travel_time_seconds": 600.0,
      "round_trip_seconds": 1200.0,
      "isk_per_hour": 2250000.0,
      "jumps": 20,
      "item_volume": 10.0
    }
  ]
}
```

**Error Responses:**
- `400 Bad Request`: Ungültige Request Parameter
- `500 Internal Server Error`: Fehler bei Berechnung

**Beispiel:**
```bash
curl -X POST http://localhost:9001/api/v1/trading/routes/calculate \
  -H "Content-Type: application/json" \
  -d '{
    "region_id": 10000002,
    "ship_type_id": 648
  }'
```

---

### 2. Get Character Location (Protected)

Ruft die aktuelle Location des Characters ab (ESI Proxy mit SDE Enrichment).

**Endpoint:** `GET /api/v1/character/location`

**Authentication:** Bearer Token (EVE SSO) erforderlich

**Headers:**
```
Authorization: Bearer <access_token>
```

**Response:** `200 OK`
```json
{
  "solar_system_id": 30000142,
  "solar_system_name": "Jita",
  "region_id": 10000002,
  "region_name": "The Forge",
  "station_id": 60003760,
  "station_name": "Jita IV - Moon 4 - Caldari Navy Assembly Plant"
}
```

**Error Responses:**
- `401 Unauthorized`: Kein oder ungültiges Token
- `500 Internal Server Error`: ESI Fehler

**Beispiel:**
```bash
curl http://localhost:9001/api/v1/character/location \
  -H "Authorization: Bearer <your_token>"
```

---

### 3. Get Character Ship (Protected)

Ruft Informationen über das aktuelle Schiff des Characters ab.

**Endpoint:** `GET /api/v1/character/ship`

**Authentication:** Bearer Token (EVE SSO) erforderlich

**Response:** `200 OK`
```json
{
  "ship_type_id": 648,
  "ship_name": "Badger",
  "ship_item_id": 1234567890,
  "ship_type_name": "Badger",
  "cargo_capacity": 15000.0
}
```

**Error Responses:**
- `401 Unauthorized`: Kein oder ungültiges Token
- `500 Internal Server Error`: ESI Fehler

**Beispiel:**
```bash
curl http://localhost:9001/api/v1/character/ship \
  -H "Authorization: Bearer <your_token>"
```

---

### 4. Get Character Ships (Protected)

Listet alle Schiffe des Characters im Hangar auf.

**Endpoint:** `GET /api/v1/character/ships`

**Authentication:** Bearer Token (EVE SSO) erforderlich

**Response:** `200 OK`
```json
{
  "ships": [
    {
      "item_id": 1234567890,
      "type_id": 648,
      "type_name": "Badger",
      "location_id": 60003760,
      "location_name": "Jita IV - Moon 4",
      "location_flag": "Hangar",
      "cargo_capacity": 15000.0,
      "is_singleton": true
    }
  ],
  "count": 1
}
```

**Error Responses:**
- `401 Unauthorized`: Kein oder ungültiges Token
- `500 Internal Server Error`: ESI Fehler

**Beispiel:**
```bash
curl http://localhost:9001/api/v1/character/ships \
  -H "Authorization: Bearer <your_token>"
```

---

## Allgemeine Fehlerbehandlung

Alle Endpoints können folgende generische Fehler zurückgeben:

- `400 Bad Request`: Ungültige Eingabe
- `401 Unauthorized`: Authentifizierung fehlgeschlagen (nur protected endpoints)
- `429 Too Many Requests`: ESI Rate Limit erreicht
- `500 Internal Server Error`: Server-Fehler
- `503 Service Unavailable`: ESI nicht verfügbar

**Fehler Format:**
```json
{
  "error": "Kurze Fehlerbeschreibung",
  "details": "Detaillierte Fehlermeldung (optional)"
}
```

## Authentifizierung

Protected Endpoints erfordern einen gültigen EVE SSO Access Token:

1. Führe EVE SSO Login Flow durch
2. Erhalte Access Token
3. Sende Token im Authorization Header: `Authorization: Bearer <token>`

**Erforderliche Scopes:**
- `esi-location.read_location.v1` (für `/character/location`)
- `esi-location.read_ship_type.v1` (für `/character/ship`)
- `esi-assets.read_assets.v1` (für `/character/ships`)

## Rate Limits

- ESI Rate Limit: 10 Requests/Sekunde (konfigurierbar)
- Error Threshold: 15 Fehler bevor Circuit Breaker aktiviert
- Max Retries: 3

## Caching

- Market Orders: 5 Minuten In-Memory Cache
- Character Context: Kein Cache (immer frisch von ESI)

## Simplified Algorithm (Phase 2)

Diese Implementierung nutzt einen vereinfachten Algorithmus:

- **Max Market Pages**: 10 (statt alle 383)
- **Verarbeitung**: Sequenziell (kein Worker Pool)
- **Cache**: In-Memory (kein Redis in Phase 2)
- **Max Routes**: 50
- **Min Spread**: 5%
- **Travel Time**: ~30 Sekunden pro Jump (vereinfacht)
- **Performance**: < 60 Sekunden Berechnungszeit

**Performance-Optimierung erfolgt in Phase 3:**
- Worker Pool Parallelisierung
- Redis Cache
- Timeout Handling (HTTP 206)
- Alle 383 ESI Pagination Pages
