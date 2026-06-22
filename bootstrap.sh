#!/usr/bin/env bash
# Fork this reference service into a new, renamed project.
#
# This repository is a runnable reference service. Its committed identity is:
#   app name     my_project
#   app title    My Project
#   env prefix   MY_PROJECT          (env vars: MY_PROJECT_API_*)
#   module path  github.com/sunkek/samsara-template/backend
#   author       Sunkek
#   author email security@example.com
#
# bootstrap.sh rewrites those concrete values to the names you choose, turning
# the reference into your own project. It does NOT add or remove features.
#
# Usage:
#   ./bootstrap.sh -d ../myapp -n myapp -m github.com/me/myapp/backend   # fork into a new dir (recommended)
#   ./bootstrap.sh                                                       # interactive, convert in place
#
# Flags:
#   -f   force: proceed even if the destination dir already exists and is non-empty
#   -V   verify: after rename, run `go build ./...` in the fork to confirm it compiles
#
# With -d the reference is copied to the destination and renamed there; the
# original is left untouched (so you can fork it again). Without -d the current
# checkout is renamed in place. The destination must be outside this directory.
set -euo pipefail

# --- Source identity: what this repo is named right now. -----------------------
# (Kept verbatim so the renamer knows what to search for. bootstrap.sh excludes
# itself from the rewrite, so these stay valid for repeated `-d` forks.)
OLD_APP_NAME="my_project"
OLD_APP_TITLE="My Project"
OLD_ENV_PREFIX="MY_PROJECT"
OLD_MODULE_PATH="github.com/sunkek/samsara-template/backend"
OLD_AUTHOR="Sunkek"
OLD_AUTHOR_EMAIL="security@example.com"

# --- Target identity: filled from flags / prompts. -----------------------------
APP_NAME="" APP_TITLE="" ENV_PREFIX="" MODULE_PATH="" AUTHOR="" AUTHOR_EMAIL="" DEST=""
FORCE="" VERIFY=""

while getopts "n:t:e:m:a:E:d:fVh" opt; do
  case "$opt" in
    n) APP_NAME="$OPTARG" ;;
    t) APP_TITLE="$OPTARG" ;;
    e) ENV_PREFIX="$OPTARG" ;;
    m) MODULE_PATH="$OPTARG" ;;
    a) AUTHOR="$OPTARG" ;;
    E) AUTHOR_EMAIL="$OPTARG" ;;
    d) DEST="$OPTARG" ;;
    f) FORCE=1 ;;
    V) VERIFY=1 ;;
    h) grep '^#' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *) echo "see -h"; exit 1 ;;
  esac
done

# Prompt for any value not supplied on the command line. $3 is the default.
ask() {
  local var="$1" prompt="$2" default="$3" val
  val="$(eval "printf '%s' \"\${$var}\"")"
  [ -n "$val" ] && return 0
  read -r -p "$prompt [$default]: " val
  eval "$var=\"\${val:-\$default}\""
}

ask APP_NAME     "New app name (lowercase slug)" "myapp"
ask APP_TITLE    "New app title (human readable)" "$(printf '%s' "$APP_NAME" | sed 's/.*/\u&/')"
ask ENV_PREFIX   "New env var prefix (UPPER_SNAKE)" "$(printf '%s' "$APP_NAME" | tr '[:lower:]-' '[:upper:]_')"
ask MODULE_PATH  "New Go module path" "example.com/${APP_NAME}/backend"
ask AUTHOR       "Author / copyright holder" "Your Name"
ask AUTHOR_EMAIL "Security / contact email" "you@example.com"
ask DEST         "Destination dir ('.' = rename in place)" "."

echo
echo "  $OLD_APP_NAME    -> $APP_NAME"
echo "  $OLD_APP_TITLE   -> $APP_TITLE"
echo "  $OLD_ENV_PREFIX  -> $ENV_PREFIX"
echo "  $OLD_MODULE_PATH -> $MODULE_PATH"
echo "  $OLD_AUTHOR      -> $AUTHOR"
echo "  $OLD_AUTHOR_EMAIL -> $AUTHOR_EMAIL"
echo "  DEST = $DEST"
echo
read -r -p "Apply rename? [y/N] " ok
[[ "$ok" =~ ^[Yy]$ ]] || { echo "Aborted."; exit 0; }

ROOT="$(cd "$(dirname "$0")" && pwd)"
SELF="$(basename "$0")"

# Resolve the target the rename is applied to. In place by default; with a
# destination, copy the reference there (minus VCS, deps, build output, and
# generated secrets) and rename there, leaving the original untouched.
if [ -z "$DEST" ] || [ "$DEST" = "." ]; then
  TARGET="$ROOT"
