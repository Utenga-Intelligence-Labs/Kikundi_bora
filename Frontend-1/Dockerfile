# syntax=docker/dockerfile:1.7
# Ultra-slim multi-stage build for Kikundi Bora frontend
# TanStack Start produces dist/client (assets) + dist/server (SSR).
# We ship static client assets via nginx (SPA shell) — target < 30MB.

# ---------------------------------------------------------------------------
# Stage 1: install + production build
# ---------------------------------------------------------------------------
FROM node:20-alpine AS builder

WORKDIR /app

# Install deps
COPY package.json package-lock.json ./
RUN npm install --no-audit --no-fund

COPY . .

# Bake API URL into the client bundle at build time (browser-facing)
ARG VITE_API_URL=http://localhost:8080/api/v1
ENV VITE_API_URL=$VITE_API_URL

RUN npm run build \
    && node scripts/generate-spa-index.mjs

# ---------------------------------------------------------------------------
# Stage 2: nginx alpine — static assets only
# ---------------------------------------------------------------------------
FROM nginx:1.27-alpine AS final

# Remove default site + unused conf noise
RUN rm -rf /usr/share/nginx/html/* \
    && rm -f /etc/nginx/conf.d/default.conf

COPY nginx.conf /etc/nginx/nginx.conf
COPY --from=builder /app/dist/client /usr/share/nginx/html

# Non-root: use unprivileged ports + writable temp/cache dirs
RUN chown -R nginx:nginx /usr/share/nginx/html \
    && mkdir -p /var/cache/nginx /var/run /tmp/nginx \
    && chown -R nginx:nginx /var/cache/nginx /var/run /tmp/nginx /var/log/nginx \
    && touch /var/run/nginx.pid \
    && chown nginx:nginx /var/run/nginx.pid

USER nginx

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -qO- http://127.0.0.1:8080/healthz || exit 1

CMD ["nginx", "-g", "daemon off;"]
