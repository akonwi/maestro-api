#!/bin/sh
set -e

echo "Running migrations..."
cd server
ard run migrations.ard up

echo "Starting server..."
exec ard run main.ard
