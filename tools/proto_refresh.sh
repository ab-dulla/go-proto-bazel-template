#!/usr/bin/env bash
set -euo pipefail

# Find and run all proto_link targets to copy generated sources into workspace
# Usage: ./tools/proto_refresh.sh

echo "Querying for proto_link targets..." >&2
TARGETS=$(bazel query 'kind("proto_link", //proto/...)')
if [ -z "$TARGETS" ]; then
  echo "No proto_link targets found." >&2
  exit 0
fi

echo "$TARGETS" | while read -r tgt; do
  echo "Running $tgt" >&2
  bazel run "$tgt"
done
echo "Proto refresh complete." >&2
