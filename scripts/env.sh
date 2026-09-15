#!/usr/bin/env bash
set -euo pipefail

# env.sh -- add and propagate environment variables.
#
# env/example/*.env is the source of truth for WHICH variables exist; the
# per-environment env/<env>/*.env hold what they are set to. Adding a variable
# by hand means editing the template plus every environment plus, once secrets
# are encrypted, a re-encrypt -- four steps that are easy to do three of. These
# two commands do all of them.
#
#   scripts/env.sh add <file> <NAME> [value] [options]
#       Append a variable to env/example/<file> and to every environment that
#       already has that file.
#
#       --secret            value is a credential: the template gets the
#                           "password" placeholder that gen-env understands, and
#                           each environment gets its OWN freshly minted value.
#                           A given value, if passed, is ignored.
#       --envs "a b"        environments to touch (default: every existing
#                           env/<env>/ that already has <file>)
#       --comment "text"    comment line written above the variable
#       --no-encrypt        skip the re-encrypt of environments that have
#                           ciphertext under env/sops/
#
#   scripts/env.sh sync [env...]
#       Add variables that exist in env/example/ but are missing from the named
#       environments, leaving every existing value alone. This is the one to
#       run after pulling someone else's new variable, and after
#       `secrets.sh decrypt` of a file older than the template.
#
# Examples:
#   scripts/env.sh add api.env MY_PROJECT_API_FEATURE_X_ENABLED false
#   scripts/env.sh add api.env MY_PROJECT_API_STRIPE_KEY --secret --envs prod
#   scripts/env.sh sync

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

EXAMPLE_DIR="env/example"

die() { echo "$*" >&2; exit 1; }
gen_secret() { openssl rand -hex 32; }

# Environments = every env/<dir> except example/ and sops/.
all_envs() {
  find env -maxdepth 1 -mindepth 1 -type d \
    ! -name example ! -name sops -printf '%f\n' 2>/dev/null | sort
}

# Matches `NAME=`, `export NAME=` and leading whitespace, so a variable is never
# added twice under a different spelling of the same line.
has_var() {
  grep -qE "^[[:space:]]*(export[[:space:]]+)?$2=" "$1" 2>/dev/null
}

# Mirrors the style already in the file: this repo's env files use `export`.
write_var() {
  local dst="$1" name="$2" value="$3" comment="$4"
  local prefix=""
  grep -qE '^[[:space:]]*export[[:space:]]' "$dst" 2>/dev/null && prefix="export " || true
  # One blank line before the addition, but never two.
  if [ -s "$dst" ]; then
    [ -n "$(tail -c1 "$dst")" ] && printf '\n' >> "$dst" || true
    [ -n "$(tail -n1 "$dst")" ] && printf '\n' >> "$dst" || true
  fi
  [ -n "$comment" ] && printf '# %s\n' "$comment" >> "$dst" || true
  printf '%s%s="%s"\n' "$prefix" "$name" "$value" >> "$dst"
}

# Environments whose ciphertext is now stale, collected during a run and
# re-encrypted once at the end rather than per file.
declare -a TOUCHED_ENVS=()

reencrypt_touched() {
  [ "${#TOUCHED_ENVS[@]}" -gt 0 ] || return 0
  for e in $(printf '%s\n' "${TOUCHED_ENVS[@]}" | sort -u); do
    [ -d "env/sops/$e" ] || continue
    echo "re-encrypting env/sops/$e/"
    ENVIRONMENT="$e" scripts/secrets.sh encrypt "$e"
  done
}

cmd_add() {
  local file="${1:-}" name="${2:-}"; shift 2 || true
  [ -n "$file" ] && [ -n "$name" ] || die "usage: scripts/env.sh add <file> <NAME> [value] [--secret] [--envs \"a b\"] [--comment text]"

  local value="" secret=0 envs="" comment="" encrypt=1
  # A bare first argument is the value; everything else is a flag.
  case "${1:-}" in -*|"") ;; *) value="$1"; shift ;; esac
  while [ $# -gt 0 ]; do
    case "$1" in
      --secret)     secret=1; shift ;;
      --envs)       envs="${2:-}"; shift 2 ;;
      --comment)    comment="${2:-}"; shift 2 ;;
      --no-encrypt) encrypt=0; shift ;;
      *) die "unknown option: $1" ;;
    esac
  done

  case "$name" in
    *[!A-Za-z0-9_]*|"") die "not a valid variable name: $name" ;;
  esac

  local tmpl="$EXAMPLE_DIR/$file"
  [ -f "$tmpl" ] || die "no $tmpl -- create it first, or check the filename"

  # The template carries the placeholder, not the secret. `password` is what
  # gen-env recognises and replaces on a fresh materialization.
  if has_var "$tmpl" "$name"; then
    echo "$tmpl already defines $name -- leaving it alone"
  else
    write_var "$tmpl" "$name" "$([ "$secret" -eq 1 ] && echo password || echo "$value")" "$comment"
    echo "added $name to $tmpl"
  fi

  [ -n "$envs" ] || envs="$(all_envs)"
  for e in $envs; do
    local dst="env/$e/$file"
    [ -f "$dst" ] || { echo "skipping env/$e (no $file)"; continue; }
    if has_var "$dst" "$name"; then
      echo "env/$e/$file already defines $name -- leaving it alone"
      continue
    fi
    # Each environment mints its own: a prod credential shared with dev is not
    # a prod credential.
    write_var "$dst" "$name" "$([ "$secret" -eq 1 ] && gen_secret || echo "$value")" "$comment"
    echo "added $name to $dst"
    TOUCHED_ENVS+=("$e")
  done

  [ "$encrypt" -eq 1 ] && reencrypt_touched
  return 0
}

cmd_sync() {
  local envs="$*"
  [ -n "$envs" ] || envs="$(all_envs)"

  for e in $envs; do
    [ -d "env/$e" ] || { echo "no env/$e -- skipping"; continue; }
    for tmpl in "$EXAMPLE_DIR"/*.env; do
      [ -e "$tmpl" ] || die "no .env files in $EXAMPLE_DIR"
      local base dst; base="$(basename "$tmpl")"; dst="env/$e/$base"
      if [ ! -f "$dst" ]; then
        echo "env/$e/$base is missing entirely -- run: make gen-env GEN_ENVS=$e APP=<name>"
        continue
      fi
      # Read the template rather than the environment, so a variable the
      # environment holds and the template does not is kept, not deleted.
      while IFS= read -r line; do
        case "$line" in ''|'#'*) continue ;; esac
        local name value
        name="${line#export }"; name="${name%%=*}"
        case "$name" in *[!A-Za-z0-9_]*|"") continue ;; esac
        has_var "$dst" "$name" && continue
        value="${line#*=}"; value="${value%\"}"; value="${value#\"}"
        case "$value" in
          password) value="$(gen_secret)" ;;
          app)      value="${APP:-my_project}" ;;
        esac
        write_var "$dst" "$name" "$value" ""
        echo "added $name to $dst"
        TOUCHED_ENVS+=("$e")
      done < "$tmpl"
    done
  done

  [ "${#TOUCHED_ENVS[@]}" -gt 0 ] || { echo "every environment already has every variable in $EXAMPLE_DIR"; return 0; }
  reencrypt_touched
}

case "${1:-}" in
  add)  shift; cmd_add "$@" ;;
  sync) shift; cmd_sync "$@" ;;
  *) sed -n '/^# env.sh --/,/^$/p;/^#   scripts/,/^#   scripts.env.sh sync$/p' "$0" | sed 's/^# \{0,1\}//' >&2; exit 1 ;;
esac
