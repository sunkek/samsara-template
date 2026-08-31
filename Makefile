SHELL := /bin/bash

USER_ID := $(shell id -u)
GROUP_ID := $(shell id -g)

ENVIRONMENT ?= dev
ENV_DIR := ./env
APP ?= my_project

# Base compose + per-environment override, merged in order. ENVIRONMENT drives
# which override (and which env/<env>/ dir + volume/container name suffix).
COMPOSE_FILES := -f infra/docker-compose.yml -f infra/docker-compose.$(ENVIRONMENT).yml
# Shared external bridge created once via `docker network create dev`.
NETWORK := dev
# feat:if postgresql
# Postgres container for this environment (psql / migrate / dump targets).
PG_CONTAINER := my_project_postgresql_$(ENVIRONMENT)

POSTGRES_ENV := $(ENV_DIR)/$(ENVIRONMENT)/postgresql.env
# feat:end
PORTS_ENV := $(ENV_DIR)/$(ENVIRONMENT)/ports.env
# feat:if postgresql
MIGRATIONS_DIR := ./infra/postgresql/migration
BACKUP_DIR := ./infra/postgresql/backup
# feat:end

# Host-side port mappings live in $(PORTS_ENV). Source it before every docker
# compose invocation so the file's variables drive port interpolation, and
# export ENVIRONMENT so the compose files resolve their env/<env>/ paths and
# container/volume name suffixes.
COMPOSE_WITH_PORTS = set -a; ENVIRONMENT=$(ENVIRONMENT); [ -f "$(PORTS_ENV)" ] && . "$(PORTS_ENV)"; set +a; docker compose $(COMPOSE_FILES)

.PHONY: help gen-env gen-key-hex gen-key-b64 \
	up down down-v restart restart-v restart-local restart-local-v \
	run run-local stop logs ps pull
# feat:if backend
.PHONY: gen-api-docs
# feat:end
# feat:if postgresql
.PHONY: psql psql-admin migrate-new migrate-up migrate-down migrate-force \
	pg-dump pg-restore
# feat:end
# feat:if backend,postgresql
.PHONY: test-integration
# feat:end

help:
	@echo "Targets:"
	@echo "  make gen-env APP=<name>       - Materialize env/<env>/*.env with fresh secrets"
	@echo "  make gen-key-hex              - Print a random 32-byte hex secret"
	@echo "  make gen-key-b64              - Print a random 32-byte base64 secret"
	@echo "  make up [ENVIRONMENT=dev|stage|prod] - Start full stack in Docker for the environment"
	@echo "  make down                     - Stop and remove all containers"
	@echo "  make down-v                   - Stop and remove all containers and volumes"
	@echo "  make restart                  - down then up"
	@echo "  make restart-v                - down-v then up"
	@echo "  make restart-local            - down then run-local"
	@echo "  make restart-local-v          - down-v then run-local"
# feat:if postgresql|redis|rabbitmq
	@echo "  make run                      - Start infra only (docker compose up -d)"
# feat:end
# feat:if backend,frontend
	@echo "  make run-local                - Start dev infra; run backend (air) + frontend (vite) on host"
# feat:end
# feat:if backend,!frontend
#~ 	@echo "  make run-local                - Start dev infra; run backend (air) on host"
# feat:end
# feat:if frontend,!backend
#~ 	@echo "  make run-local                - Run the frontend dev server (vite) on host"
# feat:end
	@echo "  make stop                     - Stop and remove containers"
	@echo "  make logs                     - Follow compose logs"
	@echo "  make ps                       - List compose containers"
	@echo "  make pull                     - Pull compose images"
# feat:if backend
	@echo "  make gen-api-docs             - Regenerate Swagger docs (swag fmt + swag init)"
# feat:end
# feat:if postgresql
	@echo "  make psql                     - Open psql as APP_USER"
	@echo "  make psql-admin               - Open psql as POSTGRES_USER"
	@echo "  make migrate-new n=<name>     - Create migration file pair"
	@echo "  make migrate-up               - Apply all up migrations"
	@echo "  make migrate-down             - Roll back one migration"
	@echo "  make migrate-force v=<num>    - Force migration version"
# feat:if backend
	@echo "  make test-integration         - Run tagged integration tests against dev infra"
# feat:end
	@echo "  make pg-dump [DUMP_FILE=...]  - Dump APP_DB to SQL file"
	@echo "  make pg-restore DUMP_FILE=... - Restore APP_DB from SQL file"
# feat:end

# Generators

