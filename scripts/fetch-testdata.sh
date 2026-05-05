#!/usr/bin/env bash
# Fetch the gismanager testdata fixtures listed in testdata/manifest.sha256
# into testdata/fetched/, verifying each download's sha256 against the
# manifest. Idempotent: files already present with a matching hash are
# skipped, so re-running is cheap.
#
# Usage:
#   scripts/fetch-testdata.sh                # uses defaults
#   MANIFEST=path/to/manifest.sha256 scripts/fetch-testdata.sh
#   DEST=/tmp/fixtures scripts/fetch-testdata.sh
#
# Manifest format (whitespace-separated, '#' starts a comment line):
#   <sha256>  <relpath-under-DEST>  <url>
#
# Exits non-zero on hash mismatch, missing curl/sha256sum, or any download
# failure. CI wraps this in `make fetch-testdata`.

set -euo pipefail

MANIFEST="${MANIFEST:-testdata/manifest.sha256}"
DEST="${DEST:-testdata-fetched}"

if ! command -v curl >/dev/null 2>&1; then
    echo "fetch-testdata: curl not found in PATH" >&2
    exit 127
fi
if ! command -v sha256sum >/dev/null 2>&1; then
    echo "fetch-testdata: sha256sum not found in PATH" >&2
    exit 127
fi

if [[ ! -f "$MANIFEST" ]]; then
    echo "fetch-testdata: manifest $MANIFEST not found" >&2
    exit 1
fi

mkdir -p "$DEST"

ok=0
fetched=0
fail=0

while IFS= read -r line || [[ -n "$line" ]]; do
    # Skip blank + comment lines.
    [[ -z "${line// }" ]] && continue
    [[ "$line" =~ ^[[:space:]]*# ]] && continue

    # Parse: hash, relpath, url. Tolerant of any whitespace count between fields.
    read -r expected_hash relpath url <<< "$line"
    if [[ -z "$expected_hash" || -z "$relpath" || -z "$url" ]]; then
        echo "fetch-testdata: malformed manifest line: $line" >&2
        fail=$((fail + 1))
        continue
    fi

    target="$DEST/$relpath"
    if [[ -f "$target" ]]; then
        actual=$(sha256sum "$target" | awk '{print $1}')
        if [[ "$actual" == "$expected_hash" ]]; then
            echo "OK    $relpath"
            ok=$((ok + 1))
            continue
        fi
        echo "STALE $relpath (expected $expected_hash, got $actual) — refetching" >&2
        rm -f "$target"
    fi

    mkdir -p "$(dirname "$target")"
    echo "FETCH $relpath <- $url"
    if ! curl -fsSL --retry 3 --retry-delay 2 --max-time 120 -o "$target" "$url"; then
        echo "fetch-testdata: download failed for $relpath" >&2
        fail=$((fail + 1))
        continue
    fi
    actual=$(sha256sum "$target" | awk '{print $1}')
    if [[ "$actual" != "$expected_hash" ]]; then
        echo "fetch-testdata: hash mismatch for $relpath (expected $expected_hash, got $actual)" >&2
        rm -f "$target"
        fail=$((fail + 1))
        continue
    fi
    fetched=$((fetched + 1))
done < "$MANIFEST"

echo "fetch-testdata: ok=$ok fetched=$fetched failed=$fail"
if [[ $fail -gt 0 ]]; then
    exit 1
fi
