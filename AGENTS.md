<!-- sdd:agents -->
# Hinweise für Agenten

Dieses Projekt arbeitet nach Spec-Driven Development und wird mit `sdd` geprüft. Diese Datei gilt für jeden Agenten; `CLAUDE.md` und `GEMINI.md` verweisen hierher.

## Verfassung zuerst

[`constitution.md`](constitution.md) ist das ranghöchste Artefakt. Ihre Regeln gelten für jede Änderung, ohne Ausnahme. Lies sie vor jeder Arbeit.

Bei einem Widerspruch gilt diese Reihenfolge: Verfassung → ADR (`docs/adr`) → Tech Spec (`docs/design`) → PRD (`docs/prd`) → Plan (`docs/plans`). Zwischen gleich bindenden Regeln nicht entscheiden, sondern nachfragen.

## Ablauf

Idee → PRD → Tech Spec (wenn nötig) → Plan → Tests je Kriterium → Umsetzung → `sdd gate pre-merge` → Merge → Release → Archiv.

- Vor einer Änderung: `sdd next` zeigt den Stand und den nächsten Schritt.
- Nach einer Änderung: `sdd lint`, vor dem Merge `sdd gate pre-merge`.
- Jeder Test nennt sein Kriterium in der Zeile direkt darüber: `// <slug> AC-<n>: …` (Kommentarsyntax der Sprache).
- Eine Änderung der Verfassung braucht ein ADR, das sie in `references` nennt.
- Der Mensch entscheidet: Baseline (`sdd baseline`), ADR-Lockfile, `sdd adopt --apply` und das Entscheiden eines ADR (`status: accepted`).

## Werkzeuge

`sdd mcp` stellt die Prüfungen als MCP-Werkzeuge bereit (`sdd_next`, `sdd_lint`, `sdd_check_intent`, `sdd_context`, `sdd_validate_draft` …):

```json
{ "mcpServers": { "sdd": { "command": "sdd", "args": ["mcp"] } } }
```
