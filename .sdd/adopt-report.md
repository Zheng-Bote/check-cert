---
status: active
---

# Adopt-Bericht — Check_Cert

Angewendet am 2026-11-02 auf Branch `sdd/adopt` (ausgehend von `main`), Profil `standard`.

## Reifegrad: L0

| Stufe | Kriterium | Erfüllt | Befund |
|---|---|---|---|
| L1 | Verfassung vorhanden | nein | constitution.md |
| L1 | Verfassung ohne Befunde (SDD-001, SDD-002a) | nein | 1 Befunde |
| L2 | PRD und Tech Spec getrennt vorhanden | nein | 0 PRD, 0 Tech Spec |
| L2 | Akzeptanzkriterien mit Kennung | nein | 0 AC-Kennungen, 0 Befunde SDD-021 |
| L3 | CI ruft sdd gate auf | nein | 2 Workflow-Datei(en) |
| L3 | Baseline vorhanden | nein | .sdd/baseline.json |
| L4 | Feature-Artefakte tragen feature: und status | ja | 0 von 0 ohne beide Felder |
| L4 | Archiv konfiguriert | ja | docs/archive |
| L4 | Archivierung aktuell (SDD-050) | ja | 0 Befunde |

**Nächster Schritt (L1):** Verfassung constitution.md anlegen (Schritt „Verfassung“ von sdd adopt --apply).

Der Reifegrad beschreibt den Stand vor der Übernahme.

## Inventar

- Artefaktpfade: Standardpfade von sdd init
- Verfassung: 0
- ADR: 0
- PRD: 0
- Tech Spec: 0
- Plan: 0
- Doku/Exploration: 4
- CI-Workflows: 2 (ruft `sdd gate` auf: nein)

| Datei | Typ | Grund |
|---|---|---|
| `.gemini/GEMINI.md` | Doku/Exploration | kein Muster |
| `CHANGELOG.md` | Doku/Exploration | kein Muster |
| `README.md` | Doku/Exploration | kein Muster |
| `docs/index.md` | Doku/Exploration | kein Muster |

Die Zuordnung ist ein Vorschlag nach festen Mustern; „kein Muster“ heißt Doku/Exploration, nicht „geraten“.

## Schritte

1. **Konfiguration** — Commit `60f09c5`: sdd.config.yaml
2. **Verfassung** — Commit `872ee45`: constitution.md
3. **Baseline** — Commit `12fe30a`: .sdd/baseline.json, .sdd/adr-lock.json
4. **Bericht** — diese Datei

Baseline: 0 Einträge.

## Konflikte

Keine.

## Archivierung vor dem ersten Release

Kein Feature-Artefakt ist abgeschlossen oder abgelaufen.

## Hinweise

- Ergänzte Feature-Artefakte tragen `status: active`. Längst umgesetzte Features auf `status: implemented` setzen — dann weist sie SDD-050 für die Archivierung aus.
- Die Größenschwelle ist leer: SDD-020/030 verlangen erst dann PRD und Tech Spec, wenn `thresholds.layers` oder `touched_paths` gesetzt sind.
