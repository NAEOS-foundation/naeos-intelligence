#!/bin/sh
set -eu

: "${RUNTIME_DB_PASSWORD:=run_dev}"
: "${MIGRATOR_DB_PASSWORD:=mig_dev}"
: "${ROLE_BOOTSTRAP_SQL_FILE:=/docker-entrypoint-initdb.d/02-runtime-migrator-roles.sql.in}"

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" \
  --set runtime_password="$RUNTIME_DB_PASSWORD" \
  --set migrator_password="$MIGRATOR_DB_PASSWORD" \
  --set POSTGRES_DB="$POSTGRES_DB" \
  --file "$ROLE_BOOTSTRAP_SQL_FILE"
