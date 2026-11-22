#!/bin/sh
set -eu

: "${RUNTIME_DB_PASSWORD:=naeos_runtime_dev_only}"
: "${MIGRATOR_DB_PASSWORD:=naeos_migrator_dev_only}"

psql --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" \
  --set runtime_password="$RUNTIME_DB_PASSWORD" \
  --set migrator_password="$MIGRATOR_DB_PASSWORD" \
  --set POSTGRES_DB="$POSTGRES_DB" \
  --file /docker-entrypoint-initdb.d/02-runtime-migrator-roles.sql
