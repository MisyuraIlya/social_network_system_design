#!/usr/bin/env bash
# dump_files_to_readme.sh
# Rebuild a README-like file by concatenating every text file in the tree,
# prefixed by its relative path, then the file's full contents.
# Usage:
#   ./dump_files_to_readme.sh [ROOT_DIR=. ] [OUTPUT_FILE=README.md]

set -euo pipefail

ROOT_DIR="${1:-.}"
OUTPUT_FILE="${2:-README.md}"

ROOT_DIR="$(cd "$ROOT_DIR" && pwd)"
SCRIPT_ABS_PATH="$(cd "$(dirname "$0")" && pwd)/$(basename "$0")"

timestamp() { date +"%Y-%m-%d %H:%M:%S%z"; }

# Coarse excludes anywhere
GEN_EXCLUDES=(
  "*/.git/*" "*/.idea/*" "*/.vscode/*" "*/node_modules/*" "*/vendor/*"
  "*/dist/*" "*/build/*" "*/.cache/*" "*/coverage/*"
)

# Decide if a RELATIVE path (from ROOT_DIR) should be skipped
should_skip_rel() {
  local rel="$1"

  case "$rel" in
    # skip output file itself
    "$(basename "$OUTPUT_FILE")" ) return 0 ;;
    # skip the script itself if under ROOT_DIR
    "$(realpath --relative-to="$ROOT_DIR" "$SCRIPT_ABS_PATH" 2>/dev/null)" ) return 0 ;;
  esac

  # project-specific skips (work whether ROOT_DIR is repo root or services/user-service)
  case "$rel" in
    # exact files
    ".air.toml" | "Dockerfile" | "go.sum" | "go.mod" ) return 0 ;;
    "services/user-service/.air.toml" ) return 0 ;;
    "services/user-service/Dockerfile" ) return 0 ;;
    "services/user-service/go.sum" ) return 0 ;;
    "services/user-service/go.mod" ) return 0 ;;
    # tmp dirs
    tmp/* ) return 0 ;;
    services/user-service/tmp/* ) return 0 ;;
  esac

  return 1
}

# Header (overwrite without making a backup)
{
  echo "# Project code dump"
  echo
  echo "- Generated: $(timestamp)"
  echo "- Root: \`$ROOT_DIR\`"
  echo
} > "$OUTPUT_FILE"

# Build find args
build_find_args() {
  local args=()
  args+=( "$ROOT_DIR" -type f )
  for pat in "${GEN_EXCLUDES[@]}"; do
    args+=( ! -path "$pat" )
  done
  printf '%q ' "${args[@]}"
}

# Find, sort, and append
# shellcheck disable=SC2046
eval find $(build_find_args) -print0 | sort -z | while IFS= read -r -d '' f; do
  # Skip likely-binary files
  if ! grep -Iq . "$f"; then
    continue
  fi

  # extra guard: skip the script itself
  if [[ "$f" == "$SCRIPT_ABS_PATH" ]]; then
    continue
  fi

  rel="${f#"$ROOT_DIR"/}"
  if should_skip_rel "$rel"; then
    continue
  fi

  {
    echo "$rel"
    cat "$f"
    echo
  } >> "$OUTPUT_FILE"
done

echo "Done. Wrote ${OUTPUT_FILE}"
