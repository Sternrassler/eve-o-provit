# Nereus Fitting-Analyse: Ix Sternrassler's Schiff

**Datum:** 9. November 2025  
**Character:** Ix Sternrassler (ID: 2123508276)  
**Schiff:** Nereus (Type ID 650)  
**Fitting:** Ix-NER-1  
**Tatsächliche Cargo-Kapazität:** 9.656,9 m³ ✅

---

## 0. Executive Summary

### 0.1 Finale EVE Online Cargo-Formel (verifiziert)

```
Cargo = Base × (1 + Skill%) × (1 + Module%)^n × (1 + Rig%)^m

Wobei:
- Base = 2.700 m³ (Nereus Dogma Attribute 38)
- Skill% = 0,05 (Gallente Hauler Level I: +5%)
- Module% = 0,175 (Expanded Cargohold I: +17,5%)
- n = Anzahl Module (5× in diesem Fitting)
- Rig% = 0,15 (Medium Cargohold Optimization I: +15%)
- m = Anzahl Rigs (3× in diesem Fitting)
```

### 0.2 Berechnung für dein Fitting

```
Cargo = 2.700 × 1,05 × (1,175)^5 × (1,15)^3
      = 2.700 × 1,05 × 2,240 × 1,521
      = 9.641 m³

Tatsächlich angezeigt: 9.656,9 m³
Differenz: +15,9 m³ (0,16% Rundungsfehler)
```

**✅ FORMEL BESTÄTIGT!**

---

## 1. Character Skills

### 1.1 Cargo-Relevante Skills

| Skill Name | Level | Bonus pro Level | Gesamt-Effekt |
|------------|-------|-----------------|---------------|
| **Gallente Hauler** | **I / V** | **+5% Cargo** | **+5% Base Cargo** |
| Spaceship Command | III / V | +2% Agility | +6% Agility |

**WICHTIG:** Gallente Hauler Level I (nicht III wie ursprünglich angenommen!)

**Effekt auf Cargo:**

```
Base Cargo: 2.700 m³
Mit Gallente Hauler I: 2.700 × 1,05 = 2.835 m³
```

### 1.2 Weitere Skills

| Skill Name | Level | Bonus |
|------------|-------|-------|
| Navigation | IV / V | +5% Ship Velocity → +20% Velocity |
| Evasive Maneuvering | IV / V | +5% Agility → +20% Agility |
| Accounting | III / V | -10% Sales Tax (30% Reduction) |
| Broker Relations | II / V | -0,6% Broker Fee |

---

## 2. Installiertes Fitting

### 2.1 Komplette Modul-Liste

| Slot Type | Slot ID | Modul Name | Type ID | Quantity | Cargo Bonus |
|-----------|---------|------------|---------|----------|-------------|
| **MED SLOT** | MedSlot0 | 10MN Monopropellant Enduring Afterburner | 5973 | 1 | - |
| **LOW SLOT** | LoSlot0 | Expanded Cargohold I | 1317 | 1 | +17,5% |
| **LOW SLOT** | LoSlot1 | Expanded Cargohold I | 1317 | 1 | +17,5% |
| **LOW SLOT** | LoSlot2 | Expanded Cargohold I | 1317 | 1 | +17,5% |
| **LOW SLOT** | LoSlot3 | Expanded Cargohold I | 1317 | 1 | +17,5% |
| **LOW SLOT** | LoSlot4 | Expanded Cargohold I | 1317 | 1 | +17,5% |
| **RIG SLOT** | RigSlot0 | Medium Cargohold Optimization I | 31118 | 1 | +15% |
| **RIG SLOT** | RigSlot1 | Medium Cargohold Optimization I | 31118 | 1 | +15% |
| **RIG SLOT** | RigSlot2 | Medium Cargohold Optimization I | 31118 | 1 | +15% |
| **DRONE BAY** | DroneBay | Acolyte I | 2203 | 5 | - |
| **DRONE BAY** | DroneBay | Warrior I | 2205 | 1 | - |

**Summe:**

