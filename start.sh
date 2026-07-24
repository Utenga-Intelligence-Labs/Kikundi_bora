#!/usr/bin/env bash
set -e

ROOT="$(cd "$(dirname "$0")" && pwd)"
BACKEND="$ROOT/backend"
FRONTEND="$ROOT/Frontend-1"
BACKEND_BIN="$BACKEND/kikundi-api"

cleanup() {
  echo ""
  echo "Stopping..."
  kill $BE_PID 2>/dev/null
  kill $FE_PID 2>/dev/null
  wait $BE_PID 2>/dev/null
  wait $FE_PID 2>/dev/null
  echo "Done."
  exit 0
}
trap cleanup SIGINT SIGTERM

# ---- Free ports ----
fuser -k 8080/tcp 2>/dev/null || true
fuser -k 8081/tcp 2>/dev/null || true
sleep 1

# ---- Backend ----
echo "[1/2] Starting backend..."
cd "$BACKEND"

if [ ! -f .env ]; then
  cp .env.example .env
  echo "       Created .env from .env.example — review before first run."
fi

if [ ! -f "$BACKEND_BIN" ]; then
  echo "       Building backend (one-time)..."
  go build -o kikundi-api .
fi

./kikundi-api &
BE_PID=$!
sleep 2

if ! kill -0 $BE_PID 2>/dev/null; then
  echo "ERROR: Backend failed to start."
  exit 1
fi
echo "       Backend → http://localhost:8080 (PID $BE_PID)"

# ---- Frontend ----
echo "[2/2] Starting frontend..."
cd "$FRONTEND"

if [ ! -d node_modules ]; then
  echo "       Installing dependencies (one-time)..."
  npm install --silent
fi

npx vite dev --host 0.0.0.0 --port 8081 &
FE_PID=$!
sleep 4

if ! kill -0 $FE_PID 2>/dev/null; then
  echo "ERROR: Frontend failed to start."
  kill $BE_PID 2>/dev/null
  exit 1
fi
echo "       Frontend → http://localhost:8081 (PID $FE_PID)"

# ---- Ready ----
echo ""
echo "============================================"
echo "  Kikundi Bora is running!"
echo ""
echo "  Frontend: http://localhost:8081"
echo "  Backend:  http://localhost:8080"
echo ""
echo "  Login:    http://localhost:8081/ingia"
echo ""
echo "  Demo accounts (password: demo123):"
echo "    Mwenyekiti   juma@kikundi.tz"
echo "    Mweka Hazina fatuma@kikundi.tz"
echo "    Katibu       rashidi@kikundi.tz"
echo "    Mwanachama   asha@kikundi.tz"
echo "============================================"
echo ""
echo "Press Ctrl+C to stop."
echo ""

wait
