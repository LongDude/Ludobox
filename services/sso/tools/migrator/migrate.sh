#!/bin/sh
echo "Running migrations..."
go run main.go --user="$POSTGRES_USER" --password="$POSTGRES_PASSWORD" --host="$POSTGRES_HOST" --port="$POSTGRES_PORT" --dbname="$POSTGRES_DB" --migrations-path=./db/migrations