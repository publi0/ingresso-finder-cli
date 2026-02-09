#!/usr/bin/env bash
set -euo pipefail

FORMULA_FILE="${FORMULA_FILE:-Formula/ingresso-finder-cli.rb}"
TARGET_COMMIT="${1:-}"
REPO_SLUG="${2:-${GITHUB_REPOSITORY:-publi0/ingresso-finder-cli}}"

if [[ -z "$TARGET_COMMIT" ]]; then
  TARGET_COMMIT="$(git rev-parse HEAD)"
fi

if [[ ! "$TARGET_COMMIT" =~ ^[0-9a-f]{40}$ ]]; then
  echo "Invalid commit hash: $TARGET_COMMIT" >&2
  exit 1
fi

if [[ ! -f "$FORMULA_FILE" ]]; then
  echo "Formula file not found: $FORMULA_FILE" >&2
  exit 1
fi

TARBALL_URL="https://github.com/${REPO_SLUG}/archive/${TARGET_COMMIT}.tar.gz"
if command -v shasum >/dev/null 2>&1; then
  NEW_SHA256="$(curl -fsSL "$TARBALL_URL" | shasum -a 256 | awk '{print $1}')"
elif command -v sha256sum >/dev/null 2>&1; then
  NEW_SHA256="$(curl -fsSL "$TARBALL_URL" | sha256sum | awk '{print $1}')"
else
  echo "Missing checksum tool: expected shasum or sha256sum" >&2
  exit 1
fi

# Update only the pinned source hash and checksum lines.
perl -i -pe "s/^\\s*COMMIT = \"[0-9a-f]{40}\"(?:\\.freeze)?/  COMMIT = \"${TARGET_COMMIT}\".freeze/" "$FORMULA_FILE"
perl -i -pe "s/^\\s*sha256 \"[0-9a-f]{64}\"/  sha256 \"${NEW_SHA256}\"/" "$FORMULA_FILE"

echo "Updated $FORMULA_FILE"
echo "COMMIT=$TARGET_COMMIT"
echo "sha256=$NEW_SHA256"
