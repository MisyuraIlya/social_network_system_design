#!/usr/bin/env bash
# run_dump_in_all.sh
# Execute each service's scripts/dump_files_to_readme.sh to (re)generate README.md
# Usage:
#   ./run_dump_in_all.sh            # run in current "services" dir
#   ./run_dump_in_all.sh /path/to/services

set -euo pipefail

SERVICES_DIR="${1:-.}"
SERVICES_DIR="$(cd "$SERVICES_DIR" && pwd)"

echo "Running dump in all services under: $SERVICES_DIR"
shopt -s nullglob

overall_rc=0

for svc in "$SERVICES_DIR"/*/; do
  [[ -d "$svc" ]] || continue
  svc_name="$(basename "$svc")"
  script_path="$svc/scripts/dump_files_to_readme.sh"

  if [[ -f "$script_path" ]]; then
    echo "→ [$svc_name] executing: scripts/dump_files_to_readme.sh"
    # run from the service directory so relative paths work the same everywhere
    if ( cd "$svc" && bash "scripts/dump_files_to_readme.sh" . README.md ); then
      echo "✔ [$svc_name] README.md updated"
    else
      echo "✖ [$svc_name] failed (continuing)"
      overall_rc=1
    fi
  else
    echo "⏭  [$svc_name] no scripts/dump_files_to_readme.sh — skipping"
  fi
done

exit "$overall_rc"
