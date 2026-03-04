#!/bin/sh
set -e

echo "Running database migrations..."
/app/kazi-server -migrate

exec "$@"