#!/usr/bin/env bash
# Fork this reference service into a new, renamed project.
#
# This repository is a runnable reference service. Its committed identity is:
#   app name     my_project
#   app title    My Project
#   env prefix   MY_PROJECT          (env vars: MY_PROJECT_API_*)
#   module path  github.com/sunkek/samsara-template/backend
#   author       Nikita Zotov
#   author email security@example.com
#
# bootstrap.sh rewrites those concrete values to the names you choose, turning
# the reference into your own project. It also cuts the fork down to the
# features you pick (-F), via scripts/apply_features.sh. `./bootstrap.sh -h`
# prints the flags; docs/BOOTSTRAP.md has copy-paste commands per feature set.
#
# With -d the reference is copied to the destination and renamed there; the
# original is left untouched (so you can fork it again). Without -d the current
# checkout is renamed in place. The destination must be outside this directory.
set -euo pipefail

# --- Source identity: what this repo is named right now. -----------------------
# (Kept verbatim so the renamer knows what to search for. bootstrap.sh is never
# copied into a fork and excludes itself from the rewrite, so these stay valid
# for repeated `-d` forks.)
OLD_APP_NAME="my_project"
OLD_APP_TITLE="My Project"
OLD_ENV_PREFIX="MY_PROJECT"
OLD_MODULE_PATH="github.com/sunkek/samsara-template/backend"
OLD_AUTHOR="Nikita Zotov"
OLD_AUTHOR_EMAIL="security@example.com"

# --- Target identity: filled from flags / prompts. -----------------------------
APP_NAME="" APP_TITLE="" ENV_PREFIX="" MODULE_PATH="" AUTHOR="" AUTHOR_EMAIL="" DEST=""
FORCE="" VERIFY="" FEATURES="" ASSUME_YES="" FEATURES_SET=""
ALL_FEATURES="backend,frontend,postgresql,redis,rabbitmq"

usage() {
  cat <<EOF
Usage: ./bootstrap.sh [-y] -d <dest> -n <name> [options]

Fork this template into a renamed project, keeping only the chosen features.
Without -y, any value not given as a flag is prompted for.

  -n <name>    app name, lowercase slug (e.g. myapp)                 required
  -t <title>   human-readable title          default: name, capitalised
  -e <PREFIX>  env var prefix, UPPER_SNAKE   default: name, upper-cased
  -m <module>  Go module path (e.g. github.com/me/myapp/backend)
               required when the backend feature is kept; ignored otherwise
  -a <author>  author / copyright holder (LICENSE, docs)             required
  -E <email>   security / contact email (SECURITY.md)                required
  -d <dir>     destination; '.' renames this checkout in place       required
               An existing git repo is fine: its .git is never touched.
  -F <list>    comma-separated features to keep  default: all
               features: ${ALL_FEATURES//,/ }   (-F '' = infra only)
  -f           proceed when the destination exists and is not empty
               (template files overwrite same-named files there)
  -V           after the rename, run go mod tidy + go build in the fork
  -y           non-interactive: never prompt. Fails listing every
               missing required value instead of asking for it.
  -h           print this help

Example (non-interactive, frontend only):
  ./bootstrap.sh -y -d ../myapp -n myapp -t "My App" \\
    -a "Jane Doe" -E security@example.org -F frontend

See docs/BOOTSTRAP.md for the common feature sets and the steps after.
EOF
}

while getopts "n:t:e:m:a:E:d:F:fVyh" opt; do
  case "$opt" in
    n) APP_NAME="$OPTARG" ;;
    t) APP_TITLE="$OPTARG" ;;
    e) ENV_PREFIX="$OPTARG" ;;
    m) MODULE_PATH="$OPTARG" ;;
    a) AUTHOR="$OPTARG" ;;
    E) AUTHOR_EMAIL="$OPTARG" ;;
    d) DEST="$OPTARG" ;;
    F) FEATURES="$OPTARG"; FEATURES_SET=1 ;;
    f) FORCE=1 ;;
    V) VERIFY=1 ;;
    y) ASSUME_YES=1 ;;
    h) usage; exit 0 ;;
    *) usage >&2; exit 1 ;;
  esac
done
shift $((OPTIND - 1))
[ "$#" -eq 0 ] || { echo "Unexpected argument: $1 (see -h)" >&2; exit 1; }

has_feature() { case ",$FEATURES," in *",$1,"*) return 0 ;; *) return 1 ;; esac; }

# Prompt for any value not supplied on the command line. $3 is the default.
# Under -y nothing is prompted; the caller has already checked the required
# values, so an unset one simply takes its default.
ask() {
  local var="$1" prompt="$2" default="$3" val
  val="$(eval "printf '%s' \"\${$var}\"")"
  [ -n "$val" ] && return 0
  if [ -n "$ASSUME_YES" ]; then
    val=""
  else
    read -r -p "$prompt [$default]: " val
  fi
  eval "$var=\"\${val:-\$default}\""
}