# Environments materialized in one gen-env run. They SHARE a single secret
# pool, so dev and local intentionally hold the same credentials. Generate
# stage/prod in SEPARATE invocations so they each get distinct secrets (never
# share a prod secret with dev):
#   make gen-env GEN_ENVS=prod APP=my_project
GEN_ENVS ?= dev local

# gen-env materializes env/<env>/*.env from env/example/*.env, replacing the
# "password" placeholders with random secrets and "app" with $(APP). The recipe
# is ONE backslash-continued shell line, so feature markers cannot go inside it
# (the shell would join the marker comment onto the previous line and comment
# out the rest). The unused secrets in the pool are harmless: secret_for() only
# reaches a branch when the matching env file is present.
# feat:if postgresql|redis|rabbitmq
#
# Two consistency rules, both critical — get either wrong and the backend fails
# to authenticate against its own infra (SQLSTATE 28P01 etc.):
#   1. The same logical credential lives in two files (e.g. the Postgres app
#      password is in postgresql.env's APP_PASSWORD AND api.env's
#      _API_POSTGRESQL_PASS), so each secret is mapped by variable name + file.
#   2. run-local brings infra up from env/dev but loads the backend from
#      env/local, so the secret pool is generated ONCE and shared across every
#      environment in GEN_ENVS — dev and local get identical credentials.
# feat:end
gen-env:
	@set -e; \
	src_dir="$(ENV_DIR)/example"; \
	gen() { openssl rand -hex 32; }; \
	pg_app="$$(gen)"; pg_super="$$(gen)"; \
	mq_app="$$(gen)"; mq_admin="$$(gen)"; \
	redis_app="$$(gen)"; redis_admin="$$(gen)"; \
	jwt="$$(gen)"; \
	for env in $(GEN_ENVS); do \
		dst_dir="$(ENV_DIR)/$$env"; \
		mkdir -p "$$dst_dir"; \
		for src in "$$src_dir"/*.env; do \
			[ -e "$$src" ] || { echo "No .env files found in $$src_dir"; exit 1; }; \
			base="$$(basename "$$src")"; \
			dst="$$dst_dir/$$base"; \
			echo "Creating $$dst"; \
			awk -v app="$(APP)" -v file="$$base" \
				-v pg_app="$$pg_app" -v pg_super="$$pg_super" \
				-v mq_app="$$mq_app" -v mq_admin="$$mq_admin" \
				-v redis_app="$$redis_app" -v redis_admin="$$redis_admin" \
				-v jwt="$$jwt" '\
			function gen_secret() { \
				cmd = "openssl rand -hex 32"; cmd | getline r; close(cmd); return r; \
			} \
			function secret_for(name) { \
				if (name ~ /_API_POSTGRESQL_PASS$$/) return pg_app; \
				if (name ~ /_API_RABBITMQ_PASS$$/)   return mq_app; \
				if (name ~ /_API_REDIS_PASS$$/)      return redis_app; \
				if (name ~ /_API_JWT_SECRET$$/)      return jwt; \
				if (file == "postgresql.env") return (name ~ /APP/) ? pg_app : pg_super; \
				if (file == "rabbitmq.env")   return (name ~ /APP/) ? mq_app : mq_admin; \
				if (file == "redis.env")      return (name ~ /APP/) ? redis_app : redis_admin; \
				return gen_secret(); \
			} \
			{ \
				if ($$0 ~ /=("password"|password)$$/) { \
					name = $$0; \
					sub(/^[ \t]*export[ \t]+/, "", name); \
					sub(/=.*/, "", name); \
					sub(/=("password"|password)$$/, "=\"" secret_for(name) "\""); \
				} \
				if (app != "" && $$0 ~ /=("app"|app)$$/) { \
					sub(/=("app"|app)$$/, "=\"" app "\""); \
				} \
				print \
			}' "$$src" > "$$dst"; \
		done; \
	done

gen-key-hex:
	openssl rand --hex 32

gen-key-b64:
	openssl rand --base64 32

# feat:if backend
# The search dir is the module root, not cmd/main: the route annotations live on
# the handlers in internal/domain/*/adapter/fiber, and swag only walks the tree
# it is pointed at. -g names the file carrying the general API info. JSON+YAML
# only: the server serves docs/swagger.json as a static file, so the generated
# docs.go would add a swag import to the module for nothing.
gen-api-docs:
	cd ./services/backend && \
	swag fmt -d ./cmd/main -d ./internal && \
	swag init -g cmd/main/main.go -d ./ -o ./docs --outputTypes json,yaml --parseInternal --parseDependency --parseDependencyLevel=1

