.PHONY: dev dev-backend dev-frontend migrate healthcheck test build up down logs

dev:
	@trap 'kill 0' INT TERM EXIT; \
	$(MAKE) dev-backend & \
	$(MAKE) dev-frontend

dev-backend:
	go run ./cmd

dev-frontend:
	cd frontend && npm run dev

migrate:
	go run ./cmd migrate

healthcheck:
	go run ./cmd healthcheck

test:
	go test ./...

build:
	go build ./...

up:
	docker compose up -d --build

down:
	docker compose down

.PHONY: metrics
metrics:
	helm repo add vm https://victoriametrics.github.io/helm-charts/ --force-update
	helm repo update vm
	helm upgrade --install vmsingle vm/victoria-metrics-single \
		--version 0.49.0 --namespace monitoring --create-namespace \
		-f k8s/observability/victoria-metrics/values.yaml --wait --timeout 180s