else
  dest_parent="$(cd "$(dirname "$DEST")" 2>/dev/null && pwd)" \
    || { echo "Destination parent does not exist: $(dirname "$DEST")"; exit 1; }
  TARGET="$dest_parent/$(basename "$DEST")"
  # The reference and the destination must not overlap in either direction.
  case "$TARGET/" in
    "$ROOT/"*) echo "Destination must be outside the reference directory ($ROOT)."; exit 1 ;;
  esac
  case "$ROOT/" in
    "$TARGET/"*) echo "Destination must not contain the reference directory ($ROOT)."; exit 1 ;;
  esac
  # Refuse to scribble into a populated destination unless -f is given. cp -R
  # merges into an existing tree, which can silently mix the reference with
  # unrelated files already there.
  if [ -e "$TARGET" ] && [ -n "$(ls -A "$TARGET" 2>/dev/null)" ]; then
    if [ -z "$FORCE" ]; then
      echo "Destination $TARGET already exists and is not empty. Existing entries:" >&2
      ls -A "$TARGET" | sed 's/^/  /' >&2
      echo "Re-run with -f to merge the reference into it anyway." >&2
      exit 1
    fi
    echo "Destination $TARGET non-empty; merging anyway (-f)."
  fi
  mkdir -p "$TARGET"
  echo "Copying reference → $TARGET"
  cp -R "$ROOT"/. "$TARGET"/
  # Strip VCS, deps, build output, generated env files (secrets!), and local
  # editor/agent state so none of it leaks into the fork. env/example is the
  # only env dir that ships; everything else is generated by `make gen-env`.
  rm -rf "$TARGET/.git" \
         "$TARGET/node_modules" "$TARGET/service/frontend/node_modules" \
         "$TARGET/service/frontend/dist" "$TARGET/service/backend/tmp" \
         "$TARGET/service/backend/main" \
         "$TARGET/.claude/settings.local.json"
  find "$TARGET/env" -mindepth 1 -maxdepth 1 -type d -not -name example -exec rm -rf {} + 2>/dev/null || true
  # Drop fork-only cruft: bootstrap.sh is a template-maintenance tool with no
  # purpose in a fork, and dated session-message dumps are local agent state.
  rm -f "$TARGET/$SELF"
  find "$TARGET" -type f -name '*command-message*.txt' -delete 2>/dev/null || true
fi

# Escape a string for the search (BRE) side of `s|…|…|`.
esc_search() { printf '%s' "$1" | sed -e 's/[][\.*^$/&|]/\\&/g'; }
# Escape a string for the replacement side of `s|…|…|`.
esc_repl() { printf '%s' "$1" | sed -e 's/[&|\\]/\\&/g'; }

# Project-identity substitutions, applied to EVERY file. Order matters: replace
# the longest / most specific identities first so a later, broader match cannot
# corrupt them (module path before prefix before name).
sed_args=(
  -e "s|$(esc_search "$OLD_MODULE_PATH")|$(esc_repl "$MODULE_PATH")|g"
  -e "s|$(esc_search "$OLD_ENV_PREFIX")|$(esc_repl "$ENV_PREFIX")|g"
  -e "s|$(esc_search "$OLD_APP_TITLE")|$(esc_repl "$APP_TITLE")|g"
  -e "s|$(esc_search "$OLD_APP_NAME")|$(esc_repl "$APP_NAME")|g"
)

# Author / email substitutions, applied ONLY to documentation files. Author and
# email are copyright/contact metadata — they live in LICENSE, SECURITY.md, and
# other docs, never in code. Running them repo-wide is unsafe: the author slug
# can be a substring of the NEW module path (e.g. gitlab.com/Sunkek/<app>), so a
# blanket pass rewrites "Sunkek" inside the freshly-written module path and
# produces an invalid go.mod ("gitlab.com/Nikita Zotov/<app>/backend"). Scoping
# to docs sidesteps the collision entirely.
sed_author=(
  -e "s|$(esc_search "$OLD_AUTHOR_EMAIL")|$(esc_repl "$AUTHOR_EMAIL")|g"
  -e "s|$(esc_search "$OLD_AUTHOR")|$(esc_repl "$AUTHOR")|g"
)

# Rename in every file except VCS, deps, build output, and this script. Run from
# TARGET with relative paths so exclusions anchor to the project, not to absolute
# path components. bootstrap.sh excludes itself so its OLD_* identity stays
# valid (and so it never rewrites itself mid-run).
cd "$TARGET"
find . -type f \
  -not -path './.git/*' \
  -not -path '*/node_modules/*' \
  -not -path '*/dist/*' \
  -not -path '*/tmp/*' \
  -not -name "$SELF" \
  -exec sed -i "${sed_args[@]}" {} +

# Author/email pass: documentation files only (see comment above).
find . -type f \
  -not -path './.git/*' \
  -not -path '*/node_modules/*' \
  -not -path '*/dist/*' \
  -not -path '*/tmp/*' \
  \( -name '*.md' -o -name 'LICENSE' -o -name 'SECURITY.md' \) \
  -exec sed -i "${sed_author[@]}" {} +

# Guard: a valid Go module path has no whitespace. If the rename produced one
# (e.g. an author slug leaked into the module path), fail loud rather than ship
# a broken fork.
if grep -qE '^module[[:space:]]+[^[:space:]]+[[:space:]]+[^[:space:]]' service/backend/go.mod; then
  echo "ERROR: service/backend/go.mod module path contains whitespace after rename:" >&2
  grep -n '^module' service/backend/go.mod >&2
  exit 1
fi

# Optional: confirm the renamed backend still compiles. Requires Go on PATH.
if [ -n "$VERIFY" ]; then
  echo "Verifying build (go build ./...) ..."
  if command -v go >/dev/null 2>&1; then
    ( cd "$TARGET/service/backend" && go mod tidy && go build ./... ) \
      && echo "Build OK." \
      || { echo "ERROR: fork does not build — see output above." >&2; exit 1; }
  else
    echo "WARNING: go not found on PATH; skipping build verification." >&2
  fi
fi

# Nudge if the security/contact email is still a placeholder — it lands in
# SECURITY.md as a real contact address.
case "$AUTHOR_EMAIL" in
  ""|you@example.com|security@example.com)
    echo "WARNING: author/contact email is a placeholder ($AUTHOR_EMAIL). Set a real address before publishing (SECURITY.md)." >&2 ;;
esac

echo "Done. Next:"
[ "$TARGET" = "$ROOT" ] || echo "  cd $TARGET"
echo "  cd service/backend && go mod tidy"
echo "  cd service/frontend && npm install"
echo "  docker network create dev   # if not already"
echo "  make gen-env APP=$APP_NAME   # fills env/dev + env/local"
