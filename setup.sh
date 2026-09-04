#!/usr/bin/env bash
# =============================================================================
# Kikundi Bora — one-command setup script (Docker-based)
#
# Sets up the full stack:
#   1. PostgreSQL 16        (container, host port configurable, default 5433)
#   2. Backend Go API       -> http://localhost:8080
#   3. Frontend React SPA   -> http://localhost:8081
#
# Safe to re-run: it skips steps that are already done.
#
# Usage:
#   ./setup.sh              # full setup + start + verify
#   ./setup.sh --no-start   # configure & build only, don't start
#   ./setup.sh --reset-db   # DANGER: wipe the database volume, then set up
# =============================================================================
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT"

BACKEND_PORT="${BACKEND_PORT:-8080}"
FRONTEND_PORT="${FRONTEND_PORT:-8081}"
DB_HOST_PORT="${DB_HOST_PORT:-5433}"   # avoid clashing with a local Postgres on 5432
ADMIN_PASSWORD="${ADMIN_PASSWORD:-$(openssl rand -hex 16)}"

NO_START=0
RESET_DB=0
for arg in "$@"; do
  case "$arg" in
    --no-start) NO_START=1 ;;
    --reset-db) RESET_DB=1 ;;
    -h|--help) grep '^#' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *) echo "Unknown option: $arg (try --help)"; exit 1 ;;
  esac
done

say()  { printf '\n\033[1;36m==> %s\033[0m\n' "$*"; }
ok()   { printf '    \033[1;32m✔\033[0m %s\n' "$*"; }
warn() { printf '    \033[1;33m!\033[0m %s\n' "$*"; }
die()  { printf '\n\033[1;31mERROR:\033[0m %s\n' "$*" >&2; exit 1; }

# -----------------------------------------------------------------------------
say "[1/7] Checking prerequisites"
# -----------------------------------------------------------------------------
command -v docker >/dev/null 2>&1 || die "docker is not installed. Install Docker first: https://docs.docker.com/engine/install/"
docker info >/dev/null 2>&1       || die "Docker daemon is not running. Start it (e.g. 'sudo systemctl start docker') and retry."
ok "docker $(docker --version | awk '{print $3}' | tr -d ',')"

# -----------------------------------------------------------------------------
say "[2/7] Generating backend/.env"
# -----------------------------------------------------------------------------
if [ -f backend/.env ]; then
  ok "backend/.env already exists — keeping it"
else
  JWT_SECRET="$(openssl rand -hex 32 2>/dev/null || head -c 32 /dev/urandom | od -An -tx1 | tr -d ' \n')"
  DB_PASSWORD="$(openssl rand -hex 16 2>/dev/null || head -c 16 /dev/urandom | od -An -tx1 | tr -d ' \n')"
  cat > backend/.env <<EOF
DB_HOST=db
DB_PORT=5432
DB_USER=kikundi
DB_PASSWORD=$DB_PASSWORD
DB_NAME=kikundi_db
DB_SSLMODE=disable
JWT_SECRET=$JWT_SECRET
PORT=8080
PUBLIC_BASE_URL=http://localhost:$BACKEND_PORT
CORS_ORIGINS=http://localhost:3000,http://localhost:5173,http://localhost:$FRONTEND_PORT
ENVIRONMENT=development
ADMIN_PASSWORD=$ADMIN_PASSWORD
EOF
  ok "created backend/.env (random JWT secret generated)"
fi

# -----------------------------------------------------------------------------
say "[3/7] Choosing a free host port for PostgreSQL"
# -----------------------------------------------------------------------------
port_in_use() { command -v ss >/dev/null && ss -tln 2>/dev/null | grep -q ":$1 " ; }
if port_in_use "$DB_HOST_PORT"; then
  warn "host port $DB_HOST_PORT is busy"
  for p in 5434 5435 5436 5437; do
    if ! port_in_use "$p"; then DB_HOST_PORT="$p"; break; fi
  done
fi
export DB_HOST_PORT
echo "DB_HOST_PORT=$DB_HOST_PORT" > .env
ok "Postgres will be published on host port $DB_HOST_PORT (internal traffic uses the docker network)"

