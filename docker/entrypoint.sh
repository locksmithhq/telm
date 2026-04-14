#!/bin/sh
set -e

# ── Defaults ─────────────────────────────────────────────────────────────────
export POSTGRES_HOST="${POSTGRES_HOST:-localhost}"
export POSTGRES_PORT="${POSTGRES_PORT:-5432}"
export POSTGRES_USER="${POSTGRES_USER:-telm}"
export POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-telm123}"
export POSTGRES_DB="${POSTGRES_DB:-telm}"
export SSL_MODE="${SSL_MODE:-disable}"
export HTTP_PORT="${HTTP_PORT:-8080}"
export GRPC_PORT="${GRPC_PORT:-9317}"
export JWT_SECRET="${JWT_SECRET:-}"
export ADMIN_EMAIL="${ADMIN_EMAIL:-}"
export ADMIN_PASSWORD="${ADMIN_PASSWORD:-}"
export HTTPS="${HTTPS:-false}"
export CORS_ORIGIN="${CORS_ORIGIN:-}"

# ── PostgreSQL init ───────────────────────────────────────────────────────────
if [ ! -f "$PGDATA/PG_VERSION" ]; then
    echo "[telm] Initializing PostgreSQL data directory..."
    su-exec postgres initdb -D "$PGDATA" \
        --username=postgres \
        --auth-local=trust \
        --auth-host=md5

    # Start postgres temporarily to create user and database
    su-exec postgres pg_ctl start -D "$PGDATA" \
        -o "-c listen_addresses='localhost'" -w -t 30

    su-exec postgres psql --username=postgres -v ON_ERROR_STOP=1 -c \
        "CREATE USER \"${POSTGRES_USER}\" WITH PASSWORD '${POSTGRES_PASSWORD}';"

    su-exec postgres psql --username=postgres -v ON_ERROR_STOP=1 -c \
        "CREATE DATABASE \"${POSTGRES_DB}\" OWNER \"${POSTGRES_USER}\";"

    su-exec postgres pg_ctl stop -D "$PGDATA" -m fast -w
    echo "[telm] PostgreSQL initialized."
fi

# ── Start all services via supervisord ───────────────────────────────────────
exec /usr/bin/supervisord -c /etc/supervisor/conf.d/supervisord.conf
