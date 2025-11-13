# Manual Testing Guide - Character Integration für Intra-Region Trading

## Setup

1. Start the backend API:
   ```bash
   cd backend
   go run ./cmd/api
   ```

2. Start the frontend:
   ```bash
   cd frontend
   npm run dev
   ```

## Test Cases

### Test 1: Region Endpoint Functionality
**Goal:** Verify that the new `/api/v1/sde/regions` endpoint returns all EVE regions.

**Steps:**
1. Make a GET request to `http://localhost:9001/api/v1/sde/regions`
2. Verify response contains a `regions` array with region objects
3. Check that each region has `id` and `name` fields
4. Verify regions are sorted alphabetically by name

**Expected Result:**
```json
{
  "regions": [
    {"id": 10000001, "name": "Derelik"},
    {"id": 10000002, "name": "The Forge"},
    ...
  ],
  "count": 100+
}
```

**Command:**
```bash
curl http://localhost:9001/api/v1/sde/regions | jq
```

---

### Test 2: Non-Authenticated User Experience
**Goal:** Verify that unauthenticated users see default values and standard industrial ships.

**Steps:**
1. Open browser in incognito mode
2. Navigate to `http://localhost:9000/intra-region`
3. Verify region dropdown loads all regions from backend
4. Verify "The Forge" (10000002) is pre-selected
5. Verify ship dropdown shows 8 standard industrial ships (Badger, Tayra, etc.)
6. Verify default ship "Badger" (648) is selected
7. Click "Berechnen" to verify route calculation works

**Expected Behavior:**
- Region dropdown: All EVE regions, sorted alphabetically
- Default region: "The Forge"
- Ship dropdown: Standard industrial haulers
- Default ship: "Badger"
- No authentication-related errors in console

---

### Test 3: Authenticated User - Character Location
**Goal:** Verify that authenticated users see their current clone location's region pre-selected.

**Prerequisites:**
- Valid EVE SSO login
- Character with `esi-location.read_location.v1` scope
- Character must be in a system (not in wormhole space)

**Steps:**
1. Login to EVE SSO via the app
2. Navigate to `http://localhost:9000/intra-region`
3. Observe loading state while character data loads
4. Verify region dropdown shows character's current region pre-selected
5. Note the region shown should match character's current system location

**Expected Behavior:**
- Brief loading indicator: "Lade Character-Daten..."
- Region auto-selected based on character's solar system
- If character is in Jita → "The Forge" selected
- If character is in Amarr → "Domain" selected
- Console logs show successful API calls

**Verification:**
Check browser console for:
```
[AuthContext] Token verified, character: <CharacterName>
```

---

### Test 4: Authenticated User - Current Ship
**Goal:** Verify that authenticated users see their current active ship pre-selected.

**Prerequisites:**
- Valid EVE SSO login
- Character with `esi-location.read_ship_type.v1` scope
- Character must be in a ship (not in a pod)

**Steps:**
1. Login to EVE SSO
2. Navigate to `http://localhost:9000/intra-region`
3. Wait for character data to load
4. Verify ship dropdown shows character's current ship pre-selected
5. Verify cargo capacity is shown correctly (e.g., "Badger (15k m³)")

**Expected Behavior:**
- Ship dropdown shows character's active ship selected
- Cargo capacity matches the ship's actual capacity
- Ship name format: `{ship_name} ({cargo_capacity}k m³)`

---

### Test 5: Authenticated User - Available Ships
**Goal:** Verify that authenticated users only see their owned ships in the dropdown.

**Prerequisites:**
- Valid EVE SSO login
- Character with assets in hangars
- Multiple ships owned by the character

**Steps:**
1. Login to EVE SSO
2. Navigate to `http://localhost:9000/intra-region`
3. Open ship dropdown
4. Verify only ships in character's hangars are shown
5. Verify all ships have correct cargo capacity displayed

**Expected Behavior:**
- Ship list contains only character's ships (from `/api/v1/character/ships`)
- Each ship shows: name and cargo capacity
- Ships from mock data should NOT appear for authenticated users
- If character has no ships → fallback to standard industrial ships

---

### Test 6: Error Handling - API Unavailable
**Goal:** Verify graceful degradation when backend APIs fail.

**Steps:**
1. Stop the backend server
2. Navigate to `http://localhost:9000/intra-region` (unauthenticated)
3. Verify region dropdown falls back to hardcoded list
4. Verify page remains functional