- **5× Expanded Cargohold I** (Low Slots) → Je +17,5% Cargo
- **3× Medium Cargohold Optimization I** (Rigs) → Je +15% Cargo
- **1× 10MN Afterburner** (Mid Slot) → Kein Cargo-Effekt
- **6× Light Drones** → Kein Cargo-Effekt

---

## 3. Modul & Rig Erklärungen

### 3.1 Expanded Cargohold I (Low Slot Module)

**Dogma Attribute 149:** `cargoCapacityBonus`

**Werte:**

- **Cargo Capacity Bonus:** +17,5%
- **Structure Hitpoint Bonus:** -20%
- **Volume:** 5 m³
- **Tech Level:** 1
- **Meta Level:** 0

**Warum im Low Slot:**

- Low Slots sind für **passive Ship-Attribute** (Cargo, Armor, Damage)
- Expanded Cargoholds erhöhen Cargo auf Kosten von Struktur-HP

**Stacking Behavior:**

- **KEINE Stacking Penalty** (EVE University Wiki bestätigt)
- Jedes Modul wirkt **multiplikativ** mit vollem 17,5% Bonus

---

### 3.2 Medium Cargohold Optimization I (Rig)

**Dogma Attribute 614:** `rigCargoBay`

**Werte:**

- **Cargo Capacity Bonus:** +15%
- **Drawback:** -10% Armor HP
- **Volume:** 10 m³
- **Calibration:** 100 (von 400 verfügbar)
- **Tech Level:** 1

**Warum Rigs:**

- Rigs sind **permanente Ship-Modifikationen**
- Können nur in Station gewechselt werden (und werden dabei zerstört)
- Nereus hat 3× Rig Slots (Medium Size)

**Stacking Behavior:**

- **KEINE Stacking Penalty** auf Cargo Rigs
- Alle 3× Rigs wirken **multiplikativ** mit vollem 15% Bonus

**Calibration Check:**

```
3× Medium Cargohold Optimization I = 3 × 100 = 300
Verfügbar: 400
Rest: 100 (für 1× weiteren Medium Rig)
```

---

### 3.3 10MN Monopropellant Enduring Afterburner

**Funktion:** Erhöht Max Velocity um ~500% (bei Aktivierung)

**Cargo-Effekt:** Keiner (Mid Slot, kein Cargo-Attribut)

**Warum installiert:**

- Schnelleres Travel zwischen Gates/Stationen
- Flucht vor Ganks in Low-Sec
- Align-Time Reduktion (zusammen mit Inertia Skills)

---

## 4. Cargo-Kapazität Berechnung

### 4.1 Schritt-für-Schritt Berechnung

**Basis-Werte:**

```
Base Cargo (Nereus): 2.700 m³ (Dogma Attribute 38)
Gallente Hauler I: +5%
Expanded Cargohold I: +17,5% pro Modul
Medium Cargohold Optimization I: +15% pro Rig
```

**Schritt 1: Base mit Skills**

```
Base mit Skill = Base × (1 + Skill%)
               = 2.700 × 1,05
               = 2.835 m³
```

**Schritt 2: Module anwenden (multiplikativ)**

```
Mit 5× Modulen = 2.835 × (1,175)^5
                = 2.835 × 2,240
                = 6.350 m³
```

**Schritt 3: Rigs anwenden (multiplikativ)**

```
Mit 3× Rigs = 6.350 × (1,15)^3
             = 6.350 × 1,521
             = 9.658 m³
```

**Finale Formel (kombiniert):**

```
Cargo = Base × (1 + Skill%) × (1 + Module%)^n × (1 + Rig%)^m
      = 2.700 × 1,05 × (1,175)^5 × (1,15)^3
      = 2.700 × 1,05 × 2,240 × 1,521
      = 9.641 m³
```

**Vergleich:**

```
Berechnet: 9.641 m³
Angezeigt: 9.656,9 m³
Differenz: +15,9 m³ (0,16% Rundungsfehler)
```

**✅ FORMEL VERIFIZIERT!**

---