# feat:end

# Runtime

# Full stack in Docker for the chosen ENVIRONMENT (dev = hot reload; stage/prod
# = built images, nginx-served frontend). --build keeps images in sync with the
# Dockerfiles.
# feat:if backend
# Backend needs go.sum committed (`cd services/backend && go mod tidy` once).
# feat:end
# Env files must exist: `make gen-env GEN_ENVS="$(ENVIRONMENT)" APP=...`.
up:
	$(COMPOSE_WITH_PORTS) --profile app up -d --build

down: stop

down-v:
	$(COMPOSE_WITH_PORTS) --profile app down -v

restart:
	$(MAKE) down up

restart-v:
	$(MAKE) down-v up

restart-local:
	$(MAKE) down run-local

restart-local-v:
	$(MAKE) down-v run-local

run:
	$(COMPOSE_WITH_PORTS) up -d

# run-local is always the dev environment: infra in Docker, the app services on
# the host. It sources env/local/* directly.
# feat:if backend,frontend
# The backend's air launch reads env/local/api.env via -l, so the vite proxy and
# the compose host-port mapping must agree with that same file.
# feat:end
# Infra containers run-local brings up, and the host processes it launches.
# Both are feature-dependent; `true;` keeps the joined recipe line valid when a
# half is not selected.
LOCAL_INFRA :=
# feat:if postgresql
LOCAL_INFRA += postgresql
# feat:end
# feat:if rabbitmq
LOCAL_INFRA += rabbitmq
# feat:end
# feat:if redis
LOCAL_INFRA += redis
# feat:end

# feat:if backend
RUN_LOCAL_BACKEND := (cd ./services/backend && air -c .air.toml) &
# feat:else
#~ RUN_LOCAL_BACKEND := true;
# feat:end
# feat:if frontend
RUN_LOCAL_FRONTEND := (cd ./services/frontend && npm run dev);
# feat:else
#~ RUN_LOCAL_FRONTEND := true;
# feat:end

LOCAL_PORTS_ENV := $(ENV_DIR)/local/ports.env
LOCAL_API_ENV   := $(ENV_DIR)/local/api.env

run-local:
	@set -a; \
	ENVIRONMENT=dev; \
	[ -f "$(LOCAL_PORTS_ENV)" ] && . "$(LOCAL_PORTS_ENV)"; \
	[ -f "$(LOCAL_API_ENV)" ] && . "$(LOCAL_API_ENV)"; \
	set +a; \
	docker compose -f infra/docker-compose.yml -f infra/docker-compose.dev.yml up -d $(LOCAL_INFRA); \
	trap 'kill 0' EXIT; \
	$(RUN_LOCAL_BACKEND) \
	$(RUN_LOCAL_FRONTEND) \
	wait

stop:
	$(COMPOSE_WITH_PORTS) --profile app down

logs:
	$(COMPOSE_WITH_PORTS) logs -f

ps:
	$(COMPOSE_WITH_PORTS) ps

pull:
	$(COMPOSE_WITH_PORTS) pull

# feat:if postgresql
# PostgreSQL

psql:
	@set -a; source "$(POSTGRES_ENV)"; set +a; \
	PGPASSWORD="$$APP_PASSWORD" docker exec -it $(PG_CONTAINER) \
		psql -U "$$APP_USER" -d "$$APP_DB"

psql-admin:
	@set -a; source "$(POSTGRES_ENV)"; set +a; \
	PGPASSWORD="$$POSTGRES_PASSWORD" docker exec -it $(PG_CONTAINER) \
		psql -U "$$POSTGRES_USER" -d postgres

# feat:if backend
# Run the integration-tagged tests (services/backend/internal/integration)
# against the running dev infra. Bring infra up and apply migrations first:
#   make run && make migrate-up && make test-integration
# Builds INTEGRATION_DATABASE_URL from the env files; tests self-skip if it is
# unset, and connect to the host-published Postgres port.
test-integration:
	@set -a; \
	[ -f "$(PORTS_ENV)" ] && . "$(PORTS_ENV)"; \
	. "$(POSTGRES_ENV)"; \
	set +a; \
	export INTEGRATION_DATABASE_URL="postgres://$$APP_USER:$$APP_PASSWORD@localhost:$$MY_PROJECT_PG_PORT/$$APP_DB?sslmode=disable"; \
	echo "Running integration tests against localhost:$$MY_PROJECT_PG_PORT/$$APP_DB"; \
	cd services/backend && go test -tags=integration ./internal/integration/...
