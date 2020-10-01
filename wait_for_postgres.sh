#!/bin/bash
# wait_for_postgres.sh

set -e

host="$1"
user="$2"
shift 2
cmd="$@"
until psql -h "$host" -U "$user" -c '\l'; do
  >&2 echo "Postgres is unavailable - sleeping"
  sleep 1
done

>&2 echo "Postgres is up - executing command"
exec $cmd