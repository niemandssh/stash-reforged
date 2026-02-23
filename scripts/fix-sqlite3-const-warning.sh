#!/usr/bin/env bash
# Fix -Wdiscarded-qualifiers in go-sqlite3's sqlite3-binding.c:
#   zTail = strrchr(zName, '_');  ->  zTail = (const char *)strrchr(zName, '_');
# strrchr() returns char*; assigning to const char* zTail triggers the warning.
# This script finds sqlite3-binding.c in GOMODCACHE and applies the fix in-place.

set -e
GOMODCACHE="${GOMODCACHE:-$(go env GOMODCACHE)}"
if [[ -z "$GOMODCACHE" ]]; then
  echo "Cannot determine GOMODCACHE" >&2
  exit 1
fi
CANDIDATE=$(find "$GOMODCACHE" -name "sqlite3-binding.c" -path "*mattn*go-sqlite3*" 2>/dev/null | head -1)
if [[ -z "$CANDIDATE" ]]; then
  echo "sqlite3-binding.c not found in GOMODCACHE (run: go mod download)" >&2
  exit 1
fi
if grep -q 'zTail = (const char \*)strrchr(zName,' "$CANDIDATE"; then
  echo "Already patched: $CANDIDATE"
  exit 0
fi
if ! grep -q "zTail = strrchr(zName, '_');" "$CANDIDATE"; then
  echo "Target line not found in $CANDIDATE (SQLite version may differ)" >&2
  exit 1
fi
# Module cache files are read-only (0444); make writable so we can patch
chmod u+w "$CANDIDATE"
tmp=$(mktemp)
sed "s/  zTail = strrchr(zName, '_');/  zTail = (const char *)strrchr(zName, '_');/" "$CANDIDATE" > "$tmp"
mv "$tmp" "$CANDIDATE"
echo "Patched: $CANDIDATE"
