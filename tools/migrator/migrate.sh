#!/bin/sh
set -eu

echo "Running shared migrations..."
go run main.go \
  --user="$POSTGRES_USER" \
  --password="$POSTGRES_PASSWORD" \
  --host="$POSTGRES_HOST" \
  --port="$POSTGRES_PORT" \
  --dbname="$POSTGRES_DB" \
  --migrations-path="/workspace/db/migrations"
