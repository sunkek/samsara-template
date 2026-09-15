# Secrets

Environment files here come in two halves, and the split is the whole design:

| Path | What | Git |
|---|---|---|
| `env/example/*.env` | which variables exist, with placeholder values | committed |
| `env/<env>/*.env` | what they are set to in that environment | **ignored, by directory** |
| `env/sops/<env>/*` | the same values, encrypted with SOPS + age | committed |

`env/sops/` is the only path in this repository where a secret may sit in git,
and `make secrets-check` fails if anything under it is not ciphertext.

Nothing below is mandatory. A fork that never deploys anything can ignore SOPS
entirely and keep using `make gen-env` — the encrypted half stays empty and
costs nothing.

## Adding a variable

```bash
make env-add FILE=api.env NAME=MY_PROJECT_API_FEATURE_X VALUE=false
make env-add FILE=api.env NAME=MY_PROJECT_API_STRIPE_KEY SECRET=1 ENVS=prod
make env-sync                      # after pulling someone else's new variable
```

`env-add` writes the variable to `env/example/` *and* to every environment that
has that file, then re-encrypts whichever of them are committed as ciphertext.
Doing it by hand is four steps and it is easy to do three.

`SECRET=1` puts the `password` placeholder in the template — which is what
`gen-env` recognises on a fresh materialization — and mints a **different**
value per environment. A prod credential shared with dev is not a prod
credential.

`env-sync` adds variables the template has and an environment lacks, leaving
every existing value alone. `gen-env` is not the tool for that: it materializes
a whole environment with fresh secrets and will not fill a gap in an existing
one.

## Does this repository want SOPS?

> A repository wants SOPS when a fresh clone plus one age key has to be able to
> deploy it, and the values it needs are credentials that open production.

A committed `*.dev.env` holding the password of a Postgres container on a laptop
does not meet that test. Encrypting it demands an age key from every developer
before they can run `make up`, and protects nothing. The right action for an
inert file is a comment on its first line saying it is inert, so the next reader
does not have to work it out again.

The structure exists from the start; fill it when there is something real to put
in it.

## Setup

Each operator generates their own key. Only the **public** half is collected —
if a private key is passed around, two recipients stop being two independent
operators and become one with two copies of a key.

```bash
mkdir -p ~/.config/sops/age
age-keygen -o ~/.config/sops/age/keys.txt
chmod 600 ~/.config/sops/age/keys.txt
grep '^# public key:' ~/.config/sops/age/keys.txt
```

Then, once per repository:

```bash
make secrets-init NAME=<you> KEY=age1...   # writes .sops.yaml + a roundtrip check
# put the real prod values in env/prod/
make secrets-encrypt                        # env/prod/ -> env/sops/prod/
make secrets-check
git add .sops.yaml env/sops && git commit
```

On another machine, or a fresh clone:

```bash
make secrets-decrypt                        # refuses to overwrite existing plaintext
make env-sync                               # in case the template moved on
```

## Adding an operator

```bash
make secrets-add-recipient NAME=<them> KEY=age1...
```

This adds the key to `.sops.yaml` and runs `sops updatekeys` over every
encrypted file. `updatekeys` rewrites the recipient list and nothing else —
a re-encrypt would also publish whatever the working tree's plaintext currently
says, which is not what granting access is supposed to do. Editing a file with
`sops <file>` does **not** pick up a new recipient at all: it reuses the list
stored inside the file, which is how a newly added operator silently keeps
failing to decrypt.

Then have them prove it on their own machine, both files, in this order:

```bash
sops -d env/sops/prod/roundtrip-check.env            # their key works
sops -d env/sops/prod/api.env >/dev/null && echo OK  # the real files were re-keyed
```

The first proves the key and nothing about the real files — `roundtrip-check`
is encrypted to every recipient by construction. The second proves both. Its
output goes to `/dev/null` on purpose: decrypting a real secret onto a screen
that is being shared is a routine way to leak one.

## Removing an operator

Dropping a key from `.sops.yaml` and re-encrypting stops them reading *future*
versions and nothing else. They hold a clone, or held one, and every value in it
is still valid.

**Removing an operator means rotating every secret they could read.** That is
expensive, and it is the argument for keeping the recipient list to the people
who genuinely need it rather than everyone who might.

Rotation otherwise has no cadence and should not acquire a calendar one. Rotate
on events: someone leaves, a credential appears on a screen or in a chat, a host
is rebuilt from an unknown state.

## Encrypting part of an environment

Not every dev secret is inert — a dev stack can need a real SMTP account to send
debug mail. Keep that value in a file of its own and encrypt only that file:

```bash
make secrets-encrypt ENVIRONMENT=dev FILES="smtp.env"
make secrets-decrypt ENVIRONMENT=dev FILES="smtp.env" FORCE=1
```

`FORCE=1` is needed on the way back because `gen-env` has usually already
written a placeholder at that path. It is opt-in for a reason: a local file may
hold a value that was never encrypted, and overwriting it loses the only copy.

One file per credential-owning service beats mixing real and generated values in
`api.env` — it keeps the blast radius of a forced decrypt to the file that is
genuinely shared. Compose reads a second file with no ceremony, and
`required: false` (Compose 2.24+) keeps the stack starting for a developer who
has no age key:

```yaml
    env_file:
      - ../env/${ENVIRONMENT:-dev}/api.env
      - path: ../env/${ENVIRONMENT:-dev}/smtp.env
        required: false
```

The alternative is `encrypted_regex` in `.sops.yaml`, which encrypts only the
matching variables and leaves the rest readable in git (sops matches with RE2,
so write the pattern as what to encrypt, not what to skip). It means committing
the whole file, generated dev values and all — prefer the separate file.

## Per-environment recipients

`.sops.yaml` carries separate rules for `prod` and for everything else, with the
same recipients in both to start with. The split is there so `dev` can later be
opened to the whole team while `prod` stays with the operators — a developer
brings the stack up from a clone with their own key and still cannot read prod.
Add the team to the `dev` rule when there is a non-inert dev secret to put in
it, not before: it costs every developer an age key.

Order matters — sops takes the **first** matching rule, so the prod rule stays
above the catch-all. `make secrets-add-recipient` appends a key to every rule,
which is right for an operator and wrong for a developer; add a dev-only
recipient by hand, then re-key that environment alone:

```bash
for f in env/sops/dev/*; do sops updatekeys -y "$f"; done
```

## The environment is part of the ciphertext path

`env/sops/<env>/` mirrors `env/<env>/`, and that is not decoration. `dev`,
`stage` and `prod` carry the same filenames here. With a flat `env/sops/`:

- encrypting `dev` overwrites the `prod` ciphertext with dev values;
- decrypting `dev` writes prod secrets into `env/dev/`;
- `secrets-check` passes both, because it asks whether a file is encrypted, not
  whether it holds what its path claims.

The collision disappears structurally rather than by discipline.

## What this does not solve

SOPS protects a credential at rest in a repository. It does nothing about the
same credential sitting in plaintext on the host that consumes it, and nothing
about console logins, registrar accounts or recovery keys — those want a
credential store that does not share a failure domain with what it recovers, and
a repository on a self-hosted git server is inside that domain.

If a secret has ever reached a remote in plaintext, removing the file does not
remove it from history. Rotate it; rewriting history is pointless once clones
have spread.
