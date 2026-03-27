# ── Stage 1: build web (Vue → dist/) ──────────────────────────────────────────
FROM oven/bun:1-alpine AS web-builder
WORKDIR /web
COPY web/package.json web/bun.lock ./
RUN bun install --frozen-lockfile
COPY web/ ./
RUN bun run build

# ── Stage 2: build API binary ──────────────────────────────────────────────────
FROM golang:1.24-alpine AS api-builder
WORKDIR /api
COPY api/go.mod api/go.sum ./
RUN go mod tidy
COPY api/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/telm ./cmd

# ── Stage 3: fetch OTel collector contrib ─────────────────────────────────────
FROM alpine:3.20 AS otelcol-fetcher
ARG OTELCOL_VERSION=0.120.0
ARG TARGETARCH=amd64
RUN apk add --no-cache curl && \
    curl -fsSL \
      "https://github.com/open-telemetry/opentelemetry-collector-releases/releases/download/v${OTELCOL_VERSION}/otelcol-contrib_${OTELCOL_VERSION}_linux_${TARGETARCH}.tar.gz" \
    | tar -xz -C /usr/local/bin otelcol-contrib

# ── Stage 4: final all-in-one image ───────────────────────────────────────────
FROM alpine:3.20
RUN apk add --no-cache \
    postgresql16 \
    postgresql16-contrib \
    supervisor \
    ca-certificates \
    tzdata \
    su-exec

# PostgreSQL data dir
ENV PGDATA=/var/lib/postgresql/data
RUN mkdir -p "$PGDATA" /run/postgresql /var/log/postgresql \
    && chown -R postgres:postgres "$PGDATA" /run/postgresql /var/log/postgresql \
    && chmod 700 "$PGDATA"

# App binary + SPA assets
COPY --from=api-builder /bin/telm /usr/local/bin/telm
COPY --from=web-builder /web/dist /web/dist

# OTel collector binary
COPY --from=otelcol-fetcher /usr/local/bin/otelcol-contrib /usr/local/bin/otelcol-contrib

# Runtime configs
COPY docker/supervisord.conf /etc/supervisor/conf.d/supervisord.conf
COPY docker/otelcol-config.yaml /etc/otelcol/config.yaml
COPY docker/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# 8080 = REST API + Web UI
# 4317 = OTLP gRPC  ← your apps send telemetry here
# 4318 = OTLP HTTP  ← alternative HTTP ingestion
EXPOSE 8080 4317 4318

VOLUME ["/var/lib/postgresql/data"]

ENTRYPOINT ["/entrypoint.sh"]
