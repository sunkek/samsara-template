SHELL := /bin/bash

USER_ID := $(shell id -u)
GROUP_ID := $(shell id -g)

ENVIRONMENT ?= dev
ENV_DIR := ./env
APP ?= my_project

# Base compose + per-environment override, merged in order. ENVIRONMENT drives
# which override (and which env/<env>/ dir + volume/container name suffix).
COMPOSE_FILES := -f deploy/docker-compose.yml -f deploy/docker-compose.$(ENVIRONMENT).yml
# Shared external bridge created once via `docker network create dev`.
NETWORK := dev
# Postgres container for this environment (psql / migrate / dump targets).
PG_CONTAINER := my_project_postgresql_$(ENVIRONMENT)

POSTGRES_ENV := $(ENV_DIR)/$(ENVIRONMENT)/postgresql.env
PORTS_ENV := $(ENV_DIR)/$(ENVIRONMENT)/ports.env
MIGRATIONS_DIR := ./deploy/postgresql/migration
BACKUP_DIR := ./deploy/postgresql/backup

# Host-side port mappings live in $(PORTS_ENV). Source it before every docker
# compose invocation so the file's variables drive port interpolation, and
# export ENVIRONMENT so the compose files resolve their env/<env>/ paths and
# container/volume name suffixes.
COMPOSE_WITH_PORTS = set -a; ENVIRONMENT=$(ENVIRONMENT); [ -f "$(PORTS_ENV)" ] && . "$(PORTS_ENV)"; set +a; docker compose $(COMPOSE_FILES)

.PHONY: help gen-env gen-key-hex gen-key-b64 gen-api-docs \
	up down down-v restart restart-v run run-local stop logs ps pull \
	psql psql-admin test-integration \
	migrate-new migrate-up migrate-down migrate-force \
	pg-dump pg-restore

help:
	@echo "Targets:"
	@echo "  make up [ENVIRONMENT=dev|stage|prod] - Start full stack in Docker for the environment"
	@echo "  make down                     - Stop and remove all containers"
	@echo "  make down-v                   - Stop and remove all containers and volumes"
	@echo "  make restart                  - down then up"
	@echo "  make restart-v                - down-v then up"
	@echo "  make run                      - Start infra only (docker compose up -d)"
	@echo "  make run-local                - Start dev infra; run backend (air) + frontend (vite) on host"
	@echo "  make stop                     - Stop and remove containers"
	@echo "  make logs                     - Follow compose logs"
	@echo "  make psql                     - Open psql as APP_USER"
	@echo "  make psql-admin               - Open psql as POSTGRES_USER"
	@echo "  make migrate-new n=<name>     - Create migration file pair"
	@echo "  make migrate-up               - Apply all up migrations"
	@echo "  make migrate-down             - Roll back one migration"
	@echo "  make migrate-force v=<num>    - Force migration version"
	@echo "  make test-integration         - Run tagged integration tests against dev infra"
	@echo "  make pg-dump [DUMP_FILE=...]  - Dump APP_DB to SQL file"
	@echo "  make pg-restore DUMP_FILE=... - Restore APP_DB from SQL file"

# Generators

# Environments materialized in one gen-env run. They SHARE a single secret
# pool, so the host backend authenticates against the dockerized infra whether
# it loads env/local (run-local) or env/dev — these two intentionally hold the
# same credentials. Generate stage/prod in SEPARATE invocations so they each
# get distinct secrets (never share a prod secret with dev):
#   make gen-env GEN_ENVS=prod APP=my_project
GEN_ENVS ?= dev local

# gen-env materializes env/<env>/*.env from env/example/*.env, replacing the
# "password" placeholders with random secrets and "app" with $(APP).
#
# Two consistency rules, both critical — get either wrong and the backend fails
# to authenticate against its own infra (SQLSTATE 28P01 etc.):
#   1. The same logical credential lives in two files (e.g. the Postgres app
#      password is in postgresql.env's APP_PASSWORD AND api.env's
#      _API_POSTGRESQL_PASS), so each secret is mapped by variable name + file.
#   2. run-local brings infra up from env/dev but loads the backend from
#      env/local, so the secret pool is generated ONCE and shared across every
#      environment in GEN_ENVS — dev and local get identical credentials.
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

gen-api-docs:
	cd ./service/backend && \
	swag fmt -d ./cmd/main && \
	swag init -d ./cmd/main -o ./docs --parseInternal --parseDependency --parseDependencyLevel=1

# Runtime

