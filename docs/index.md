# CheckCert & SendMail

This repository contains two small Go console applications (single static binaries, no runtime dependencies):

- **CheckCert** checks TLS certificates and creates `<recipient>.SendMail` files.
- **SendMail** watches a directory and mails the content of `<recipient>.SendMail` files (Go port of the C++/Qt [SendMail](https://github.com/Zheng-Bote/send-mail)).

```
websites.txt -> CheckCert -> out/zheng.bote+webcertificate@googlemail.com.SendMail -> SendMail -> SMTP
```

## CheckCert

CheckCert is a small Go console application that checks the TLS certificates of a list of websites. If a certificate expires within a configurable number of days (default: 5), it creates a file `<recipient>.SendMail` in an output directory. This file is picked up and mailed by [SendMail](https://github.com/Zheng-Bote/send-mail).

### Architecture

The application follows the **Model-View-Controller (MVC)** pattern:

- **Model (`internal/model`)**: `ConfigModel` loads and validates the configuration from `CheckCert.env`; `Website` / `CertResult` hold the data of a single check and build the notification text.
- **View**: Console output (stdout for status, stderr for errors).
- **Controller (`internal/controller`)**: `CheckController` reads the website list, runs the checks in parallel, evaluates the results and writes the SendMail file.
- **Service (`internal/service`)**: `CertService` connects via TLS (with SNI) and reads the expiry date of the server certificate.
- **App info (`internal/config`)**: `rz_config.go`, the Go counterpart of `rz_config.hpp`. Version and build date are injected via `-ldflags`.

#### Workflow

1. **Initialization**: Settings are loaded from `CheckCert.env` (next to the binary, otherwise the current directory, or via `-c`). Command line options override them.
2. **Reading**: The website list is parsed (`host`, `host:port` or a URL; `#` comments and blank lines are ignored).
3. **Checking**: All websites are checked in parallel. The expiry date is read even for already expired or untrusted certificates.
4. **Notification**: For every certificate expiring within `WARN_DAYS` days, a line is written to `<OUTPUT_DIR>/<MAIL_RECIPIENT>.SendMail`:
   ```
   The web certificate of the website hase-zheng.net expires in 4 days. See also "openssl s_client -connect hase-zheng.net:443 </dev/null 2>/dev/null | openssl x509 -noout -enddate"
   ```
   An existing (not yet sent) file is extended. The file is written atomically (temp file + rename), so SendMail never reads a half-written file.

## SendMail

### Architecture

- **Model**: `SmtpConfigModel` loads `SendMail.env` (same keys as the C++ version); `MailMessage` extracts the recipient from the file name and builds the MIME message (UTF-8, quoted-printable).
- **View**: Log output on stderr.
- **Controller**: `MailController` scans `WATCH_DIR` every `POLL_SECONDS` for `*.SendMail` files.
- **Service**: `SmtpService` connects via STARTTLS (or SSL/TLS on port 465), authenticates with `AUTH LOGIN` (fallback `AUTH PLAIN`) and sends the mail.

#### Workflow

1. **Initialization**: Settings are loaded from `SendMail.env` (next to the binary, otherwise the current directory, or via `-c`).
2. **Monitoring**: `WATCH_DIR` is scanned immediately and then every `POLL_SECONDS` (default 2 s). Polling works on every file system, including network shares and removable media.
3. **Stabilization**: A file is only processed once it has not been modified for `STABLE_SECONDS` (default 2 s). Hidden files (`.*`) are ignored.
4. **Processing**: The recipient is taken from the file name (`user@example.com.SendMail` -> `user@example.com`), the file content becomes the mail body, the subject is `SMTP_SUBJECT`.
5. **Cleanup**: After a successful send the file is deleted. If the file was changed while sending, it is kept (no data loss).
6. **Errors**: The file is kept and retried after `RETRY_SECONDS` (default 300 s) - or immediately when the file changes. This avoids hammering the server, e.g. on wrong passwords. Files with an invalid recipient are kept and logged.

### Configuration

Copy `docs/SendMail.env` next to the binary:

```ini
SMTP_SERVER="smtp.example.com"
SMTP_PORT="587"
SMTP_USERNAME="user@example.com"
SMTP_PASSWORD="secret_password"
SMTP_FROM="sender@example.com"
SMTP_STARTTLS="true"
WATCH_DIR="out"
# optional: SMTP_SSL, SMTP_SUBJECT, SMTP_TIMEOUT_SECONDS, POLL_SECONDS, STABLE_SECONDS, RETRY_SECONDS
```

Credentials are never sent over an unencrypted connection (except to localhost).

### Usage

```
SendMail [-c config] [-w watchdir] [-once] [-v] [watchdir]
```

| Option  | Description                                                    |
| ------- | -------------------------------------------------------------- |
| `-c`    | Configuration file                                             |
| `-w`    | Directory to watch (overrides `WATCH_DIR`; also first argument) |
| `-once` | Process existing files once and exit (exit code 1 if any left)  |
| `-v`    | Show version                                                   |

```bash
./build/SendMail &
echo "Hello World" > out/recipient@domain.com.SendMail
```

Run as a service: see `docs/sendmail.service` (systemd user unit). Stop with SIGINT/SIGTERM.

## Prerequisites

- **Linux**
- **Go** 1.24 or higher (build only)

## Build Instructions

```bash
git clone https://github.com/Zheng-Bote/check-cert.git
cd check-cert
make            # -> build/CheckCert, build/SendMail
make test       # unit tests
```

## CheckCert Configuration

Copy `docs/CheckCert.env` next to the binary (or into the working directory). Relative paths are resolved relative to the `.env` file.

```ini
WEBSITES_FILE="websites.txt"
OUTPUT_DIR="../send_mail/data"     # = WATCH_DIR of SendMail
MAIL_RECIPIENT="zheng.bote+webcertificate@googlemail.com"
WARN_DAYS="5"
TIMEOUT_SECONDS="10"
WORKERS="8"
```

Variables already set in the environment take precedence over the file.

## CheckCert Usage

```
CheckCert [-c config] [-l websites] [-o outdir] [-d days] [-t timeout] [-r recipient] [-v]
```

| Option | Description                            | Overrides         |
| ------ | -------------------------------------- | ----------------- |
| `-c`   | Configuration file                     |                   |
| `-l`   | Website list                           | `WEBSITES_FILE`   |
| `-o`   | Output directory for the SendMail file | `OUTPUT_DIR`      |
| `-d`   | Warning threshold in days              | `WARN_DAYS`       |
| `-t`   | Timeout per connection in seconds      | `TIMEOUT_SECONDS` |
| `-r`   | Recipient (file name)                  | `MAIL_RECIPIENT`  |
| `-v`   | Show version                           |                   |

Example `websites.txt`:

```
hase-zheng.net
bote.hase-zheng.net:443
https://flag-me.info/
```

**Exit codes:** `0` all OK, `1` at least one certificate expires soon, `2` usage/configuration error, `3` at least one website could not be checked.

### Cron

```
0 6 * * * /opt/CheckCert/CheckCert -c /opt/CheckCert/CheckCert.env
```

## License

This project is licensed under the MIT License.
