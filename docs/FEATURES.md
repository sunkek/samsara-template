# Feature markers

Read this before editing a file that carries `feat:if` comments, or when adding
a feature to the template.

Files carry `feat:if <expr>` / `feat:else` / `feat:end` comments in their own
comment syntax. `scripts/apply_features.sh` renders them for a chosen feature
set (`backend`, `frontend`, `postgresql`, `redis`, `rabbitmq`) in two passes:
prune whole files listed in its table, then run `scripts/features.awk` over
every surviving text file. `scripts/features.awk` is the marker syntax's source
of truth — expressions are `|`-separated ORs of `,`-separated ANDs, each term
optionally negated with `!`.

## Rules

- The checked-in template is the **all-features** build and must run as-is.
  Code that exists only when a feature is *off* lives behind a `~` comment
  leader (`//~`, `#~`, `<!--~`), which the renderer uncomments.
- The `~` leader is **per line**, and an HTML one closes on the same line:
  `<!--~ text -->`. A multi-line HTML comment renders as literal comment text.
- Markers are recognized only at the **start of a line**; mid-sentence they are
  ordinary prose and stay in the output verbatim. Gate a whole line or block,
  never a phrase inside one.
- Whole files and directories belonging to one feature go in the prune table in
  `scripts/apply_features.sh` instead of being marked up.
- Keep markers **out of** a backslash-continued Make recipe or shell command:
  the shell joins the lines, so `# feat:if …` comments out the rest of the
  command in the unrendered template. Gate the variable that feeds the recipe
  instead (see `LOCAL_INFRA` / `RUN_LOCAL_*` in the `Makefile`).
- Inside a positively-gated block, write plain uncommented content. The `~`
  leader marks blocks *inactive* in the all-features build; elsewhere it hides
  the content from the template's own readers.
- Adding a feature: add a combination to the `feature-matrix` CI job, and
  extend `scripts/features_test.sh` when you change marker semantics.

## Checking your work

Render every combination your change touches and confirm no marker or `~`
leader survives:

```bash
awk -v feats="backend,postgresql" -f scripts/features.awk <file> | grep -n 'feat:\|~ '
scripts/features_test.sh
```
