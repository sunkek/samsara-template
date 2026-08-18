# features.awk — render a file for a chosen feature set.
#
# Files in this template carry feature markers inside their own comment syntax,
# so an unrendered file is still a valid, runnable file with every feature on:
#
#   // feat:if rabbitmq          keep the block only when `rabbitmq` is selected
#   ...                          (Go, YAML, sh, Make, Dockerfile, Markdown —
#   // feat:else                  the marker is found anywhere on the line, so
#   //~ noteEvents := ...         the surrounding comment leader is irrelevant)
#   // feat:end
#
# Lines inside a `feat:else` block are commented out in the source (that is why
# the file works as-is) and are UNCOMMENTED when the else branch is taken: the
# renderer strips the first `~`-suffixed comment token and one following space,
# e.g. `//~ x := 1` -> `x := 1` and `#~ image: foo` -> `image: foo`.
#
# Blocks nest. The expression is a comma-separated AND of feature names, each
# optionally negated with `!`:  `feat:if postgresql,!redis`.
#
# Usage: awk -v feats="postgresql,redis" -f features.awk <file>

BEGIN {
    n = split(feats, f, /[, ]+/)
    for (i = 1; i <= n; i++) if (f[i] != "") on[f[i]] = 1
    depth = 0            # nesting depth
    emit[0] = 1          # emitting at top level
}

# Evaluate a feature expression: `|`-separated OR of `,`-separated ANDs, each
# term optionally negated with `!` — e.g. `postgresql|redis|rabbitmq`, `pg,!redis`.
function truth(expr,   ors, k, nor, res) {
    nor = split(expr, ors, "|")
    for (k = 1; k <= nor; k++) if (truthAnd(ors[k])) return 1
    return 0
}

function truthAnd(expr,   m, parts, j, t, neg, ok) {
    ok = 1
    m = split(expr, parts, ",")
    for (j = 1; j <= m; j++) {
        t = parts[j]
        gsub(/[ \t]/, "", t)
        if (t == "") continue
        neg = 0
        if (substr(t, 1, 1) == "!") { neg = 1; t = substr(t, 2) }
        if (!(t in on) != neg) { ok = 0 }
    }
    return ok
}

# Markers are recognized only at the START of a line (optionally behind a
# comment leader), so prose and scripts can mention them mid-sentence without
# being parsed as markers.
/^[ \t]*(\/\/|#|--|<!--|;)?[ \t]*feat:if[ \t]+/ {
    expr = $0
    sub(/^.*feat:if[ \t]+/, "", expr)
    sub(/[ \t]*(\*\/|-->)?[ \t]*$/, "", expr)
    depth++
    cond[depth] = truth(expr)
    seenelse[depth] = 0
    emit[depth] = emit[depth - 1] && cond[depth]
    next
}

/^[ \t]*(\/\/|#|--|<!--|;)?[ \t]*feat:else[ \t]*$/ {
    if (depth == 0) { print "features.awk: feat:else outside a block" > "/dev/stderr"; exit 2 }
    seenelse[depth] = 1
    emit[depth] = emit[depth - 1] && !cond[depth]
    next
}

/^[ \t]*(\/\/|#|--|<!--|;)?[ \t]*feat:end[ \t]*(\*\/|-->)?[ \t]*$/ {
    if (depth == 0) { print "features.awk: feat:end without feat:if" > "/dev/stderr"; exit 2 }
    depth--
    next
}

# Emitted lines carrying a `~` comment leader (`//~`, `#~`, `<!--~`, `--~`) are
# uncommented: that leader is how a block that is INACTIVE in the all-features
# source (an else branch, or a `feat:if !x` block) stays inert until selected.
{
    if (!emit[depth]) next
    line = $0
    ind = line; sub(/[^ \t].*$/, "", ind)              # leading indentation
    rest = substr(line, length(ind) + 1)
    if (rest ~ /^(\/\/~|#~|<!--~|--~|;~)([ \t]|$)/) {
        html = (rest ~ /^<!--~/)
        sub(/^(\/\/~|#~|<!--~|--~|;~)[ \t]?/, "", rest)
        # An HTML/Markdown leader has a closing half; drop it with the opener.
        if (html) sub(/[ \t]*-->[ \t]*$/, "", rest)
        $0 = ind rest
    }
    print
}

END {
    if (depth != 0) { print "features.awk: unclosed feat:if block" > "/dev/stderr"; exit 2 }
}