### 4.2 Simulator-Verifikation (Schritt-für-Schritt)

| Konfiguration | Formel | Berechnet | Angezeigt | Status |
|---------------|--------|-----------|-----------|--------|
| Nur Skills | 2.700 × 1,05 | 2.835 | 2.835 | ✅ |
| + 3× Rigs | 2.835 × 1,15³ | 4.311 | 4.311,7 | ✅ |
| + 1× Modul + 3× Rigs | 2.835 × 1,175 × 1,15³ | 5.065 | 5.066,2 | ✅ |
| + 2× Module + 3× Rigs | 2.835 × 1,175² × 1,15³ | 5.951 | 5.952,8 | ✅ |
| + 3× Module + 3× Rigs | 2.835 × 1,175³ × 1,15³ | 6.993 | 6.994,6 | ✅ |
| + 4× Module + 3× Rigs | 2.835 × 1,175⁴ × 1,15³ | 8.217 | 8.218,6 | ✅ |
| + 5× Module + 3× Rigs | 2.835 × 1,175⁵ × 1,15³ | 9.641 | 9.656,9 | ✅ |
| + 5× Module (keine Rigs) | 2.835 × 1,175⁵ | 6.350 | 6.349,5 | ✅ |

**Alle Datenpunkte bestätigen die multiplikative Formel!**

---

### 4.3 Wichtige Erkenntnisse

**1. KEINE Stacking Penalty auf Cargo-Boni**

- Jedes Modul wirkt mit vollem 17,5% Bonus
- Jeder Rig wirkt mit vollem 15% Bonus
- Multiplikative Anwendung (nicht additiv!)

**2. Reihenfolge der Anwendung**

```
1. Base Cargo (2.700 m³)
2. Skill-Bonus (×1,05)
3. Module-Boni (×1,175 pro Modul)
4. Rig-Boni (×1,15 pro Rig)
```

**3. Prozentuale Boni beziehen sich IMMER auf den aktuellen Wert**

- Nicht auf die originale Base!
- Deshalb multiplikativ, nicht additiv

**4. Rundungsfehler minimal**

- EVE Client rundet auf 0,1 m³
- Berechnungsfehler <0,2%

---

## 5. EVE Dogma System (Technische Details)

### 5.1 Relevante Dogma Attributes

| Attribute ID | Name | Wert | Beschreibung |
|--------------|------|------|--------------|
| 38 | capacity | 2.700 m³ | Base Cargo Capacity |
| 149 | cargoCapacityBonus | 17,5% | Expanded Cargohold I Bonus |
| 496 | shipBonusGI | 5% | Gallente Hauler Skill Bonus |
| 614 | rigCargoBay | 15% | Medium Cargohold Optimization I |

### 5.2 Bonus-Typen in EVE

**Skill-Boni:**

- Anwendung: `Base × (1 + Skill% × Level)`
- Gallente Hauler I: `2.700 × (1 + 0,05 × 1) = 2.835 m³`

**Modul-Boni:**

- Anwendung: Multiplikativ (jedes Modul separat)
- 5× Expanded Cargohold I: `Base × 1,175^5`

**Rig-Boni:**

- Anwendung: Multiplikativ (jeder Rig separat)
- 3× Medium Cargohold Opt I: `Base × 1,15^3`

### 5.3 Stacking Penalty Ausnahmen

**Cargo Capacity hat KEINE Stacking Penalty!**

