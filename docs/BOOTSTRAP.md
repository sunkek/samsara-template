# Bootstrapping a new repository

Read this when creating a project from the template — it is written so an agent
can do it without studying the template first. Template-only: `bootstrap.sh`
deletes this file from the fork.

## Commands

Run from the template checkout. `-y` never prompts and, when a required value is
missing, fails with the full list instead. `./bootstrap.sh -h` prints every flag.

```bash
# frontend only
./bootstrap.sh -y -d ../myapp -n myapp -t "My App" -a "Jane Doe" -E security@example.org -F frontend
# backend only
./bootstrap.sh -y -d ../myapp -n myapp -t "My App" -a "Jane Doe" -E security@example.org -m github.com/me/myapp/backend -F backend
# backend + Postgres
./bootstrap.sh -y -d ../myapp -n myapp -t "My App" -a "Jane Doe" -E security@example.org -m github.com/me/myapp/backend -F backend,postgresql
# full stack (omitting -F keeps every feature)
./bootstrap.sh -y -d ../myapp -n myapp -t "My App" -a "Jane Doe" -E security@example.org -m github.com/me/myapp/backend
```

`-n` is the lowercase slug (container, volume and database names); the env
prefix defaults to it upper-cased (`-e` to override). `-m` is required only
when `backend` is kept.

| `-F` | The fork has | Removed |
|---|---|---|
| `frontend` | React/Vite SPA, its nginx image, compose stacks, CI | `services/backend`, `api.env`, `docs/ARCHITECTURE.md`, `docs/adr/`, all infra (`infra/postgresql`, `redis`, `rabbitmq`) and their env files, Go CI jobs and Dependabot entries |
| `backend` | Go service: samsara supervisor, HTTP server, health, metrics — no domains | `services/frontend`, all infra, every sample domain and migration |
| `backend,postgresql` | the above plus `auth` (in-memory token denylist) and the `note` sample, migrations 1–2, integration tests | frontend, Redis, RabbitMQ, the `article`/`articlestats` samples |
| all five | everything | nothing |

Every fork also loses the template tooling: `bootstrap.sh`, the feature
renderer (`scripts/apply_features.sh`, `features.awk`, `features_test.sh`),
`docs/FEATURES.md`, this file, `CHANGELOG.md`, ADR 0001, the template-only CI
jobs and the template-only doc sections.

## Into an existing git repository

```bash
git -C ../myapp status                 # commit or stash first
./bootstrap.sh -y -f -d ../myapp -n myapp ...
git -C ../myapp status                 # review what the fork added and overwrote
```

The template's `.git` is never copied and the destination's `.git` is never
touched, so the repository keeps its history. A directory holding only `.git`
(a fresh `git init`) needs no `-f`. Anything else there is merged: template
files overwrite same-named ones (`README.md`, `.gitignore`, `LICENSE`, …), and
the rename and the marker render run over the whole destination, so an existing
file containing `my_project` / `MY_PROJECT` / `My Project` is rewritten too.

`-d .` renames the template checkout in place, keeping the *template's* history
and remote — not what you want for a new repository.

## After bootstrapping

```bash
cd ../myapp
(cd services/backend && go mod tidy)    # backend: CI waits for a committed go.sum
(cd services/frontend && npm install)   # frontend
docker network create dev               # once per machine; an "exists" error is fine
make gen-env APP=myapp                  # env/dev + env/local, git-ignored
git init                                # unless it already is a repository
git add -A && git commit -m "chore: bootstrap from samsara-template"
```

`POST-FORK.md` lists the docs whose prose the feature set cut; read those,
then delete it.

## Verifying

The same checks CI runs:

```bash
shellcheck -x -S warning scripts/*.sh
make secrets-check
(cd services/frontend && npm ci && npm run lint && npm test && npm run build)
(cd services/backend && test -z "$(gofmt -l .)" && go vet ./... && go build ./... && go test ./...)
grep -rnI 'my_project\|MY_PROJECT\|My Project' . --exclude-dir=node_modules --exclude-dir=.git   # expect nothing
```