# Features first: whether the Go module path is needed at all depends on them.
# Under -y an absent -F means all features (the same default the prompt offers).
if [ -z "$FEATURES_SET" ]; then
  ask FEATURES "Features to keep ($ALL_FEATURES)" "$ALL_FEATURES"
fi

# Reject typos now rather than silently shipping a fork missing a feature.
for feat in ${FEATURES//,/ }; do
  case ",$ALL_FEATURES," in
    *",$feat,"*) ;;
    *) echo "Unknown feature: $feat (choose from $ALL_FEATURES)" >&2; exit 1 ;;
  esac
done

if [ -n "$ASSUME_YES" ]; then
  missing=()
  [ -n "$APP_NAME" ]     || missing+=("-n <app name>")
  [ -n "$DEST" ]         || missing+=("-d <destination dir, or . for in place>")
  [ -n "$AUTHOR" ]       || missing+=("-a <author>")
  [ -n "$AUTHOR_EMAIL" ] || missing+=("-E <contact email>")
  if has_feature backend && [ -z "$MODULE_PATH" ]; then
    missing+=("-m <Go module path> (the backend feature is selected)")
  fi
  if [ "${#missing[@]}" -gt 0 ]; then
    echo "bootstrap.sh -y: missing required values:" >&2
    printf '  %s\n' "${missing[@]}" >&2
    echo "See ./bootstrap.sh -h." >&2
    exit 1
  fi
fi

ask APP_NAME     "New app name (lowercase slug)" "myapp"
ask APP_TITLE    "New app title (human readable)" "$(printf '%s' "$APP_NAME" | sed 's/.*/\u&/')"
ask ENV_PREFIX   "New env var prefix (UPPER_SNAKE)" "$(printf '%s' "$APP_NAME" | tr '[:lower:]-' '[:upper:]_')"
if has_feature backend; then
  ask MODULE_PATH "New Go module path" "example.com/${APP_NAME}/backend"
else
  # No Go code survives, so nothing reads the module path; fill it only so the
  # rename below has a well-formed value to substitute.
  MODULE_PATH="${MODULE_PATH:-example.com/${APP_NAME}/backend}"
fi
ask AUTHOR       "Author / copyright holder" "Your Name"
ask AUTHOR_EMAIL "Security / contact email" "you@example.com"
ask DEST         "Destination dir ('.' = rename this checkout in place)" "../${APP_NAME}"

echo
echo "  $OLD_APP_NAME    -> $APP_NAME"
echo "  $OLD_APP_TITLE   -> $APP_TITLE"
echo "  $OLD_ENV_PREFIX  -> $ENV_PREFIX"
has_feature backend && echo "  $OLD_MODULE_PATH -> $MODULE_PATH"
echo "  $OLD_AUTHOR      -> $AUTHOR"
echo "  $OLD_AUTHOR_EMAIL -> $AUTHOR_EMAIL"
echo "  features = $FEATURES"
echo "  DEST = $DEST"
echo
if [ -z "$ASSUME_YES" ]; then
  read -r -p "Apply rename? [y/N] " ok
  [[ "$ok" =~ ^[Yy]$ ]] || { echo "Aborted."; exit 0; }
fi

ROOT="$(cd "$(dirname "$0")" && pwd)"
SELF="$(basename "$0")"

