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

if [ "${1:-}" = "-v" ]; then
  echo "Stopping AND removing volumes (database data will be wiped)..."
  docker compose down -v
else
  echo "Stopping containers (data volumes are kept)..."
  docker compose down
fi
echo "Kikundi Bora stopped. Start again with: ./setup.sh"
