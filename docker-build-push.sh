#!/usr/bin/env bash
# Build & push ultra-slim Kikundi Bora images (linux/amd64 by default).
# Usage:
#   export DOCKERHUB_USER=youruser
#   export TAG=latest            # optional
#   export PLATFORMS=linux/amd64 # or linux/amd64,linux/arm64
#   export VITE_API_URL=https://api.example.com/api/v1
#   ./docker-build-push.sh
#
# Push requires: docker login (Docker Hub) or ghcr.io auth.

set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
USER_NS="${DOCKERHUB_USER:-kikundi}"
TAG="${TAG:-latest}"
PLATFORMS="${PLATFORMS:-linux/amd64}"
VITE_API_URL="${VITE_API_URL:-http://localhost:8080/api/v1}"
PUSH="${PUSH:-1}"

BACKEND_IMAGE="${USER_NS}/kikundi-bora-backend:${TAG}"
FRONTEND_IMAGE="${USER_NS}/kikundi-bora-frontend:${TAG}"

echo "==> Backend image:  ${BACKEND_IMAGE}"
echo "==> Frontend image: ${FRONTEND_IMAGE}"
echo "==> Platforms:      ${PLATFORMS}"
echo "==> VITE_API_URL:   ${VITE_API_URL}"

# Ensure buildx builder exists
if ! docker buildx inspect kikundi-builder &>/dev/null; then
  docker buildx create --name kikundi-builder --use
else
  docker buildx use kikundi-builder
fi

PUSH_FLAG=()
if [[ "${PUSH}" == "1" ]]; then
  PUSH_FLAG=(--push)
else
  # load into local docker only works for single-platform
  PUSH_FLAG=(--load)
fi

echo ""
echo "==> Building backend..."
docker buildx build \
  --platform "${PLATFORMS}" \
  -t "${BACKEND_IMAGE}" \
  -f "${ROOT}/backend/Dockerfile" \
  "${ROOT}/backend" \
  "${PUSH_FLAG[@]}"

echo ""
echo "==> Building frontend..."
docker buildx build \
  --platform "${PLATFORMS}" \
  --build-arg "VITE_API_URL=${VITE_API_URL}" \
  -t "${FRONTEND_IMAGE}" \
  -f "${ROOT}/Frontend-1/Dockerfile" \
  "${ROOT}/Frontend-1" \
  "${PUSH_FLAG[@]}"

echo ""
echo "==> Done."
echo "    docker pull ${BACKEND_IMAGE}"
echo "    docker pull ${FRONTEND_IMAGE}"

if [[ "${PUSH}" != "1" ]]; then
  echo ""
  echo "Local images:"
  docker images --format "table {{.Repository}}\t{{.Tag}}\t{{.Size}}" \
    | grep -E "kikundi-bora|REPOSITORY" || true
fi
