#!/bin/sh
set -e

echo "Running migrations..."
ard run server/migrations.ard up

echo "Starting server..."
exec ard run server/main.ard
