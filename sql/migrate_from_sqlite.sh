#!/bin/sh
# Поднимает нативную схему и переносит файл SQLite с приведением типов.
# Нужны запущенный postgres и BOT_TOKEN: сервис aero требует его даже для migrate.
# Повтор затирает строки public снимком файла /data/aero.db.
set -eu
root=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
cd "$root"
docker compose run --rm aero migrate
docker compose exec -T postgres psql -U aero -d aero -v ON_ERROR_STOP=1 \
  -c "drop schema if exists legacy cascade;"
docker compose --profile migrate run --rm pgload
docker compose exec -T postgres psql -U aero -d aero -v ON_ERROR_STOP=1 \
  -f - < "$root/sql/migrate_from_sqlite.sql"
