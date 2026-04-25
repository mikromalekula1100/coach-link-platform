COMPOSE = docker compose -f deployments/docker-compose.yml

.PHONY: up down restart build logs clean ps test-e2e test-integration \
        logs-auth logs-user logs-training logs-notification logs-gateway

## Start all services (build if needed)
up:
	$(COMPOSE) up -d --build

## Stop all services
down:
	$(COMPOSE) down

## Restart all services
restart: down up

## Build images without starting
build:
	$(COMPOSE) build

## Tail logs from all services
logs:
	$(COMPOSE) logs -f --tail=50

## Tail logs for individual services
logs-auth:
	$(COMPOSE) logs -f --tail=50 auth-service

logs-user:
	$(COMPOSE) logs -f --tail=50 user-service

logs-training:
	$(COMPOSE) logs -f --tail=50 training-service

logs-notification:
	$(COMPOSE) logs -f --tail=50 notification-service

logs-gateway:
	$(COMPOSE) logs -f --tail=50 api-gateway

logs-web:
	$(COMPOSE) logs -f --tail=50 web

## Show running containers
ps:
	$(COMPOSE) ps

## Full reset: stop, remove volumes, rebuild
clean:
	$(COMPOSE) down -v
	docker image prune -f

## Open Swagger UI in browser
swagger:
	open http://localhost:8090

## Run unit tests for all services (no Docker required)
test-unit:
	@echo "Running unit tests..."
	@for svc in ai-service analytics-service training-service notification-service; do \
		echo "--- $$svc ---"; \
		cd services/$$svc && go test ./internal/service/... -v && cd ../..; \
	done
	@echo "Unit tests done."

## Run E2E smoke test
test-e2e:
	@bash scripts/e2e-test.sh

## Run integration tests (requires services to be up: make up)
test-integration:
	@echo "Running integration tests against the API gateway..."
	$(COMPOSE) run --rm --build integration-test
