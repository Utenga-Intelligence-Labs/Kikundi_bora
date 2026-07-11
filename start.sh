#!/usr/bin/env bash
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
BACKEND_DIR="$SCRIPT_DIR/backend"
FRONTEND_DIR="$SCRIPT_DIR/Frontend-1"
BACKEND_BIN="$BACKEND_DIR/kikundi-api"
BACKEND_LOG="/tmp/kikundi-api.log"
FRONTEND_LOG="/tmp/kikundi-frontend.log"
BACKEND_PORT=8080
FRONTEND_PORT=8081

cleanup() {
  echo ""
  echo "Shutting down..."
  kill $BACKEND_PID 2>/dev/null
  kill $FRONTEND_PID 2>/dev/null
  wait $BACKEND_PID 2>/dev/null
  wait $FRONTEND_PID 2>/dev/null
  echo "All stopped."
  exit 0
}
trap cleanup SIGINT SIGTERM EXIT

# ---- Pre-flight checks ----
if ! command -v go &>/dev/null; then
  echo "ERROR: Go is not installed."
  exit 1
fi

if ! command -v node &>/dev/null; then
  echo "ERROR: Node.js is not installed."
  exit 1
fi

if ! pg_isready -q 2>/dev/null; then
  echo "WARNING: PostgreSQL is not accepting connections."
  echo "         Make sure PostgreSQL is running before using the app."
fi

# ---- Backend ----
echo "[1/4] Setting up backend..."

cd "$BACKEND_DIR"

# Kill anything already on the backend port
if fuser "$BACKEND_PORT/tcp" &>/dev/null; then
  echo "       Port $BACKEND_PORT in use — freeing it..."
  fuser -k "$BACKEND_PORT/tcp" 2>/dev/null
  sleep 1
fi

if [ ! -f .env ]; then
  if [ -f .env.example ]; then
    cp .env.example .env
    echo "       .env created from .env.example — please review before starting!"
    echo "       Edit $BACKEND_DIR/.env then re-run this script."
    exit 0
  else
    echo "ERROR: No .env or .env.example found."
    exit 1
  fi
fi

if [ ! -f "$BACKEND_BIN" ]; then
  echo "       Building backend..."
  cd "$BACKEND_DIR"
  GONOSUMCHECK='*' GONOSUMDB='*' go build -o kikundi-api .
fi

echo "       Starting backend on :$BACKEND_PORT..."
cd "$BACKEND_DIR"
./kikundi-api > "$BACKEND_LOG" 2>&1 &
BACKEND_PID=$!

sleep 2
if ! kill -0 $BACKEND_PID 2>/dev/null; then
  echo "ERROR: Backend failed to start. Check $BACKEND_LOG"
  exit 1
fi
echo "       Backend running (PID $BACKEND_PID)"

# ---- Frontend ----
echo "[2/4] Setting up frontend..."

cd "$FRONTEND_DIR"

if [ ! -d node_modules ]; then
  echo "       Installing frontend dependencies..."
  npm install --silent
fi

echo "       Starting frontend on :$FRONTEND_PORT..."
npx vite dev --host 0.0.0.0 --port "$FRONTEND_PORT" > "$FRONTEND_LOG" 2>&1 &
FRONTEND_PID=$!

sleep 3
if ! kill -0 $FRONTEND_PID 2>/dev/null; then
  echo "ERROR: Frontend failed to start. Check $FRONTEND_LOG"
  kill $BACKEND_PID 2>/dev/null
  exit 1
fi
echo "       Frontend running (PID $FRONTEND_PID)"

# ---- Ready ----
echo ""
echo "============================================"
echo "  Kikundi is running!"
echo ""
echo "  Backend:   http://localhost:$BACKEND_PORT"
echo "  Frontend:  http://localhost:$FRONTEND_PORT"
echo "============================================"
echo ""
echo "Logs: $BACKEND_LOG / $FRONTEND_LOG"
echo "Press Ctrl+C to stop."
echo ""

wait
