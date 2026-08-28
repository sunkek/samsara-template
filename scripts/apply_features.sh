#!/usr/bin/env bash
# apply_features.sh — cut this template down to a chosen feature set, in place.
#
#   scripts/apply_features.sh -f backend,frontend,postgresql,redis,rabbitmq [-C dir]
#
# Two passes:
#   1. PRUNE   — delete whole files/dirs that only exist for a deselected
#                feature (adapters, domains, compose fragments, migrations).
#   2. RENDER  — run scripts/features.awk over every surviving text file, so
#                marker blocks (`feat:if` / `feat:else`) collapse to the chosen
#                variant. See features.awk for the marker syntax.
#
# The unrendered template is the all-features build: with every feature
# selected this script is a no-op beyond stripping the marker comments.
#
# Features: backend frontend postgresql redis rabbitmq
set -euo pipefail

FEATS="" ; TARGET="."
while getopts "f:C:h" opt; do
  case "$opt" in
    f) FEATS="$OPTARG" ;;
    C) TARGET="$OPTARG" ;;
    h) grep '^#' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *) echo "see -h" >&2; exit 1 ;;
  esac
done

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
AWK_FILE="$SCRIPT_DIR/features.awk"
cd "$TARGET"

has() { case ",$FEATS," in *",$1,"*) return 0 ;; *) return 1 ;; esac; }

# --- Dependency rules ---------------------------------------------------------
# The sample domains are Postgres-backed; without it there is nothing to serve,
# so the backend degrades to a bare supervisor + fiber skeleton. Redis/RabbitMQ
# are optional everywhere: the note domain falls back to its Noop ports and auth
# swaps its Redis revoker for the in-memory one.
if ! has backend && ! has frontend; then
  echo "note: neither backend nor frontend selected — infra-only scaffold." >&2
fi

# --- Pass 1: prune ------------------------------------------------------------
# Each line: "<feature> <path> [path...]" — paths are removed when the feature
# is NOT selected. Keep this table as the single place that knows which files
# belong to which feature.
prune() {
  local feat="$1"; shift
  has "$feat" && return 0
  local p
  for p in "$@"; do
    [ -e "$p" ] || continue
    rm -rf "$p"
    echo "  removed $p"
  done
}

echo "Pruning for: $FEATS"

prune frontend \
  services/frontend

prune backend \
  services/backend \
  services/frontend/config \
  env/example/api.env \
  docs

prune postgresql \
  services/backend/internal/domain/note \
  services/backend/internal/domain/notestats \
  services/backend/internal/domain/auth \
  services/backend/internal/integration \
  infra/postgresql \
  env/example/postgresql.env \
  scripts/postgresql_dump.sh \
  scripts/postgresql_restore.sh

prune rabbitmq \
  services/backend/internal/domain/note/adapter/rabbitmq \
  services/backend/internal/domain/notestats \
  infra/rabbitmq \
  infra/postgresql/migration/000003_create_note_stats.up.sql \
  infra/postgresql/migration/000003_create_note_stats.down.sql \
  env/example/rabbitmq.env

prune redis \
  services/backend/internal/domain/note/adapter/redis \
  services/backend/internal/domain/auth/adapter/redis \
  infra/redis \
  env/example/redis.env

# With neither app service selected there is nothing for CI to build, and an
# empty `jobs:`/pipeline is not a valid workflow — drop the CI definitions.
if ! has backend && ! has frontend; then
  for p in .github/workflows/ci.yml .github/workflows/codeql.yml .gitlab-ci.yml; do
    [ -e "$p" ] || continue
    rm -f "$p"
    echo "  removed $p"
  done
fi

# The in-memory revoker exists only as the Redis adapter's stand-in.
if has redis; then
  rm -rf services/backend/internal/domain/auth/adapter/memory 2>/dev/null || true
fi

# --- Pass 2: render -----------------------------------------------------------
echo "Rendering feature markers..."
rendered=0
while IFS= read -r -d '' file; do
  grep -qI . "$file" 2>/dev/null || continue          # skip binaries
  grep -qE '^[[:space:]]*(//|#|--|<!--|;)?[[:space:]]*feat:if[[:space:]]' "$file" || continue
  tmp="$file.feat.tmp"
  if ! awk -v feats="$FEATS" -f "$AWK_FILE" "$file" > "$tmp"; then
    echo "  FAILED to render $file" >&2
    rm -f "$tmp"
    exit 1
  fi
  # Preserve the executable bit; awk output starts fresh.
  [ -x "$file" ] && chmod +x "$tmp"
  mv "$tmp" "$file"
  rendered=$((rendered + 1))
done < <(find . -type f \
  -not -path './.git/*' \
  -not -path '*/node_modules/*' \
  -not -path '*/dist/*' \
  -not -path '*/tmp/*' \
  -not -name 'features.awk' \
  -not -name 'apply_features.sh' \
  -not -name 'features_test.sh' \
  -print0)
echo "  rendered $rendered file(s)"

# Stripping marker comments leaves Go struct-tag alignment and blank lines that
# gofmt would reflow; normalize so the fork starts gofmt-clean.
if [ -d services/backend ] && command -v gofmt >/dev/null 2>&1; then
  gofmt -w services/backend
  echo "  gofmt -w services/backend"
fi

# Drop directories emptied by the prune pass.
find . -mindepth 1 -type d -empty -not -path './.git/*' -delete 2>/dev/null || true

echo "Done."