**Expected Behavior:**
- Region dropdown: Shows 15 hardcoded regions from `mock-data/regions.ts`
- Ship dropdown: Shows standard industrial ships
- Console error logged but page doesn't break
- User can still interact with the page

---

### Test 7: Error Handling - Character API Fails
**Goal:** Verify fallback when character APIs fail for authenticated user.

**Steps:**
1. Login to EVE SSO
2. Simulate API failure (e.g., revoke scopes or expired token)
3. Navigate to `http://localhost:9000/intra-region`
4. Verify defaults are used instead of crashing

**Expected Behavior:**
- Console shows error: "Failed to load character data"
- Region defaults to "The Forge"
- Ship defaults to "Badger"
- Standard ship list shown instead of character ships
- Page remains functional

---

### Test 8: Ship Change Propagation
**Goal:** Verify that changing ships in-game updates the selection after refresh.

**Steps:**
1. Login and note current ship selection
2. Change ship in EVE Online client
3. Refresh the `/intra-region` page
4. Verify new ship is pre-selected

**Expected Behavior:**
- New active ship should be selected automatically
- Character ships list should be updated (if different ships now available)

---

### Test 9: Loading States
**Goal:** Verify all loading states display correctly.

**Steps:**
1. Login to EVE SSO
2. Navigate to `/intra-region`
3. Observe loading indicators during:
   - Initial page load
   - Character data fetch
   - Region list fetch
   - Ship list fetch

**Expected Behavior:**
- Region dropdown: "Lade Regionen..." placeholder while loading
- Ship dropdown: "Lade Schiffe..." placeholder while loading
- Calculate button: "Lade Character-Daten..." with spinner icon
- All disabled during loading

---

### Test 10: Route Calculation Integration
**Goal:** Verify route calculation still works with character-selected region/ship.

**Steps:**
1. Login (or use as guest)
2. Select region (should be auto-selected if authenticated)
3. Select ship (should be auto-selected if authenticated)
4. Click "Berechnen"
5. Verify routes are calculated correctly

**Expected Behavior:**
- Routes calculated for selected region + ship
- Route list displays with proper formatting
- ISK/hour, profit, spread, travel time shown correctly
- Filters work as expected

---

## Browser Console Checks

### Expected Console Logs (Authenticated)
```
[AuthContext] Checking session, token: exists
[AuthContext] Verifying token with EVE ESI...
[AuthContext] Token verified, character: YourCharacterName
[AuthContext] Session set, authenticated: true
```

### Expected API Calls
1. `GET /api/v1/sde/regions` - Fetch all regions
2. `GET /api/v1/character/location` - Get character location (authenticated)
3. `GET /api/v1/character/ship` - Get current ship (authenticated)
4. `GET /api/v1/character/ships` - Get owned ships (authenticated)
5. `POST /api/v1/trading/routes/calculate` - Calculate routes

---

## Known Limitations

1. **Region List:** Requires backend database connection to SDE
2. **Character Ships:** Only shows ships in hangars (excludes ships in space)
3. **Wormhole Systems:** May not have a valid region_id (fallback to default)
4. **Structure IDs:** Station names fetched from SDE, structure names may show as IDs

---

## Troubleshooting

### Region dropdown shows only 15 regions
**Cause:** Backend endpoint failed or returned error  
**Fix:** Check backend logs, verify SDE database connection

### Ship dropdown shows fallback ships for authenticated user
**Cause:** Character has no ships or API call failed  
**Fix:** Check console for errors, verify ESI scopes are correct

### Character data doesn't load
**Cause:** Auth token expired or invalid scopes  
**Fix:** Re-login via EVE SSO

### "Lade Character-Daten..." stuck
**Cause:** Backend API timeout or network issue  
**Fix:** Check backend health, network connectivity

---

## Success Criteria

✅ All regions loaded from backend (100+ regions)  
✅ Non-authenticated users see defaults (The Forge + Badger)  
✅ Authenticated users see their region auto-selected  
✅ Authenticated users see their current ship auto-selected  
✅ Authenticated users see only their ships in dropdown  
✅ Loading states display correctly  
✅ Error handling works (graceful fallback)  
✅ Route calculation works with character-selected values  
✅ No TypeScript errors  
✅ No console errors in normal flow
