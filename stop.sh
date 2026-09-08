#!/usr/bin/env bash
# =============================================================================
# Kikundi Bora — stop every container/service started by ./setup.sh
#
#   ./stop.sh        # stop containers, KEEP data (docker compose down)
#   ./stop.sh -v     # stop AND wipe the database volume (destructive!)
#
# The host Nginx reverse proxy is NOT containerized — it is managed at the
# host level and intentionally left alone here.
# =============================================================================
set -euo pipefail
ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT"

# compose interpolates POSTGRES_PASSWORD for the db service even when just
# stopping — wire it the same way setup.sh does, or `down` fails on the
# required-variable check.
CONFIGURED_DB_PASSWORD="$(sed -n 's/^DB_PASSWORD=//p' backend/.env 2>/dev/null | head -n 1)"
if [ -z "${POSTGRES_PASSWORD:-}" ] && [ -n "$CONFIGURED_DB_PASSWORD" ]; then
  POSTGRES_PASSWORD="$CONFIGURED_DB_PASSWORD"
  export POSTGRES_PASSWORD
fi

if [ "${1:-}" = "-v" ]; then
  echo "Stopping AND removing volumes (database data will be wiped)..."
  docker compose down -v
else
  echo "Stopping containers (data volumes are kept)..."
  docker compose down
fi
echo "Kikundi Bora stopped. Start again with: ./setup.sh"
