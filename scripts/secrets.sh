#!/usr/bin/env bash
set -euo pipefail

# secrets.sh -- SOPS + age for this repository's env files.
#
# The layout, which is the part worth copying:
#
#   env/<env>/          plaintext, git-ignored by DIRECTORY
#   env/sops/<env>/     ciphertext, committed
#
# Keeping the two apart by path is the point. The mistake this prevents is a
# plaintext file edited at a path that gets committed, and a name-mask ignore
# (*.env) does not prevent it -- a certificate or a token file slips straight
# through.
#
# The environment is part of the ciphertext path on purpose. dev, stage and
# prod carry the SAME filenames here, so a flat env/sops/ means
# `encrypt ENVIRONMENT=dev` overwrites the prod ciphertext with dev values and
# `decrypt ENVIRONMENT=dev` writes prod secrets into env/dev/. `check` would
# pass both, because it asks whether a file is encrypted and not whether it
# holds what its path claims.
#
# Usage:
#   scripts/secrets.sh init <name> <age1...>        create .sops.yaml
#   scripts/secrets.sh add-recipient <name> <age1...>
#   scripts/secrets.sh remove-recipient <name>      re-key AND rotate data keys
#   scripts/secrets.sh encrypt [env]                env/<env>/  -> env/sops/<env>/
#   scripts/secrets.sh decrypt [env]                env/sops/<env>/ -> env/<env>/
#   scripts/secrets.sh check                        fail on plaintext under env/sops/
#
# `env` defaults to $ENVIRONMENT, then to prod.
#
# FILES="smtp.env s3.env" restricts encrypt/decrypt to those files. Encrypting
# PART of an environment is a normal thing to want: a dev stack whose Postgres
# password is a container on a laptop but whose SMTP gateway is a real account
# wants exactly one of its files in git. Everything not listed stays generated
# by `make gen-env` and never enters the repository.
#
# FORCE=1 lets decrypt overwrite existing plaintext. Off by default because a
# local file may hold a value that was never encrypted, and clobbering it loses
# the only copy. It is needed for the partial case above: `gen-env` materializes
# every file in env/example/, so the shared file usually exists before the
# decrypt that is supposed to replace it.

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

SOPS="${SOPS:-sops}"
ENV_NAME="${2:-${ENVIRONMENT:-prod}}"
SOPS_CONFIG=".sops.yaml"

die() { echo "$*" >&2; exit 1; }

need_sops() {
  command -v "$SOPS" >/dev/null 2>&1 \
    || die "sops not found. Install it: https://github.com/getsops/sops/releases"
}

need_config() {
  [ -f "$SOPS_CONFIG" ] \
    || die "no $SOPS_CONFIG -- run: scripts/secrets.sh init <name> <age1...>"
}

# .env files encrypt as dotenv so a diff shows which variable changed and the
# keys stay readable in review. Anything else is opaque and goes in whole.
input_type_for() {
  case "$1" in
    *.env) echo dotenv ;;
    *)     echo binary ;;
  esac
}

cmd_init() {
  local name="${1:-}" key="${2:-}"
  [ -n "$name" ] && [ -n "$key" ] || die "usage: scripts/secrets.sh init <name> <age1...>"
  case "$key" in age1*) ;; *) die "not an age public key: $key" ;; esac
  [ -f "$SOPS_CONFIG" ] && die "$SOPS_CONFIG already exists -- use add-recipient"

  cat > "$SOPS_CONFIG" <<YAML
# Who can decrypt what. Recipients are age PUBLIC keys; the private half never
# leaves its owner's machine, or the two operators stop being two independent
# operators and become one with two copies of a key.
#
# Generate yours once:
#   mkdir -p ~/.config/sops/age
#   age-keygen -o ~/.config/sops/age/keys.txt
#   chmod 600 ~/.config/sops/age/keys.txt
#   grep '^# public key:' ~/.config/sops/age/keys.txt
#
# Adding a recipient is one line here plus a re-key of every file:
#   scripts/secrets.sh add-recipient <name> <age1...>
#
# REMOVING one is not the reverse. They hold a clone, or held one, and every
# value in it stays valid -- removal means rotating every secret they could
# read. Use the script, not a hand edit here, so the data keys rotate too:
#   scripts/secrets.sh remove-recipient <name>
# Keep this list to the people who genuinely need it.
keys:
  - &$name $key

creation_rules:
  # prod -- operators only. Listed first: sops takes the FIRST matching rule.
  - path_regex: ^env/sops/prod/secrets/.*\$
    input_type: binary
    key_groups:
      - age:
          - *$name
  - path_regex: ^env/sops/prod/.*\\.env\$
    input_type: dotenv
    key_groups:
      - age:
          - *$name

  # dev/stage -- add the whole team here when there is a non-inert secret to
  # put in it. Not before: it costs every developer an age key before they can
  # run \`make up\`, and a container password on a laptop is not worth that.
  #
  # [^/]+ rather than .* is deliberate. A file dropped straight into
  # env/sops/ past the environment level then matches no rule and fails loudly,
  # instead of being encrypted to whichever recipient list a catch-all happens
  # to carry.
  - path_regex: ^env/sops/[^/]+/secrets/.*\$
    input_type: binary
    key_groups:
      - age:
          - *$name
  - path_regex: ^env/sops/[^/]+/.*\\.env\$
    input_type: dotenv
    key_groups:
      - age:
          - *$name
