# syntax=docker/dockerfile:1.7
# Ultra-slim multi-stage build for Kikundi Bora API (Go + Fiber)
# Target: gcr.io/distroless/static (no shell, non-root) — typically < 20MB

# ---------------------------------------------------------------------------
# Stage 1: build static binary
# ---------------------------------------------------------------------------
FROM golang:1.25-alpine AS builder

WORKDIR /src

RUN apk add --no-cache ca-certificates

# Cache modules first
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Static binary; GOARCH follows buildx --platform (do not hardcode amd64)
ENV CGO_ENABLED=0 GOOS=linux

RUN go build -trimpath -ldflags="-s -w" -o /out/kikundi-api . \
    && mkdir -p /out/uploads/avatars /out/uploads/docs /out/uploads/reports /out/backups

# ---------------------------------------------------------------------------
# Stage 2: certs only (for HTTPS outbound if needed later)
# ---------------------------------------------------------------------------
FROM alpine:3.21 AS certs
RUN apk add --no-cache ca-certificates && update-ca-certificates

# ---------------------------------------------------------------------------
# Stage 3: final runtime — distroless static, non-root
# ---------------------------------------------------------------------------
FROM gcr.io/distroless/static-debian12:nonroot AS final

WORKDIR /app

# TLS roots (email/SMTP optional; keeps image correct for HTTPS clients)
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Application binary only
COPY --from=builder /out/kikundi-api /app/kikundi-api

# Empty writable dirs (created in builder; owned by nonroot uid 65532)
COPY --from=builder --chown=nonroot:nonroot /out/uploads /app/uploads
COPY --from=builder --chown=nonroot:nonroot /out/backups /app/backups

USER nonroot:nonroot

ENV PORT=8080 \
    ENVIRONMENT=production \
    PUBLIC_BASE_URL=http://localhost:8080

EXPOSE 8080

# Distroless has no shell/curl. Probe from the host or compose network:
#   curl -f http://localhost:8080/health
ENTRYPOINT ["/app/kikundi-api"]
