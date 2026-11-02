---
status: active
---

# Verfassung — Check_Cert

Diese Verfassung enthält die systemweiten, dauerhaften Regeln des Projekts. Feature-Inhalte (User Stories, Tasklisten, Planung) gehören in PRD, Tech Spec und Plan. Jede Änderung einer Regel verweist auf ein ADR unter `docs/adr`.

## Architektur/Layering

1. Das Projekt besteht aus zwei getrennten Programmen, CheckCert und SendMail. Sie tauschen Daten ausschließlich über Dateien `<recipient>.SendMail` in einem Verzeichnis aus.
2. Beide Programme folgen MVC mit den Schichten Model, View (Konsole bzw. Log), Controller und Service.
3. Erlaubte Abhängigkeiten zwischen den Schichten: Controller nutzt Model und Service; Service nutzt Model; Model hängt von keiner anderen Schicht ab; keine Schicht greift auf den Controller zu.
4. Die Programme sind in Go geschrieben und werden je als statisches Binary ohne Laufzeitabhängigkeiten ausgeliefert.
5. Zielplattform ist Linux; gebaut wird mit Go 1.24 oder neuer.
6. Version und Build-Datum werden per `-ldflags` in `internal/config` gesetzt.
7. Für Einstellungen gilt die Rangfolge: Kommandozeile vor Umgebungsvariable vor `.env`-Datei.

## Security

1. Zugangsdaten werden nie über eine unverschlüsselte Verbindung gesendet; einzige Ausnahme ist localhost.
2. SMTP läuft über STARTTLS oder über SSL/TLS auf Port 465; ohne TLS nur an localhost.
3. Echte Zugangsdaten werden nie ins Repository eingecheckt; eingecheckt werden nur Beispieldateien wie `docs/SendMail.env` mit Platzhaltern.

## Strikte Regeln

1. Eine `.SendMail`-Datei wird atomar geschrieben (temporäre Datei, dann rename).
2. Kein Datenverlust: Eine `.SendMail`-Datei wird nur gelöscht, wenn der Versand erfolgreich war und sie sich währenddessen nicht geändert hat.
3. Die Exit-Codes von CheckCert (0, 1, 2, 3) sind eine feste Schnittstelle.
4. Jeder Test nennt sein Akzeptanzkriterium in der Zeile direkt darüber im Format `// <slug> AC-<n>: …`.
5. Vor jedem Merge muss `make test` grün sein.
6. Vor jedem Merge prüft ein Mensch die Änderung (Review).