YAML
  echo "wrote $SOPS_CONFIG with recipient $name"

  # A file holding no secret, encrypted to every recipient by construction. It
  # exists so a new operator can prove their key works without being handed a
  # real credential, and so "your key is broken" and "your key is not a
  # recipient of that file" can be told apart.
  mkdir -p env/prod
  if [ ! -f env/prod/roundtrip-check.env ]; then
    cat > env/prod/roundtrip-check.env <<'TXT'
# Not a secret. Encrypted to every recipient in .sops.yaml so a new operator
# can prove their age key works without decrypting anything real:
#
#   sops -d env/sops/prod/roundtrip-check.env
export ROUNDTRIP_CHECK="if-you-can-read-this-your-age-key-works"
TXT
    echo "wrote env/prod/roundtrip-check.env"
  fi

  echo "next: put the prod plaintext in env/prod/, then scripts/secrets.sh encrypt prod"
}

# Appends the key to the anchor list and to every age: block, then re-keys the
# existing files. `sops updatekeys` rewrites the recipient list and NOTHING
# else, which is why it is used here instead of a re-encrypt: a re-encrypt
# would publish whatever the working tree's plaintext currently says.
cmd_add_recipient() {
  need_sops; need_config
  local name="${1:-}" key="${2:-}"
  [ -n "$name" ] && [ -n "$key" ] || die "usage: scripts/secrets.sh add-recipient <name> <age1...>"
  case "$key" in age1*) ;; *) die "not an age public key: $key" ;; esac
  grep -q "$key" "$SOPS_CONFIG" && die "$key is already a recipient"

  awk -v name="$name" -v key="$key" '
    # Last line of the top-level anchor list: append the new anchor after it.
    /^  - &/ { last_anchor = NR }
    { lines[NR] = $0 }
    # Last "- *ref" of each age: block, recorded as we go.
    /^ *- \*/ { ref_end[NR] = 1; indent[NR] = match($0, /-/) - 1 }
    END {
      for (i = 1; i <= NR; i++) {
        print lines[i]
        if (i == last_anchor) printf "  - &%s %s\n", name, key
        if (ref_end[i] && !ref_end[i+1]) printf "%*s- *%s\n", indent[i], "", name
      }
    }
  ' "$SOPS_CONFIG" > "$SOPS_CONFIG.tmp" && mv "$SOPS_CONFIG.tmp" "$SOPS_CONFIG"
  echo "added $name to $SOPS_CONFIG"

  if [ -d env/sops ]; then
    find env/sops -type f | while read -r f; do
      "$SOPS" updatekeys -y "$f" >/dev/null
      echo "  re-keyed $f"
    done
  fi

  cat <<'TXT'

Now have the new operator prove it on their own machine, both files and in
this order:

  sops -d env/sops/prod/roundtrip-check.env          # their key works
  sops -d env/sops/prod/<a real file> >/dev/null && echo OK

The first proves the key and nothing about the real files -- roundtrip-check
is encrypted to every recipient by construction. The second proves the real
files were re-keyed after they were added. Its output goes to /dev/null:
decrypting a real secret onto a shared screen is how one gets leaked.
TXT
}

# Drops the key from .sops.yaml, then per file: `updatekeys` to take them off
# the recipient list, and `rotate` to replace the data key.
#
# The rotate is the part that is easy to miss. updatekeys re-wraps the SAME data
# key for the remaining recipients, and the removed operator can still unwrap
# that key from any old version in git history with their own age key -- so
# anything written into the file afterwards with `sops <file>` or `sops set`,
# the rotated credentials included, stays readable to them. Order matters too:
# rotate takes its recipients from the file, not from .sops.yaml, so without
# updatekeys first it re-keys the file to the removed operator all over again.
cmd_remove_recipient() {
  need_sops; need_config
  local name="${1:-}" key
  [ -n "$name" ] || die "usage: scripts/secrets.sh remove-recipient <name>"
  key="$(awk -v name="$name" '$1 == "-" && $2 == "&" name { print $3 }' "$SOPS_CONFIG")"
  [ -n "$key" ] || die "no recipient named $name in $SOPS_CONFIG"
  [ "$(grep -c '^  - &' "$SOPS_CONFIG")" -gt 1 ] \
    || die "$name is the only recipient; removing them leaves nobody able to decrypt"

  awk -v name="$name" '
    $1 == "-" && ($2 == "&" name || $2 == "*" name) { next }
    { print }
  ' "$SOPS_CONFIG" > "$SOPS_CONFIG.tmp" && mv "$SOPS_CONFIG.tmp" "$SOPS_CONFIG"
  ! grep -q "$key" "$SOPS_CONFIG" || die "$key is still in $SOPS_CONFIG; edit it by hand"
  echo "removed $name from $SOPS_CONFIG"

  if [ -d env/sops ]; then
    while IFS= read -r f; do
      local t; t="$(input_type_for "$f")"
      "$SOPS" updatekeys -y --input-type "$t" "$f" >/dev/null 2>&1
      "$SOPS" rotate -i --input-type "$t" --output-type "$t" "$f"
      grep -q "$key" "$f" && die "$f still lists $name after re-keying"
      echo "  re-keyed and rotated $f"
    done < <(find env/sops -type f | sort)
  fi

  cat <<TXT

Commit and merge this BEFORE rotating anything: a new value written into a file
whose data key $name knows is a value $name can read.

This stops $name reading future versions and nothing else. Every value they
could ever read is still valid, so rotate each one where it is issued, then
check the old one is refused. The files they could read, history included:

  git log --all --name-only --format= -S "$key" | sort -u
TXT
}

