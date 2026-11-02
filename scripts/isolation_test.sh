#!/usr/bin/env bash
# isolation_test.sh — two forks on one machine must not touch each other's
# containers.
#
#   scripts/isolation_test.sh
#
# Bootstraps two forks with every feature and checks their compose files:
# each names its own project (compose identifies containers by project and
# service label, and the default project name is the directory, "infra", in
# every fork), and no service name appears in both (service names are DNS
# aliases on the shared external dev network). Template-only: bootstrap.sh
# removes it from forks. CI runs it (see the fork-isolation job).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT
fail=0

fork() {
  "$ROOT/bootstrap.sh" -y -d "$WORK/$1" -n "$1" -a "Test" -E test@example.org \
    -m "example.org/$1/backend" >/dev/null
}

# project <fork>: the top-level `name:` of the base compose file.
project() { sed -n 's/^name:[[:space:]]*//p' "$WORK/$1/infra/docker-compose.yml"; }

# services <fork>: every service key across the base file and the overrides.
services() {
  awk '/^services:/ { in_s = 1; next }
       /^[^[:space:]#]/ { in_s = 0 }
       in_s && /^  [A-Za-z0-9_.-]+:[[:space:]]*$/ { sub(/^  /, ""); sub(/:.*/, ""); print }' \
    "$WORK/$1"/infra/docker-compose*.yml | sort -u
}

fork alpha
fork beta

for f in alpha beta; do
  p="$(project "$f")"
  case "$p" in
    "$f"-*) ;;
    *) echo "FAIL $f: compose project is '${p:-<unset, so the directory name>}', want it to start with '$f-'"; fail=1 ;;
  esac
  if [ -z "$(services "$f")" ]; then
    echo "FAIL $f: found no services; the parser no longer matches the compose files"; fail=1
  fi
  stray="$(services "$f" | grep -v "^$f-" || true)"
  if [ -n "$stray" ]; then
    echo "FAIL $f: services without the '$f-' prefix: $(echo "$stray" | tr '\n' ' ')"; fail=1
  fi
done

shared="$(comm -12 <(services alpha) <(services beta))"
if [ -n "$shared" ]; then
  echo "FAIL: service names in both forks: $(echo "$shared" | tr '\n' ' ')"; fail=1
fi

[ "$fail" -eq 0 ] && echo "ok: forks alpha and beta share no compose project or service name"
exit "$fail"