# feat:end

pg-dump:
	@set -euo pipefail; \
	set -a; source "$(POSTGRES_ENV)"; set +a; \
	mkdir -p "$(BACKUP_DIR)"; \
	dump_file="$(if $(DUMP_FILE),$(DUMP_FILE),$(BACKUP_DIR)/postgres_$$(date +%Y-%m-%d_%H%M%S).sql)"; \
	echo "Dumping $$APP_DB to $$dump_file"; \
	PGPASSWORD="$$POSTGRES_PASSWORD" docker exec -i $(PG_CONTAINER) \
		pg_dump -U "$$POSTGRES_USER" "$$APP_DB" --no-owner --no-privileges > "$$dump_file"; \
	echo "Done: $$dump_file"

pg-restore:
	@set -euo pipefail; \
	[ -n "$(DUMP_FILE)" ] || { echo "Usage: make pg-restore DUMP_FILE=<path/to/dump.sql>"; exit 1; }; \
	[ -f "$(DUMP_FILE)" ] || { echo "Dump file not found: $(DUMP_FILE)"; exit 1; }; \
	set -a; source "$(POSTGRES_ENV)"; set +a; \
	if [ "$(FORCE)" != "1" ]; then \
		read -r -p "Restore $$APP_DB from $(DUMP_FILE)? This will overwrite DB data [y/N] " confirm; \
		[[ "$$confirm" =~ ^[Yy]$$ ]] || { echo "Cancelled"; exit 0; }; \
	fi; \
	echo "Restoring $$APP_DB from $(DUMP_FILE)"; \
	PGPASSWORD="$$POSTGRES_PASSWORD" docker exec -i $(PG_CONTAINER) \
		psql -v ON_ERROR_STOP=1 -U "$$POSTGRES_USER" -d postgres \
		-c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname='$$APP_DB' AND pid <> pg_backend_pid();" \
		-c "DROP DATABASE IF EXISTS \"$$APP_DB\";" \
		-c "CREATE DATABASE \"$$APP_DB\";"; \
	PGPASSWORD="$$POSTGRES_PASSWORD" docker exec -i $(PG_CONTAINER) \
		psql -v ON_ERROR_STOP=1 -U "$$POSTGRES_USER" -d "$$APP_DB" < "$(DUMP_FILE)"; \
	echo "Restore complete"

# Migrations

migrate-new:
	@[ -n "$(n)" ] || (echo "Usage: make migrate-new n=<migration_name>" && exit 1)
	docker run --rm -u $(USER_ID):$(GROUP_ID) -v "$(PWD)/infra/postgresql/migration:/migration" \
		migrate/migrate create -ext sql -dir /migration -seq $(n)

migrate-up:
	@set -a; source "$(POSTGRES_ENV)"; set +a; \
	docker run --rm -v "$(PWD)/infra/postgresql/migration:/migration" --network "$(NETWORK)" \
		migrate/migrate -path=/migration \
		-database "postgres://$$POSTGRES_USER:$$POSTGRES_PASSWORD@$(PG_CONTAINER):5432/$$APP_DB?sslmode=disable" \
		up; \
	PGPASSWORD="$$POSTGRES_PASSWORD" docker exec -i $(PG_CONTAINER) \
		psql -U "$$POSTGRES_USER" -d "$$APP_DB" \
		-c "GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO $$APP_USER;"

migrate-down:
	@set -a; source "$(POSTGRES_ENV)"; set +a; \
	docker run --rm -v "$(PWD)/infra/postgresql/migration:/migration" --network "$(NETWORK)" \
		migrate/migrate -path=/migration \
		-database "postgres://$$POSTGRES_USER:$$POSTGRES_PASSWORD@$(PG_CONTAINER):5432/$$APP_DB?sslmode=disable" \
		down 1

migrate-force:
	@[ -n "$(v)" ] || (echo "Usage: make migrate-force v=<version>" && exit 1)
	@set -a; source "$(POSTGRES_ENV)"; set +a; \
	docker run --rm -v "$(PWD)/infra/postgresql/migration:/migration" --network "$(NETWORK)" \
		migrate/migrate -path=/migration \
		-database "postgres://$$POSTGRES_USER:$$POSTGRES_PASSWORD@$(PG_CONTAINER):5432/$$APP_DB?sslmode=disable" \
		force $(v)
# feat:end