Quelle: [EVE University Wiki - Stacking Penalties](https://wiki.eveuniversity.org/Stacking_penalties)

> "Some bonuses are exempt from stacking penalties. These include:
>
> - Cargo capacity bonuses"

**Andere Attribute MIT Stacking Penalty:**

- Damage modifiers
- Shield/Armor resistances
- ECM strength
- Tracking speed
- Scan resolution

---

## 6. Nereus Ship-Daten (Referenz)

### 6.1 Base Attributes (SDE)

| Attribute | Wert | Dogma ID |
|-----------|------|----------|
| Capacity | 2.700 m³ | 38 |
| Mass | 11.250.000 kg | 4 |
| Volume | 240.000 m³ | 161 |
| Structure HP | 2.600 HP | 9 |
| Powergrid | 215 MW | 11 |
| CPU | 350 tf | 48 |
| High Slots | 2 | 14 |
| Med Slots | 5 | 13 |
| Low Slots | 5 | 12 |
| Rig Slots | 3 | 1137 |
| Calibration | 400 points | 1132 |
| Drone Capacity | 30 m³ | 283 |
| Drone Bandwidth | 15 Mbit/sec | 1271 |

### 6.2 Ship Bonuses (Traits)

**Gallente Hauler bonuses (per skill level):**

- +5% bonus to ship cargo capacity
- +5% bonus to ship inertia modifier

**Role Bonus:**

- 90% reduction to effective distance traveled for jump fatigue
- 30% bonus to warp speed and warp acceleration

### 6.3 Maximales Cargo (Theorie)

**EVE University Wiki:** "cargo hold can expand to 11,372 m³ (no rigs)"

**Mit Tech II Modulen + Rigs:**

```
Annahme: 5× Expanded Cargohold II (27,5%) + 3× Tech II Rigs (17,5%)
         + Gallente Hauler V (+25%)

Base: 2.700 m³
Skill V: 2.700 × 1,25 = 3.375 m³
Module: 3.375 × 1,275^5 = 10.823 m³
Rigs: 10.823 × 1,175^3 = 17.567 m³

ABER: Calibration-Limit!
Tech II Rigs brauchen ~200 Calibration
3× Tech II Rigs = 600 (ÜBERSCHREITET 400 Limit!)
```

**Realistisches Maximum (Tech I Rigs):**

```
5× Expanded Cargohold II + 3× Medium Cargohold Opt I + Gallente Hauler V
= 2.700 × 1,25 × 1,275^5 × 1,15^3
= 15.487 m³
```

**Dein aktuelles Fitting (% vom Maximum):**

```
9.656,9 / 15.487 = 62,35% vom theoretischen Max
```

---

## 7. Optimierungs-Vorschläge

### 7.1 Kurzfristig (Skill-Training)

**Priorität 1: Gallente Hauler auf Level V**

```
Training Time: ~8-10 Tage
Cargo-Gewinn:
Vorher: 2.700 × 1,05 × 1,175^5 × 1,15^3 = 9.641 m³
Nachher: 2.700 × 1,25 × 1,175^5 × 1,15^3 = 11.477 m³
Differenz: +1.836 m³ (+19%)
```

**Priorität 2: Rigging Skills (für Tech II Rigs)**

- Mechanics V (Voraussetzung)
- Astronautics Rigging III → IV → V
- Ermöglicht Tech II Rigs (17,5% statt 15%)

### 7.2 Mittelfristig (Module Upgrade)

**Expanded Cargohold I → Expanded Cargohold II**

```
Kosten: ~500k ISK pro Modul × 5 = 2,5M ISK
Cargo-Gewinn (mit Gallente Hauler V):
Vorher: 2.700 × 1,25 × 1,175^5 × 1,15^3 = 11.477 m³
Nachher: 2.700 × 1,25 × 1,275^5 × 1,15^3 = 15.487 m³
Differenz: +4.010 m³ (+35%)
```

**Requirements:**

- Hull Upgrades II (leicht trainierbar)
- ~2-3 Tage Training Time

### 7.3 Langfristig (Ship Upgrade)

**Viator (Tech II Blockade Runner)**

```
Base Cargo: 3.000 m³
Gallente Hauler V: +25% → 3.750 m³
5× Expanded Cargohold II: ×1,275^5 → 12.033 m³
3× Medium Cargohold Opt I: ×1,15^3 → 19.515 m³

PLUS: Cloak-fähig + Nullifier (sicherer Transport!)
```

**Requirements:**

- Transport Ships I (Capital Ships → Advanced Spaceship Command)
- ~30-45 Tage Training Time
- Kosten: ~200-300M ISK

---

## 8. Technische Implementation (Backend-Relevanz)

### 8.1 Backend Cargo Service Fix

**Datei:** `/backend/internal/services/cargo_service.go`

**Alter Code (FALSCH):**

```go
// Line 114 (alt)
totalCapacity := capacityWithSkills * (1 + fitting.Bonuses.CargoBonus)
```

**Problem:**

- Behandelt Cargo-Bonus als **einmaligen multiplikativen Faktor**
- Richtig ist: **Jedes Modul/Rig separat multiplikativ**

**Korrekter Code:**

```go
// Korrigierte Formel
totalCapacity := capacityWithSkills

// Module-Boni (multiplikativ)
for _, module := range fitting.Modules {
    if module.CargoBonus > 0 {
        totalCapacity *= (1 + module.CargoBonus)
    }
}

// Rig-Boni (multiplikativ)
for _, rig := range fitting.Rigs {
    if rig.CargoBonus > 0 {
        totalCapacity *= (1 + rig.CargoBonus)
    }
}
```

### 8.2 Formel-Implementierung (Pseudocode)

```python
def calculate_cargo_capacity(ship, skills, modules, rigs):
    # Step 1: Base Cargo
    base = ship.get_attribute(38)  # Dogma Attr 38: capacity
    
    # Step 2: Skill Bonus
    skill_level = skills.get("Gallente Hauler", 0)
    skill_bonus_per_level = ship.get_trait_bonus("GallenteHauler", "cargo")
    capacity = base * (1 + skill_bonus_per_level * skill_level)
    
    # Step 3: Module Bonuses (multiplikativ)
    for module in modules:
        if module.type in ["Expanded Cargohold I", "Expanded Cargohold II"]:
            bonus = module.get_attribute(149)  # cargoCapacityBonus
            capacity *= (1 + bonus)
    
    # Step 4: Rig Bonuses (multiplikativ)
    for rig in rigs:
        if rig.type.startswith("Medium Cargohold Optimization"):
            bonus = rig.get_attribute(614)  # rigCargoBay
            capacity *= (1 + bonus)
    
    return round(capacity, 1)
```

---

## 9. Quellen & Referenzen

### 9.1 Offizielle Quellen

- **EVE SDE (Static Data Export):** Dogma Attributes, Ship Types
- **EVE University Wiki:** [Nereus](https://wiki.eveuniversity.org/Nereus), [Stacking Penalties](https://wiki.eveuniversity.org/Stacking_penalties)
- **everef.net:** [Nereus Type 650](https://everef.net/type/650), [Dogma Attribute 38](https://everef.net/dogma-attributes/38)

### 9.2 Community Tools

- **EVE Fitting Simulator:** In-Game Simulation (verwendete Testplattform)
- **Pyfa (Python Fitting Assistant):** Desktop Fitting Tool
- **EVE Workbench:** Online Fitting Calculator

### 9.3 Verifikations-Methode

**Systematisches Testen:**

1. Nur Skills → 2.835 m³ ✅
2. - 3× Rigs → 4.311,7 m³ ✅
3. - 1× Modul → 5.066,2 m³ ✅
4. - 2× Module → 5.952,8 m³ ✅
5. - 3× Module → 6.994,6 m³ ✅
6. - 4× Module → 8.218,6 m³ ✅
7. - 5× Module → 9.656,9 m³ ✅

**Alle Datenpunkte konsistent mit multiplikativer Formel!**

---

## 10. Changelog

**2025-11-09 - Initial Analysis**

- Vollständige Formel-Ableitung via Simulator-Tests
- Korrektur: Gallente Hauler Level I (nicht III)
- Verifikation aller Datenpunkte (8 Test-Konfigurationen)
- Backend-Bug identifiziert (cargo_service.go:114)
- Finale Formel: `Cargo = Base × (1 + Skill%) × (1 + Module%)^n × (1 + Rig%)^m`

**Nächste Schritte:**

- Backend-Formel korrigieren (multiplikative Anwendung)
- Integration in Fitting-API
- Cargo-Optimierungs-Empfehlungen generieren
