#!/usr/bin/env bash
#
# check_cert.sh - Checks the validity of web certificates based on a list of websites.
#
# If a certificate expires within <days>, the file
#   zheng.bote+webcertificate@googlemail.com.SendMail
# is created or extended in the output directory (one message per website).
#
# Format of the website list (one website per line):
#   hase-zheng.net
#   example.com:8443
#   https://www.example.org/path      (scheme and path are ignored)
#   # comments and blank lines are skipped
#
set -u

MAIL_FILE_NAME="zheng.bote+webcertificate@googlemail.com.SendMail"
DAYS=5
TIMEOUT=10
LIST=""
OUTDIR=""

usage() {
    cat <<EOF
Usage: $(basename "$0") -l <website-list> -o <output-dir> [-d <days>] [-t <timeout>]

  -l  File with websites (one per line, optionally with :port)
  -o  Directory in which the file ${MAIL_FILE_NAME} is created
  -d  Warning threshold in days (default: ${DAYS})
  -t  Connection timeout in seconds (default: ${TIMEOUT})
  -h  Show this help

Exit codes: 0 = all ok, 1 = at least one certificate expires soon,
            2 = usage error, 3 = at least one website could not be checked
EOF
}

while getopts ":l:o:d:t:h" opt; do
    case "$opt" in
        l) LIST="$OPTARG" ;;
        o) OUTDIR="$OPTARG" ;;
        d) DAYS="$OPTARG" ;;
        t) TIMEOUT="$OPTARG" ;;
        h) usage; exit 0 ;;
        :) echo "Error: option -$OPTARG requires a value." >&2; usage >&2; exit 2 ;;
        *) echo "Error: unknown option -$OPTARG" >&2; usage >&2; exit 2 ;;
    esac
done

[[ -z "$LIST" || -z "$OUTDIR" ]] && { usage >&2; exit 2; }
[[ -r "$LIST" ]] || { echo "Error: list '$LIST' not readable." >&2; exit 2; }
[[ "$DAYS" =~ ^[0-9]+$ ]] || { echo "Error: -d must be a number." >&2; exit 2; }
[[ "$TIMEOUT" =~ ^[0-9]+$ ]] || { echo "Error: -t must be a number." >&2; exit 2; }
command -v openssl >/dev/null || { echo "Error: openssl not found." >&2; exit 2; }
mkdir -p "$OUTDIR" || { echo "Error: cannot create directory '$OUTDIR'." >&2; exit 2; }

MAIL_FILE="${OUTDIR%/}/${MAIL_FILE_NAME}"
NOW=$(date +%s)
rc=0

while IFS= read -r line || [[ -n "$line" ]]; do
    # Strip comments, whitespace, scheme and path
    line="${line%%#*}"
    line="$(echo "$line" | tr -d '[:space:]')"
    [[ -z "$line" ]] && continue
    line="${line#*://}"
    line="${line%%/*}"

    host="${line%%:*}"
    port="443"
    [[ "$line" == *:* ]] && port="${line##*:}"
    target="${host}:${port}"

    enddate=$(timeout "$TIMEOUT" openssl s_client -connect "$target" -servername "$host" \
                  </dev/null 2>/dev/null | openssl x509 -noout -enddate 2>/dev/null)
    enddate="${enddate#notAfter=}"

    if [[ -z "$enddate" ]]; then
        echo "ERROR    ${target}: could not retrieve certificate." >&2
        [[ $rc -eq 0 ]] && rc=3
        continue
    fi

    end_ts=$(date -d "$enddate" +%s 2>/dev/null) || {
        echo "ERROR    ${target}: cannot parse expiry date '$enddate'." >&2
        [[ $rc -eq 0 ]] && rc=3
        continue
    }

    days_left=$(( (end_ts - NOW) / 86400 ))
    (( end_ts < NOW )) && days_left=$(( -((NOW - end_ts + 86399) / 86400) ))

    if (( days_left <= DAYS )); then
        if (( days_left < 0 )); then
            text="The web certificate of the website ${host} expired $(( -days_left )) days ago."
        else
            text="The web certificate of the website ${host} expires in ${days_left} days."
        fi
        printf '%s See also "openssl s_client -connect %s </dev/null 2>/dev/null | openssl x509 -noout -enddate"\n' \
            "$text" "$target" >> "$MAIL_FILE"
        echo "WARNING  ${target}: ${days_left} days (${enddate})"
        rc=1
    else
        echo "OK       ${target}: ${days_left} days (${enddate})"
    fi
done < "$LIST"

exit $rc
