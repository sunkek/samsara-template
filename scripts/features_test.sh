#!/usr/bin/env bash
# features_test.sh — unit tests for scripts/features.awk.
#
#   scripts/features_test.sh
#
# Each case feeds a small marked-up document through the renderer for a given
# feature set and compares the output to what the template promises. Run it
# after touching features.awk; CI runs it too (see the feature-matrix workflow).
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
AWK_FILE="$SCRIPT_DIR/features.awk"
pass=0 fail=0

# render <feats> <<< input
render() { awk -v feats="$1" -f "$AWK_FILE"; }

# check <name> <feats> <input> <expected-output>
check() {
  local name="$1" feats="$2" input="$3" want="$4" got rc
  got="$(printf '%s' "$input" | render "$feats")"; rc=$?
  if [ "$rc" -ne 0 ]; then
    echo "FAIL $name: renderer exited $rc"; fail=$((fail + 1)); return
  fi
  if [ "$got" != "$want" ]; then
    echo "FAIL $name"
    echo "  want: $(printf '%s' "$want" | tr '\n' '|')"
    echo "  got:  $(printf '%s' "$got" | tr '\n' '|')"
    fail=$((fail + 1)); return
  fi
  pass=$((pass + 1))
}

# check_err <name> <feats> <input> — renderer must reject the document.
check_err() {
  local name="$1" feats="$2" input="$3"
  if printf '%s' "$input" | render "$feats" >/dev/null 2>&1; then
    echo "FAIL $name: expected a non-zero exit"; fail=$((fail + 1))
  else
    pass=$((pass + 1))
  fi
}

# --- selection ----------------------------------------------------------------
check "if: feature on" "redis" \
'a
# feat:if redis
b
# feat:end
c' \
'a
b
c'

check "if: feature off" "postgresql" \
'a
# feat:if redis
b
# feat:end
c' \
'a
c'

check "else: takes the uncommented variant" "postgresql" \
'# feat:if redis
x := redis()
# feat:else
#~ x := noop()
# feat:end' \
'x := noop()'

check "else: skipped when the feature is on" "redis" \
'# feat:if redis
x := redis()
# feat:else
#~ x := noop()
# feat:end' \
'x := redis()'

check "negation" "postgresql" \
'# feat:if !redis
#~ no-redis
# feat:end' \
'no-redis'

check "AND: all terms must hold" "postgresql" \
'# feat:if postgresql,redis
both
# feat:end
# feat:if postgresql,!redis
pg-only
# feat:end' \
'pg-only'

check "OR: any term suffices" "rabbitmq" \
'# feat:if postgresql|rabbitmq|redis
any
# feat:end' \
'any'

check "OR: none of the terms hold" "backend" \
'# feat:if postgresql|rabbitmq|redis
any
# feat:end' \
''

# --- nesting ------------------------------------------------------------------
check "nested: both levels on" "postgresql,redis" \
'# feat:if postgresql
outer
# feat:if redis
inner
# feat:end
# feat:end' \
'outer
inner'

check "nested: inner off" "postgresql" \
'# feat:if postgresql
outer
# feat:if redis
inner
# feat:end
tail
# feat:end' \
'outer
tail'

check "nested: outer off hides the inner block entirely" "redis" \
'# feat:if postgresql
outer
# feat:if redis
inner
# feat:end
# feat:end
after' \
'after'

# --- uncomment leaders --------------------------------------------------------
check "leader: // and indentation preserved" "backend" \
'# feat:if !redis
    //~ code := 1
# feat:end' \
'    code := 1'

check "leader: yaml #~" "backend" \
'# feat:if !frontend
#~ services: {}
# feat:end' \
'services: {}'

check "leader: html comment" "backend" \
'<!-- feat:if !frontend -->
<!--~ no frontend -->
<!-- feat:end -->' \
'no frontend'

check "no leader: line is emitted verbatim" "redis" \
'# feat:if redis
plain ~ tilde stays
# feat:end' \
'plain ~ tilde stays'

# --- marker recognition -------------------------------------------------------
check "markers are only recognized at the start of a line" "backend" \
'grep -rn "feat:if" .
echo done' \
'grep -rn "feat:if" .
echo done'

check "unmarked document passes through untouched" "backend" \
'one
two' \
'one
two'

# --- malformed input ----------------------------------------------------------
check_err "unclosed block is an error" "backend" \
'# feat:if backend
x'

check_err "feat:end without feat:if is an error" "backend" \
'x
# feat:end'

check_err "feat:else outside a block is an error" "backend" \
'x
# feat:else'

echo "features.awk: $pass passed, $fail failed"
[ "$fail" -eq 0 ]
