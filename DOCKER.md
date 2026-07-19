# Kikundi Bora — Docker (ultra-slim)

Production-oriented multi-stage images for the API and UI.  
**Postgres is external** (managed DB or host Postgres). Do not bake secrets into images.

---

## Detected stack

| Layer | Stack |
|-------|--------|
| **Backend** | Go 1.25 · Fiber v2 · GORM · JWT · port `8080` · health `GET /health` |
| **Frontend** | React 19 · TanStack Start/Router · Vite 8 · npm · client assets in `dist/client` |
| **DB** | PostgreSQL (external) |

Frontend note: TanStack Start also emits `dist/server` (SSR). The Docker image ships **static `dist/client` via nginx** (SPA shell + generated `index.html`) for the smallest footprint and no Node runtime.

---

## Images

| Service | Base (final) | Port | Health |
|---------|--------------|------|--------|
| Backend | `gcr.io/distroless/static-debian12:nonroot` | `8080` | `GET /health` |
| Frontend | `nginx:1.27-alpine` (user `nginx`) | `8080` (mapped host `8081`) | `GET /healthz` |

Tag convention:

```text
<dockerhub-user>/kikundi-bora-backend:latest
<dockerhub-user>/kikundi-bora-frontend:latest
```

---

## Prerequisites

- Docker Engine + Buildx (Docker Desktop or Linux/Fedora/WSL2)
- External PostgreSQL with database `kikundi_db` (or your `DB_NAME`)
- `backend/.env` with real credentials (gitignored)

### Required `backend/.env`

```env
DB_HOST=host.docker.internal   # or real host/IP; use `db` if using compose Postgres
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your-password
DB_NAME=kikundi_db
DB_SSLMODE=disable
JWT_SECRET=generate-with-openssl-rand-hex-32-min-32-chars
ADMIN_PASSWORD=YourAdminPassword123
PORT=8080
PUBLIC_BASE_URL=http://localhost:8080
CORS_ORIGINS=http://localhost:8081,http://localhost:5173
ENVIRONMENT=production
```

On Linux, `host.docker.internal` may need:

```yaml
# already fine on Docker Desktop; on Linux compose add:
extra_hosts:
  - "host.docker.internal:host-gateway"
```

---

## Build locally (no push)

```bash
# From repo root
export DOCKERHUB_USER=youruser   # optional for local tags
export PUSH=0
export PLATFORMS=linux/amd64
export VITE_API_URL=http://localhost:8080/api/v1

./docker-build-push.sh
```

Or plain docker:

```bash
docker build -t kikundi/kikundi-bora-backend:latest ./backend

docker build \
  --build-arg VITE_API_URL=http://localhost:8080/api/v1 \
  -t kikundi/kikundi-bora-frontend:latest \
  ./Frontend-1
```

### Image size targets

| Image | Target | Estimated content |
|-------|--------|-------------------|
| Backend | **&lt; 20 MB** | stripped Linux binary ~17 MB + distroless base |
| Frontend | **&lt; 30 MB** | client assets ~2.5 MB + `nginx:1.27-alpine` |

Check:

```bash
docker images | grep kikundi-bora
```

> **Note:** Docker is not installed on the Windows host used for this scaffolding.
> Run builds on Fedora/WSL2 (or Docker Desktop) and fill the verification checklist below.

---

## Run with Compose

```bash
cp backend/.env.example backend/.env
# edit JWT_SECRET, ADMIN_PASSWORD, DB_*

docker compose up --build -d
```

| URL | Service |
|-----|---------|
| http://localhost:8080/health | Backend health |
| http://localhost:8080/api/v1 | API base |
| http://localhost:8081 | Frontend |

### First-time DB migrate / seed

Run once against the same DB (from host Go toolchain, or a one-off container):

```bash
# Host (recommended if Go is installed)
cd backend && go run . -migrate

# Or one-off (needs network access to DB)
docker compose run --rm --entrypoint /app/kikundi-api backend -migrate
```

Demo logins after seed (see root `README.md`): e.g. phone `0710000001` / `demo123`.

---

## Push to registry

```bash
docker login
# or: echo $GHCR_TOKEN | docker login ghcr.io -u USER --password-stdin

export DOCKERHUB_USER=youruser
export TAG=latest
export PLATFORMS=linux/amd64          # or linux/amd64,linux/arm64
export VITE_API_URL=https://api.yourdomain.co.tz/api/v1
export PUSH=1

./docker-build-push.sh
```

Pull elsewhere:

```bash
docker pull youruser/kikundi-bora-backend:latest
docker pull youruser/kikundi-bora-frontend:latest
```

---

## Security notes

- **No `.env` in images** — `.dockerignore` excludes all `.env*` files.
- Backend runs as **nonroot** (uid `65532` distroless).
- Frontend nginx runs as **nginx** (unprivileged port `8080`).
- Distroless final stage has **no shell**, no package manager, no compiler.
- Optional admin **pg_dump** backups need `pg_dump` on the host/sidecar — not bundled in the slim image.

### Verify no secrets in layers

```bash
docker history youruser/kikundi-bora-backend:latest --no-trunc
docker history youruser/kikundi-bora-frontend:latest --no-trunc
# Should not show .env contents or JWT_SECRET values
```

### Verify non-root

```bash
# Backend (distroless — no whoami; check process user)
docker run --rm --entrypoint /app/kikundi-api youruser/kikundi-bora-backend:latest -help 2>&1 || true
docker inspect --format '{{.Config.User}}' youruser/kikundi-bora-backend:latest
# → nonroot  (or 65532:65532)

docker run --rm --entrypoint cat youruser/kikundi-bora-frontend:latest /etc/passwd | grep nginx
docker inspect --format '{{.Config.User}}' youruser/kikundi-bora-frontend:latest
# → nginx
```

---

## Deliverables

| Path | Purpose |
|------|---------|
| `backend/Dockerfile` | Multi-stage Go → distroless |
| `backend/.dockerignore` | Exclude secrets, tests, local data |
| `Frontend-1/Dockerfile` | Multi-stage Node build → nginx |
| `Frontend-1/nginx.conf` | gzip, cache, SPA fallback, `/healthz` |
| `Frontend-1/.dockerignore` | Exclude node_modules, tests, secrets |
| `docker-compose.yml` | backend + frontend (+ optional db) |
| `docker-build-push.sh` | buildx build/tag/push |
| `DOCKER.md` | this file |

---

## Troubleshooting

| Issue | Fix |
|-------|-----|
| Backend exits: `JWT_SECRET is required` | Set in `backend/.env` (≥ 32 chars) |
| Backend: failed to connect to database | Fix `DB_HOST` / password; ensure Postgres accepts Docker network |
| Frontend blank / API CORS errors | Rebuild frontend with correct `VITE_API_URL`; add UI origin to `CORS_ORIGINS` |
| Compose healthcheck fails on backend | Distroless has no probe binary — probe `curl http://localhost:8080/health` from host |
| `host.docker.internal` on Linux | Add `extra_hosts: ["host.docker.internal:host-gateway"]` to backend service |
