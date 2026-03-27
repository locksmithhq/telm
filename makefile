.PHONY: help up down restart logs logs-api logs-web logs-db logs-gen clean build rebuild gen gen-stop publish

ENV_FILE=.env
GREEN=\033[0;32m
YELLOW=\033[1;33m
NC=\033[0m

-include $(ENV_FILE)

help: ## Show this help message
	@echo "$(GREEN)telm — available commands:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(YELLOW)%-15s$(NC) %s\n", $$1, $$2}'

up: ## Start all services
	@echo "$(GREEN)Starting services...$(NC)"
	@docker compose -f compose.yaml --env-file $(ENV_FILE) up --build -d
	@echo "$(GREEN)All services are running!$(NC)"
	@echo "$(YELLOW)App:    http://localhost:${APP_PORT}$(NC)"
	@echo "$(YELLOW)gRPC:   localhost:${GRPC_PORT}$(NC)"
	@echo "$(YELLOW)pgweb:  use 'make shell-pgweb'$(NC)"

down: ## Stop all services
	@docker compose -f compose.yaml --env-file $(ENV_FILE) --profile gen down
	@echo "$(GREEN)All services stopped$(NC)"

restart: down up ## Restart all services

logs: ## Tail logs for all services
	@docker compose -f compose.yaml --env-file $(ENV_FILE) logs -f

logs-api: ## Tail API logs
	@docker compose -f compose.yaml --env-file $(ENV_FILE) logs api -f

logs-web: ## Tail frontend logs
	@docker compose -f compose.yaml --env-file $(ENV_FILE) logs web -f

logs-db: ## Tail database logs
	@docker compose -f compose.yaml --env-file $(ENV_FILE) logs database -f

logs-gen: ## Tail load generator logs
	@docker compose -f compose.yaml --env-file $(ENV_FILE) --profile gen logs gen -f

gen: ## Start the load generator
	@echo "$(GREEN)Starting load generator...$(NC)"
	@docker compose -f compose.yaml --env-file $(ENV_FILE) --profile gen up gen -d --force-recreate
	@echo "$(GREEN)Generator started! Use 'make logs-gen' to see output$(NC)"

gen-stop: ## Stop the load generator
	@docker compose -f compose.yaml --env-file $(ENV_FILE) --profile gen stop gen
	@echo "$(GREEN)Generator stopped$(NC)"

clean: ## Stop services and remove volumes
	@echo "$(YELLOW)Removing volumes...$(NC)"
	@docker compose -f compose.yaml --env-file $(ENV_FILE) --profile gen down -v
	@echo "$(GREEN)Cleanup complete$(NC)"

build: ## Rebuild Docker images
	@docker compose -f compose.yaml --env-file $(ENV_FILE) build

rebuild: down build up ## Stop, rebuild and start

status: ## Show container status
	@docker ps -a --filter "name=telm" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"

shell-api: ## Open a shell in the API container
	@docker exec -it api-telm /bin/sh

shell-db: ## Open psql in the database container
	@docker exec -it database-telm psql -U $${POSTGRES_USER:-telm} -d $${POSTGRES_DB:-telm}

prune: ## Remove unused Docker resources
	@docker system prune -f

open: ## Open the app in the browser
	@open http://localhost:${APP_PORT}

# ── Docker Hub ───────────────────────────────────────────────────────────────

HUB_IMAGE = booscaaa/telm-all-in-one
HUB_TAG   ?= latest

publish: ## Build multi-arch and push to Docker Hub (booscaaa/telm-all-in-one)
	@echo "$(GREEN)Building $(HUB_IMAGE):$(HUB_TAG) (linux/amd64,linux/arm64)...$(NC)"
	@docker buildx build \
		--platform linux/amd64,linux/arm64 \
		--file docker/Dockerfile \
		--tag $(HUB_IMAGE):$(HUB_TAG) \
		--no-cache \
		--push \
		.
	@echo "$(GREEN)Published: $(HUB_IMAGE):$(HUB_TAG)$(NC)"

# ── Single-container (docker run) ────────────────────────────────────────────

IMAGE ?= telm:latest

image-build: ## [run] Build the all-in-one image
	@echo "$(GREEN)Building $(IMAGE)...$(NC)"
	@docker build -f docker/Dockerfile -t $(IMAGE) .
	@echo "$(GREEN)Image ready: $(IMAGE)$(NC)"

image-run: ## [run] Start the all-in-one container (postgres+otel+api+web)
	@echo "$(GREEN)Starting telm...$(NC)"
	@docker run -d \
		--name telm \
		-p $${APP_PORT:-4000}:8080 \
		-p 4317:4317 \
		-p 4318:4318 \
		-v telm-data:/var/lib/postgresql/data \
		-e POSTGRES_USER=$${POSTGRES_USER:-telm} \
		-e POSTGRES_PASSWORD=$${POSTGRES_PASSWORD:-telm123} \
		-e POSTGRES_DB=$${POSTGRES_DB:-telm} \
		--restart unless-stopped \
		$(IMAGE)
	@echo "$(GREEN)Running!$(NC)"
	@echo "$(YELLOW)App:  http://localhost:$${APP_PORT:-4000}$(NC)"
	@echo "$(YELLOW)OTLP: localhost:4317 (gRPC) / localhost:4318 (HTTP)$(NC)"

image-stop: ## [run] Stop and remove the container
	@docker rm -f telm 2>/dev/null || true
	@echo "$(GREEN)Container stopped$(NC)"

image-logs: ## [run] Tail container logs
	@docker logs -f telm

image-shell: ## [run] Open a shell in the container
	@docker exec -it telm /bin/sh

image-clean: ## [run] Stop the container and remove data volume
	@docker rm -f telm 2>/dev/null || true
	@docker volume rm telm-data 2>/dev/null || true
	@echo "$(GREEN)Container and data removed$(NC)"

# ── Production (compose) ──────────────────────────────────────────────────────

prod-up: ## [prod] Build and start in production mode (compose)
	@echo "$(GREEN)Building and starting production...$(NC)"
	@docker compose -f docker/compose.prod.yaml up --build -d
	@echo "$(GREEN)Running!$(NC)"
	@echo "$(YELLOW)App:  http://localhost:$${APP_PORT:-4000}$(NC)"
	@echo "$(YELLOW)OTLP: localhost:4317 (gRPC) / localhost:4318 (HTTP)$(NC)"

prod-down: ## [prod] Stop production services
	@docker compose -f docker/compose.prod.yaml down

prod-logs: ## [prod] Tail production logs
	@docker compose -f docker/compose.prod.yaml logs -f

prod-build: ## [prod] Rebuild the image without starting
	@docker compose -f docker/compose.prod.yaml build

prod-clean: ## [prod] Stop and remove production volumes
	@docker compose -f docker/compose.prod.yaml down -v