# Resolve the target the rename is applied to. In place by default; with a
# destination, copy the reference there and rename there, leaving the original
# untouched.
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
  # Refuse to scribble into a populated destination unless -f is given: the copy
  # merges into an existing tree and overwrites same-named files, which can
  # silently mix the reference with unrelated files already there. A lone .git
  # (a freshly `git init`-ed repo) does not count as populated.
  if [ -e "$TARGET" ] \
    && [ -n "$(find "$TARGET" -mindepth 1 -maxdepth 1 ! -name .git -print -quit)" ]; then
    if [ -z "$FORCE" ]; then
      echo "Destination $TARGET already exists and is not empty. Existing entries:" >&2
      ls -A "$TARGET" | sed 's/^/  /' >&2
      echo "Re-run with -f to merge the reference into it anyway (same-named files are overwritten;" >&2
      echo "commit first so git shows what changed)." >&2
      exit 1
    fi
    echo "Destination $TARGET non-empty; merging anyway (-f)."
  fi
  mkdir -p "$TARGET"
  echo "Copying reference → $TARGET"
  # What never leaves the template, excluded at copy time so nothing has to be
  # deleted from the destination afterwards — the destination may be an existing
  # repository, and its own .git, env files and node_modules are not ours to
  # remove:
  #   - VCS: the template's history is not the fork's;
  #   - deps and build output;
  #   - generated env files (secrets!) and env/sops/: env/example is the only
  #     env dir that ships. A fork starts with its own recipients and secrets;
  #     inheriting another project's ciphertext would leave .sops.yaml claiming
  #     keys that no longer match the files;
  #   - local editor/agent state and dated session-message dumps;
  #   - bootstrap.sh itself, a template-maintenance tool with no use in a fork.
  excludes=(
    --exclude=./.git
    --exclude=node_modules
    --exclude=./services/frontend/dist
    --exclude=./services/backend/tmp
    --exclude=./services/backend/main
    --exclude=./.claude/settings.local.json
    --exclude='*command-message*.txt'
    "--exclude=./$SELF"
  )
  for d in "$ROOT"/env/*/; do
    [ -d "$d" ] || continue
    name="$(basename "$d")"
    [ "$name" = example ] || excludes+=("--exclude=./env/$name")
  done
  tar -C "$ROOT" "${excludes[@]}" -cf - . | tar -C "$TARGET" -xf -
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
# blanket pass rewrites the author name inside the freshly-written module path
# and produces an invalid go.mod ("gitlab.com/Nikita Zotov/<app>/backend" — note
# the space). Scoping to docs sidesteps the collision entirely.
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

# Strip the template-only banner from README so the fork's README opens directly
# on the (renamed) project, not on the "this is a scaffold" preamble.
if [ -f README.md ]; then
  sed -i '/<!-- TEMPLATE-BANNER:START -->/,/<!-- TEMPLATE-BANNER:END -->/d' README.md
fi

# Guard: a valid Go module path has no whitespace. If the rename produced one
# (e.g. an author slug leaked into the module path), fail loud rather than ship
# a broken fork.
if [ -f services/backend/go.mod ] \
  && grep -qE '^module[[:space:]]+[^[:space:]]+[[:space:]]+[^[:space:]]' services/backend/go.mod; then
  echo "ERROR: services/backend/go.mod module path contains whitespace after rename:" >&2
  grep -n '^module' services/backend/go.mod >&2
  exit 1
fi

# The template gitignores services/backend/go.sum (its absence gates CI while
# the repo is still the unbootstrapped template). The fork must commit go.sum,
# so drop that ignore block here.
sed -i '/^# The template ships without go\.sum/,/^services\/backend\/go\.sum$/d' "$TARGET/.gitignore"

# Cut the fork down to the chosen features, then drop the feature tooling: it is
# a template-maintenance tool, and after this pass there are no markers left for
# it to act on.
if [ "$FEATURES" != "$ALL_FEATURES" ] || [ -f scripts/apply_features.sh ]; then
  scripts/apply_features.sh -f "$FEATURES" -C .
fi
rm -f scripts/apply_features.sh scripts/features.awk scripts/features_test.sh \
      docs/FEATURES.md docs/BOOTSTRAP.md
# Template-maintenance material that would dangle in a fork: the template's own
# changelog, the ADR about the feature renderer, and the issue template that
# tells a reporter to run a script this fork no longer has. A fork starts its
# own changelog; `git log` holds the history either way.
rm -f CHANGELOG.md docs/adr/0001-features-rendered-at-bootstrap.md
if [ -f .github/ISSUE_TEMPLATE/bug_report.yml ]; then
  # Drop the bootstrap step from the reproduction template.
  sed -i '/bootstrap\.sh/d' .github/ISSUE_TEMPLATE/bug_report.yml
fi
# In place, this script is the last template tool left (a -d fork never gets a
# copy). Deleting a running bash script is safe: bash keeps the file open.
if [ "$TARGET" = "$ROOT" ]; then
  rm -f "$TARGET/$SELF"
fi

# Optional: confirm the renamed backend still compiles. Requires Go on PATH.
if [ -n "$VERIFY" ] && [ -d "$TARGET/services/backend" ]; then
  echo "Verifying build (go build ./...) ..."
  if command -v go >/dev/null 2>&1; then
    ( cd "$TARGET/services/backend" && go mod tidy && go build ./... ) \
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
if [ -d "$TARGET/services/backend" ]; then
  echo "  (cd services/backend && go mod tidy)    # commit go.sum: CI is gated on it"
fi
if [ -d "$TARGET/services/frontend" ]; then
  echo "  (cd services/frontend && npm install)"
fi
echo "  docker network create dev   # if not already"
echo "  make gen-env APP=$APP_NAME   # fills env/dev + env/local"
if [ -f "$TARGET/POST-FORK.md" ]; then
  echo "  review the docs listed in POST-FORK.md, then delete it"
fi
