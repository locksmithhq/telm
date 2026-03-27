# telm

Self-hosted observability backend — collect traces, metrics and logs via OTLP and explore them through a web UI.

## Overview

telm is a lightweight, self-hosted platform for teams that want to collect and explore OpenTelemetry data without operating a full Grafana/Tempo/Loki/Prometheus stack. It stores all telemetry in PostgreSQL and ships as a single Docker image.

```
Your services
    │
    │  OTLP/gRPC (4317)  or  OTLP/HTTP (4318)
    ▼
┌─────────────────────────────────────────────┐
│                   telm                      │
│                                             │
│  OTel Collector → API (Go) → PostgreSQL     │
│                      ↕                      │
│               Web UI (Vue 3)                │
└─────────────────────────────────────────────┘
    │
    │  HTTP :8080
    ▼
 Browser
```

## Quick start

### Single container

```bash
docker run -d \
  --name telm \
  -p 4000:8080 \
  -p 4317:4317 \
  -p 4318:4318 \
  -v telm-data:/var/lib/postgresql/data \
  booscaaa/telm-all-in-one
```

Open **http://localhost:4000** — done.

Send telemetry from your services to `localhost:4317` (gRPC) or `localhost:4318` (HTTP).

#### Environment variables

| Variable | Default | Description |
|---|---|---|
| `POSTGRES_USER` | `telm` | PostgreSQL user |
| `POSTGRES_PASSWORD` | `telm123` | PostgreSQL password |
| `POSTGRES_DB` | `telm` | PostgreSQL database |
| `HTTP_PORT` | `8080` | Internal HTTP port (web UI + REST API) |

### Docker Compose (single service)

Create a `compose.yml` and run `docker compose up -d`:

```yaml
services:
  telm:
    image: booscaaa/telm-all-in-one
    restart: unless-stopped
    ports:
      - "4000:8080"   # Web UI + REST API
      - "4317:4317"   # OTLP gRPC
      - "4318:4318"   # OTLP HTTP
    volumes:
      - telm-data:/var/lib/postgresql/data
    environment:
      POSTGRES_PASSWORD: telm123

volumes:
  telm-data:
```

Open **http://localhost:4000** — done.

### Docker Compose (development)

```bash
git clone https://github.com/locksmithhq/telm.git
cd telm
cp .env.example .env
make up
```

| Command | Description |
|---|---|
| `make up` | Start all services |
| `make down` | Stop all services |
| `make logs` | Tail all logs |
| `make gen` | Start load generator (simulated e-commerce traffic) |
| `make gen-stop` | Stop load generator |
| `make rebuild` | Stop, rebuild and restart |
| `make status` | Show container status |

## Ports

| Port | Protocol | Description |
|---|---|---|
| `4000` | HTTP | Web UI + REST API (nginx proxy, dev only) |
| `8080` | HTTP | Web UI + REST API (direct) |
| `4317` | gRPC | OTLP gRPC receiver |
| `4318` | HTTP | OTLP HTTP receiver |
| `9317` | gRPC | Internal: OTel Collector → telm API |

## SDK

The companion Go SDK is available at [github.com/locksmithhq/telm-go](https://github.com/locksmithhq/telm-go).

```bash
go get github.com/locksmithhq/telm-go
```

```go
shutdown, err := telm.Init(ctx,
    telm.WithServiceName("my-service"),
    telm.WithEndpoint("localhost:4318"),
)
defer shutdown(ctx)

ctx, end := telm.Start(ctx, "db.users.find", telm.Client())
defer func() { end(err) }()

telm.Info(ctx, "user fetched", telm.F{"user_id": id})
telm.Count(ctx, "users.fetched", 1, telm.F{"region": "us-east"})
```

See [SDK.md](./SDK.md) for the full reference.

## Architecture (development)

```
browser → nginx (:4000) → Vue dev server (:3000)
                        → Go API       (:8080)

apps → OTel Collector (:4317/:4318) → Go API gRPC (:9317) → PostgreSQL
```

Services defined in `compose.yaml`:

| Service | Image | Role |
|---|---|---|
| `api` | golang + air | REST API + OTLP gRPC receiver |
| `web` | oven/bun | Vue 3 dev server |
| `proxy` | nginx | Reverse proxy |
| `otelcollector` | otel/opentelemetry-collector-contrib | OTLP receiver + forwarder |
| `database` | postgres 16 | Storage |
| `gen` _(profile)_ | golang | Load generator |

## Production build

```bash
make publish              # build multi-arch and push to Docker Hub
make publish HUB_TAG=1.2.0  # with a version tag
```

Requires an active `docker buildx` builder:

```bash
docker buildx create --use
```
