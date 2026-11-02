# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.0] - 2026-11-02

### Added

- **SendMail**: Go port of the C++/Qt SendMail application.
  - Watches `WATCH_DIR` (polling, works on network shares and removable media) for `<recipient>.SendMail` files.
  - Mails the file content to the recipient taken from the file name and deletes the file after success.
  - STARTTLS (port 587) and implicit SSL/TLS (port 465), `AUTH LOGIN` with fallback to `AUTH PLAIN`.
  - Credentials are never sent over unencrypted connections (except to localhost).
  - Stabilization period (`STABLE_SECONDS`) before processing a file.
  - Retry back-off after failures (`RETRY_SECONDS`) to avoid hammering the SMTP server.
  - Files modified while sending are kept (no data loss).
  - `-once` mode for cron, graceful shutdown on SIGINT/SIGTERM.
  - `SendMail.env` compatible with the C++ version.
  - systemd user unit example (`docs/sendmail.service`).
- Unit tests including a fake SMTP server (STARTTLS + AUTH LOGIN).
- Release workflow builds both binaries for linux/amd64 and linux/arm64.

### Changed

- All program output, messages and comments are in English (including the notification text).
- Shared application information (`internal/config/rz_config.go`) for both executables.
- `ConfigModel` renamed to `CheckConfigModel`.

## [0.1.0] - 2026-11-02

### Added

- **CheckCert**: Go console application to check TLS certificate expiry of a list of websites.
  - Website list supports `host`, `host:port` and URLs, `#` comments and blank lines.
  - Parallel checks, SNI, expiry date is read even for expired or untrusted certificates.
  - Creates/extends `<recipient>.SendMail` atomically if a certificate expires within `WARN_DAYS` days.
  - Configuration via `CheckCert.env`, overridable by command line options.
  - Exit codes: `0` ok, `1` warning, `2` usage error, `3` website not checkable.
- `check_cert.sh`: Bash prototype based on `openssl`.

[Unreleased]: https://github.com/Zheng-Bote/check-cert/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/Zheng-Bote/check-cert/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/Zheng-Bote/check-cert/releases/tag/v0.1.0
