.PHONY: build test unit integration e2e smoke quality up down rebuild tidy

SERVICES := auth-org-service project-key-service limits-service provider-catalog-service audit-analytics-service api-gateway
GOENV := PATH=/usr/local/go/bin:$$PATH GOCACHE=/tmp/go-build GOMODCACHE=/tmp/go-mod

build:
	@for svc in $(SERVICES); do \
		echo "building $$svc"; \
		$(GOENV) go build ./services/$$svc; \
	done

test:
	@$(GOENV) go test ./...

unit:
	@$(GOENV) go test ./internal/... ./services/...

integration:
	@RUN_INTEGRATION=1 $(GOENV) go test ./tests/integration -count=1 -v

e2e:
	@RUN_E2E=1 $(GOENV) go test ./tests/e2e -count=1 -v

smoke:
	./scripts/smoke.sh

quality:
	bash ./scripts/quality_check.sh

tidy:
	@$(GOENV) go mod tidy

up:
	docker compose up -d --build

down:
	docker compose down -v

rebuild:
	docker compose down -v
	docker compose up -d --build