# Full stack in Docker for the chosen ENVIRONMENT (dev = hot reload; stage/prod
# = built images, nginx-served frontend). --build keeps images in sync with the
# Dockerfiles. Backend needs go.sum committed (`cd service/backend && go mod
# tidy` once). Env files must exist: `make gen-env GEN_ENVS="$(ENVIRONMENT)" APP=...`.
up:
	$(COMPOSE_WITH_PORTS) --profile app up -d --build

down: stop

down-v:
	$(COMPOSE_WITH_PORTS) --profile app down -v

restart:
	$(MAKE) down up

restart-v:
	$(MAKE) down-v up

restart-local-:
	$(MAKE) down-v run-local

restart-local-v:
	$(MAKE) down-v run-local

run:
	$(COMPOSE_WITH_PORTS) up -d

# run-local is always the dev environment: dev infra in Docker, backend (air)
# and frontend (vite) on the host. It sources env/local/* directly — the
# backend's air launch reads env/local/api.env via -l, so the vite proxy and
# the compose host-port mapping must agree with that same file.
LOCAL_PORTS_ENV := $(ENV_DIR)/local/ports.env
LOCAL_API_ENV   := $(ENV_DIR)/local/api.env

run-local:
	@set -a; \
	ENVIRONMENT=dev; \
	[ -f "$(LOCAL_PORTS_ENV)" ] && . "$(LOCAL_PORTS_ENV)"; \
	[ -f "$(LOCAL_API_ENV)" ] && . "$(LOCAL_API_ENV)"; \
	set +a; \
	docker compose -f deploy/docker-compose.yml -f deploy/docker-compose.dev.yml up -d postgresql rabbitmq redis; \
	trap 'kill 0' EXIT; \
	(cd ./service/backend && air -c .air.toml) & \
	(cd ./service/frontend && npm run dev); \
	wait

stop:
	$(COMPOSE_WITH_PORTS) --profile app down

logs:
	$(COMPOSE_WITH_PORTS) logs -f

ps:
	$(COMPOSE_WITH_PORTS) ps

pull:
	$(COMPOSE_WITH_PORTS) pull

# PostgreSQL

psql:
	@set -a; source "$(POSTGRES_ENV)"; set +a; \
	PGPASSWORD="$$APP_PASSWORD" docker exec -it $(PG_CONTAINER) \
		psql -U "$$APP_USER" -d "$$APP_DB"

psql-admin:
	@set -a; source "$(POSTGRES_ENV)"; set +a; \
	PGPASSWORD="$$POSTGRES_PASSWORD" docker exec -it $(PG_CONTAINER) \
		psql -U "$$POSTGRES_USER" -d postgres

# Run the integration-tagged tests (service/backend/internal/integration)
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
	cd service/backend && go test -tags=integration ./internal/integration/...

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
	docker run --rm -u $(USER_ID):$(GROUP_ID) -v "$(PWD)/deploy/postgresql/migration:/migration" \
		migrate/migrate create -ext sql -dir /migration -seq $(n)

migrate-up:
	@set -a; source "$(POSTGRES_ENV)"; set +a; \
	docker run --rm -v "$(PWD)/deploy/postgresql/migration:/migration" --network "$(NETWORK)" \
		migrate/migrate -path=/migration \
		-database "postgres://$$POSTGRES_USER:$$POSTGRES_PASSWORD@$(PG_CONTAINER):5432/$$APP_DB?sslmode=disable" \
		up; \
	PGPASSWORD="$$POSTGRES_PASSWORD" docker exec -i $(PG_CONTAINER) \
		psql -U "$$POSTGRES_USER" -d "$$APP_DB" \
		-c "GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO $$APP_USER;"

migrate-down:
	@set -a; source "$(POSTGRES_ENV)"; set +a; \
	docker run --rm -v "$(PWD)/deploy/postgresql/migration:/migration" --network "$(NETWORK)" \
		migrate/migrate -path=/migration \
		-database "postgres://$$POSTGRES_USER:$$POSTGRES_PASSWORD@$(PG_CONTAINER):5432/$$APP_DB?sslmode=disable" \
		down 1

migrate-force:
	@[ -n "$(v)" ] || (echo "Usage: make migrate-force v=<version>" && exit 1)
	@set -a; source "$(POSTGRES_ENV)"; set +a; \
	docker run --rm -v "$(PWD)/deploy/postgresql/migration:/migration" --network "$(NETWORK)" \
		migrate/migrate -path=/migration \
		-database "postgres://$$POSTGRES_USER:$$POSTGRES_PASSWORD@$(PG_CONTAINER):5432/$$APP_DB?sslmode=disable" \
		force $(v)
