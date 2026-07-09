#!/usr/bin/env bash
#
# Build helper for the POS API.
# Keeps Go's build cache in a writable temp directory inside this workspace.
#
# Usage:
#   ./scripts/build.sh
#   ./scripts/build.sh ./cmd/api   # optional custom package path
#

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
PACKAGE_PATH="${1:-./cmd/api}"
OUTPUT_NAME="${2:-build/api}"
GOCACHE_DIR="${GOCACHE_DIR:-/private/tmp/gocache}"

mkdir -p "$GOCACHE_DIR"
mkdir -p "$(dirname "$OUTPUT_NAME")"

echo "Building $PACKAGE_PATH -> $OUTPUT_NAME"
(cd "$PROJECT_ROOT" && GOCACHE="$GOCACHE_DIR" go build -o "$OUTPUT_NAME" "$PACKAGE_PATH")

echo "Build complete: $OUTPUT_NAME"