# -----------------------------------------------------------------------------
say "[4/7] Building images"
# -----------------------------------------------------------------------------
docker compose build
ok "backend & frontend images built"

# -----------------------------------------------------------------------------
say "[5/7] Starting containers"
# -----------------------------------------------------------------------------
if [ "$RESET_DB" = "1" ]; then
  warn "--reset-db given: dropping database volume"
  docker compose down -v
fi
docker compose up -d
ok "containers started"

# Wait for db to be healthy
printf '    waiting for PostgreSQL '
for i in $(seq 1 30); do
  if [ "$(docker inspect -f '{{.State.Health.Status}}' kikundi-db 2>/dev/null)" = "healthy" ]; then echo ""; ok "PostgreSQL healthy"; break; fi
  printf '.'; sleep 2
  [ "$i" = "30" ] && { echo ""; die "PostgreSQL did not become healthy. Run: docker logs kikundi-db"; }
done

# Wait for backend API to respond
printf '    waiting for backend API '
for i in $(seq 1 30); do
  if curl -sf "http://localhost:$BACKEND_PORT/health" >/dev/null 2>&1; then echo ""; ok "backend responding"; break; fi
  printf '.'; sleep 2
  [ "$i" = "30" ] && { echo ""; die "Backend did not come up. Run: docker logs kikundi-backend"; }
done

# -----------------------------------------------------------------------------
say "[6/7] Running database migration + seed (idempotent)"
# -----------------------------------------------------------------------------
# migrate is idempotent (AutoMigrate never drops tables + Seed skips
# existing rows), so always run it: existing databases need new tables
# (e.g. fine_settings/fines) even when other tables already exist.
docker exec kikundi-backend /app/kikundi-api -migrate > /dev/null 2>&1 \
  || die "migration failed. Run: docker exec kikundi-backend /app/kikundi-api -migrate"
ok "tables migrated and demo data seeded (idempotent)"

# -----------------------------------------------------------------------------
say "[7/7] Verifying the stack"
# -----------------------------------------------------------------------------
FAIL=0
curl -sf "http://localhost:$BACKEND_PORT/health" | grep -q '"ok"' \
  && ok "Backend health check passed (http://localhost:$BACKEND_PORT)" || { warn "Backend health check FAILED"; FAIL=1; }

TOKEN="$(curl -sf -X POST "http://localhost:$BACKEND_PORT/api/v1/auth/login" \
  -H 'Content-Type: application/json' \
  -d '{"email":"asha@kikundi.tz","password":"demo123"}' 2>/dev/null | grep -o '"token"' || true)"
[ -n "$TOKEN" ] && ok "Demo login works (asha@kikundi.tz)" || { warn "Demo login failed — data may not be seeded"; FAIL=1; }

curl -sf "http://localhost:$FRONTEND_PORT/" >/dev/null \
  && ok "Frontend responding (http://localhost:$FRONTEND_PORT)" || { warn "Frontend check FAILED"; FAIL=1; }

# -----------------------------------------------------------------------------
if [ "$NO_START" = "1" ]; then
  say "Setup complete (--no-start). Start later with: docker compose up -d"
elif [ "$FAIL" = "0" ]; then
  cat <<EOF

============================================================
  ✅ Kikundi Bora is running!

     Frontend : http://localhost:$FRONTEND_PORT
     Login    : http://localhost:$FRONTEND_PORT/ingia
     Backend  : http://localhost:$BACKEND_PORT
     Postgres : localhost:$DB_HOST_PORT (kikundi / kikundi_secret_2024)

     Demo accounts (password: demo123):
       Mwenyekiti   juma@kikundi.tz
       Mweka Hazina fatuma@kikundi.tz
       Katibu       rashidi@kikundi.tz
       Mwanachama   asha@kikundi.tz

  Useful commands:
     docker compose logs -f      # follow all logs
     docker compose down         # stop (data kept)
     docker compose down -v      # stop AND wipe database
     ./setup.sh --reset-db       # wipe + re-seed
============================================================
EOF
else
  die "Setup finished but some checks failed. Inspect: docker compose ps && docker compose logs --tail 50"
fi
