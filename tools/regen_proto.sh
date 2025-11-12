#!/usr/bin/env bash
set -euo pipefail

# Find and run all proto_link targets to copy generated sources into workspace
# Usage: ./tools/proto_refresh.sh

echo "Querying for go_proto_library targets..." >&2
# We look for the underlying go_proto_library targets produced by the go_api_library macro.
# These have suffix _go_grpc per proto.bzl definition.
TARGETS=$(bazel query 'kind("go_proto_library", //proto/...)')
if [ -z "$TARGETS" ]; then
  echo "No go_proto_library targets found under //proto." >&2
  exit 0
fi

echo "$TARGETS" | while read -r tgt; do
  echo "Building $tgt" >&2
  bazel build "$tgt"
done

echo "Copying generated .pb.go files into workspace (under proto/ tree)" >&2
# Simple sync: find generated files and rsync into source tree next to .proto for IDE usage.
# Warning: These files are generated; do not commit unless desired. Consider adding them to .gitignore.
# Pre-clean any accidentally duplicated proto/proto directory produced by earlier runs.
if [ -d proto/proto ]; then
  echo "Cleaning stale duplicated directory proto/proto" >&2
  rm -rf proto/proto
fi
while read -r tgt; do
  # Derive output path dir from bazel aquery? Simpler: glob bazel-bin for this target name.
  outdir="bazel-bin/$(echo "$tgt" | sed 's#^//##' | tr ':' '/')_" # bazel adds underscore suffix
  if [ -d "$outdir" ]; then
    find "$outdir" -name '*.pb.go' -print0 | while IFS= read -r -d '' f; do
      rel="${f#bazel-bin/}"
      # Strip leading target path portion after go-monorepo-template/ to align with proto source tree layout
      if echo "$f" | grep -q 'go-monorepo-template'; then
        # Extract path tail beginning at first 'proto/' segment
        subpath=$(echo "$f" | sed -E 's#.*go-monorepo-template/(proto/.*)#\1#')
        # Collapse multiple leading proto/ occurrences (proto/proto/ -> proto/)
        subpath=$(echo "$subpath" | sed -E 's#^(proto/)+#proto/#')
        dest="${subpath}"
        mkdir -p "$(dirname "$dest")"
        # Ensure destination writable (may be read-only from previous copy)
        [ -f "$dest" ] && chmod u+w "$dest" || true
        cp "$f" "$dest"
      fi
    done
  fi
done <<< "$TARGETS"

echo "Proto refresh complete." >&2
