---
name: Feature Request
about: Neue Funktionalität vorschlagen
labels: feat
---

# Feature Request

## Kontext
<!-- Kurzbeschreibung des Problems oder der Gelegenheit -->

## Architektur-Kontext

**Betroffene Schicht(en):** (mehrfach auswählbar)
- [ ] Frontend (Next.js)
- [ ] Backend (Go API)
- [ ] Datenbank (PostgreSQL)
- [ ] Infrastructure (Docker)
- [ ] Sonstiges: _______________

**Architektur-Entscheidungen erforderlich:**
- [ ] Nein - Standard-Pattern anwendbar
- [ ] Ja - Neue ADR muss erstellt werden

**Falls "Ja", beschreibe Architektur-Frage:**
<!-- Beispiel: "Wo soll OAuth implementiert werden? Frontend (PKCE) vs Backend (Client Secret)?" -->

**Relevante bestehende ADRs:**
<!-- z. B. ADR-001 (Tech Stack), ADR-003 (Frontend OAuth) -->
- (keine) oder ADR-XXX: [Titel]

## Technische Alternativen (falls mehrere Ansätze möglich)

**Gibt es mehrere Implementierungsansätze?**
- [ ] Nein - Klarer Standard-Ansatz
- [ ] Ja - Alternativen dokumentiert unten

<!-- Falls "Ja", beschreibe Alternativen: -->

<details>
<summary>Option A: [Titel]</summary>

**Beschreibung:**  
<!-- Kurze Beschreibung des Ansatzes -->

**Vorteile:**
- 

**Nachteile:**
- 

**Code-Aufwand:** ~ Zeilen / ~ Stunden

</details>

<details>
<summary>Option B: [Titel]</summary>

**Beschreibung:**  
<!-- Kurze Beschreibung des Ansatzes -->

**Vorteile:**
- 

**Nachteile:**
- 

**Code-Aufwand:** ~ Zeilen / ~ Stunden

</details>

**Empfohlene Option:** [ ] A / [ ] B - Begründung: ...

---

## Ziel / Nutzen
<!-- Welchen Mehrwert schafft das Feature? -->

## Akzeptanzkriterien

- [ ] (Kriterium 1)
- [ ] (Kriterium 2)

## Nicht-Ziele / Abgrenzung

- (Explizit ausgeschlossen)

## Relevante ADR Referenzen

<!-- z. B. ADR-005, ADR-008 -->
<!-- Bei Architektur-Entscheidungen: ADR muss ZUERST erstellt werden -->
- (keine) oder ADR-XXX: [Titel]

## Daten / Schnittstellen / Domäne (Kurzdesign)

```text
Skizzen / Sequenzen / Datenstrukturen
```

## Edge Cases

-

## Risiken / Annahmen

-

## Teststrategie Hinweise

- Unit:
- Integration:
- E2E:

## Follow-ups (falls ausgelagert)

-

## Dokumentations-Updates (Pflicht vor Abschluss)

- [ ] Relevante Pläne (z. B. `docs/implementation-plan-*.md`) aktualisiert
- [ ] Eintrag in `CHANGELOG.md` ergänzt
- [ ] Benutzer- oder Betriebsdokumentation aktualisiert (README, Runbooks, ADR-Hinweise)

<!-- Pflicht: Issue vor Umsetzung verfeinern, unklare Punkte klären -->
