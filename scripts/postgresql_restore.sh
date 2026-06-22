#!/usr/bin/env bash
# Восстанавливает PostgreSQL biopunk_api из дампа.
#
# Использование:
#   source scripts/env/dev.env && bash scripts/restore.sh dumps/2025-01-15_120000
#
# Аргументы:
#   $1 — путь к директории с дампом (обязательно)

set -euo pipefail

# ── Проверка аргументов и переменных ─────────────────────────────────────────

if [ -z "${1:-}" ]; then
    echo "Использование: source scripts/env/dev.env && bash scripts/restore.sh <dump_dir>" >&2
    echo "Пример: bash scripts/restore.sh dumps/2025-01-15_120000" >&2
    exit 1
fi

DUMP_DIR="$1"

if [ ! -d "$DUMP_DIR" ]; then
    echo "ОШИБКА: директория не найдена: $DUMP_DIR" >&2
    exit 1
fi

: "${POSTGRES_USER:?Не задана POSTGRES_USER}"
: "${POSTGRES_DB:?Не задана POSTGRES_DB}"
: "${COMPOSE_PROJECT_DIR:?Не задана COMPOSE_PROJECT_DIR}"

POSTGRES_DUMP="$DUMP_DIR/postgres.sql"

if [ ! -f "$POSTGRES_DUMP" ]; then
    echo "ОШИБКА: не найден $POSTGRES_DUMP" >&2
    exit 1
fi

compose() {
    docker compose --project-directory "$COMPOSE_PROJECT_DIR" "$@"
}

echo "==> Восстановление из: $DUMP_DIR"
echo ""
read -r -p "Продолжить? Данные в БД будут перезаписаны. [y/N] " confirm
if [[ "${confirm}" != "y" && "${confirm}" != "Y" ]]; then
    echo "Отменено."
    exit 0
fi

# ── PostgreSQL ────────────────────────────────────────────────────────────────

echo ""
echo "==> [1/1] PostgreSQL restore..."

# Останавливаем приложение чтобы не было активных соединений
compose stop sber_knowledge_api_dev 2>/dev/null || true

compose exec -T sber_knowledge_postgresql_dev \
    psql -U "$POSTGRES_USER" -c \
    "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname='${POSTGRES_DB}' AND pid <> pg_backend_pid();" \
    postgres > /dev/null

compose exec -T sber_knowledge_postgresql_dev \
    psql -U "$POSTGRES_USER" -c "DROP DATABASE IF EXISTS \"${POSTGRES_DB}\";" postgres

compose exec -T sber_knowledge_postgresql_dev \
    psql -U "$POSTGRES_USER" -c "CREATE DATABASE \"${POSTGRES_DB}\";" postgres

compose exec -T sber_knowledge_postgresql_dev \
    psql -U "$POSTGRES_USER" "$POSTGRES_DB" \
    < "$POSTGRES_DUMP"

echo "    Готово."

# ── Запускаем приложение обратно ──────────────────────────────────────────────

echo ""
echo "==> Запуск сервисов..."
compose start sber_knowledge_api_dev 2>/dev/null || true

echo ""
echo "✓ Восстановление завершено."
