#!/bin/sh
set -eu

echo "Running shared migrations..."
go run main.go \
  --user="$DB_USER" \
  --password="$DB_PASSWORD" \
  --host="$DB_HOST" \
  --port="$DB_PORT" \
  --dbname="$DB_NAME" \
  --migrations-path="/workspace/db/migrations"
