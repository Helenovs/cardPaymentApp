.PHONY: dev-backend dev-frontend redis test build

redis:
	docker compose up -d redis

dev-backend:
	cd backend && go run ./cmd/api

dev-frontend:
	cd frontend && npm run dev

test:
	cd backend && go test ./...
	cd frontend && npm test -- --run

build:
	cd backend && go build ./cmd/api
	cd frontend && npm run build

