#!/usr/bin/env bash
# Создаёт дамп PostgreSQL для biopunk_api.
#
# Использование:
#   source scripts/env/dev.env && bash scripts/dump.sh
#
# Результат:
#   dumps/YYYY-MM-DD_HHMMSS/postgres.sql

set -euo pipefail

# ── Проверка переменных ───────────────────────────────────────────────────────

: "${POSTGRES_USER:?Не задана POSTGRES_USER}"
: "${POSTGRES_DB:?Не задана POSTGRES_DB}"
: "${COMPOSE_PROJECT_DIR:?Не задана COMPOSE_PROJECT_DIR}"

# ── Подготовка ────────────────────────────────────────────────────────────────

TIMESTAMP=$(date +%Y-%m-%d_%H%M%S)
DUMP_DIR="$(dirname "$0")/../dumps/${TIMESTAMP}"
mkdir -p "$DUMP_DIR"

compose() {
    docker compose --project-directory "$COMPOSE_PROJECT_DIR" "$@"
}

echo "==> Дамп в: $DUMP_DIR"

# ── PostgreSQL ────────────────────────────────────────────────────────────────

echo "==> [1/1] PostgreSQL dump..."

compose exec -T sber_knowledge_postgresql_dev \
    pg_dump -U "$POSTGRES_USER" "$POSTGRES_DB" \
    --no-owner --no-privileges \
    > "$DUMP_DIR/postgres.sql"

echo "    Готово: postgres.sql ($(du -sh "$DUMP_DIR/postgres.sql" | cut -f1))"

# ── Итог ──────────────────────────────────────────────────────────────────────

echo ""
echo "✓ Дамп завершён: $DUMP_DIR"
ls -lh "$DUMP_DIR"