# The files to act on, relative to $1: everything under it, or just $FILES.
list_files() {
  local base="$1" f
  if [ -n "${FILES:-}" ]; then
    for f in $FILES; do
      [ -f "$base/$f" ] || die "no $base/$f"
      echo "$base/$f"
    done
  else
    find "$base" -type f ! -name '.*' | sort
  fi
}

cmd_encrypt() {
  need_sops; need_config
  local src="env/$ENV_NAME" dst="env/sops/$ENV_NAME" n=0
  [ -d "$src" ] || die "no $src/ to encrypt -- is the environment right? (got '$ENV_NAME')"

  while IFS= read -r f; do
    local rel="${f#"$src"/}" out="$dst/${f#"$src"/}"
    mkdir -p "$(dirname "$out")"
    # --filename-override makes the path rules in .sops.yaml apply to where the
    # ciphertext LANDS rather than where the plaintext sits.
    #
    # shellcheck disable=SC2094 # $out is only a NAME to sops here; the bytes
    # come from $f, so nothing reads the file being written.
    "$SOPS" -e --input-type "$(input_type_for "$f")" --output-type "$(input_type_for "$f")" \
      --filename-override "$out" "$f" > "$out"
    echo "  encrypted $rel"
    n=$((n + 1))
  done < <(list_files "$src")

  [ "$n" -gt 0 ] || die "no files under $src/"
  echo "encrypted $n file(s): $src/ -> $dst/"
  echo "verify before committing: scripts/secrets.sh check && git diff --stat env/sops"
}

# Refuses to overwrite unless FORCE=1. A local file may hold a value that was
# never encrypted, and clobbering it loses the only copy.
cmd_decrypt() {
  need_sops
  local src="env/sops/$ENV_NAME" dst="env/$ENV_NAME" n=0
  [ -d "$src" ] || die "no $src/ to decrypt -- is the environment right? (got '$ENV_NAME')"
  umask 077

  while IFS= read -r f; do
    local rel="${f#"$src"/}" out="$dst/${f#"$src"/}"
    if [ -f "$out" ] && [ "${FORCE:-0}" != "1" ]; then
      die "$out exists; not overwriting (move it aside, or pass FORCE=1)"
    fi
    mkdir -p "$(dirname "$out")"
    "$SOPS" -d --input-type "$(input_type_for "$f")" --output-type "$(input_type_for "$f")" \
      "$f" > "$out"
    chmod 600 "$out"
    echo "  decrypted $rel"
    n=$((n + 1))
  done < <(list_files "$src")

  echo "decrypted $n file(s): $src/ -> $dst/"
}

# Sweeps every environment, not the selected one: a check scoped to one passes
# while another environment's plaintext sits committed beside it.
cmd_check() {
  [ -d env/sops ] || { echo "no env/sops/ -- nothing to check"; return 0; }
  local bad=0 n=0
  while IFS= read -r f; do
    n=$((n + 1))
    grep -q 'sops_version\|"version"\|sops:' "$f" || { echo "$f is NOT sops-encrypted" >&2; bad=1; }
  done < <(find env/sops -type f)
  [ "$bad" -eq 0 ] || exit 1
  echo "all $n file(s) under env/sops/ are sops-encrypted"
}

case "${1:-}" in
  init)           shift; cmd_init "${1:-}" "${2:-}" ;;
  add-recipient)  shift; cmd_add_recipient "${1:-}" "${2:-}" ;;
  remove-recipient) shift; cmd_remove_recipient "${1:-}" ;;
  encrypt)        cmd_encrypt ;;
  decrypt)        cmd_decrypt ;;
  check)          cmd_check ;;
  *) sed -n '/^# Usage:/,/^# .env. defaults/p' "$0" | sed 's/^# \{0,1\}//' >&2; exit 1 ;;
esac
