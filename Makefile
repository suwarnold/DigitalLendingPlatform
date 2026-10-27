.PHONY: test vet build run-api run-worker compose-up compose-down tidy

MODULE := github.com/suwarnold/DigitalLendingPlatform

test:
	go test ./...

vet:
	go vet ./...

tidy:
	go mod tidy

build:
	mkdir -p bin
	go build -o bin/api ./cmd/api
	go build -o bin/worker ./cmd/worker

# Local defaults so `make run-api` works without exporting env vars.
# Override by exporting DATABASE_URL / JWT_SECRET.
DATABASE_URL ?= postgres://lending:lending@localhost:5432/lending?sslmode=disable
JWT_SECRET ?= change-me-to-a-long-random-secret

run-api:
	DATABASE_URL="$(DATABASE_URL)" JWT_SECRET="$(JWT_SECRET)" go run ./cmd/api

run-worker:
	DATABASE_URL="$(DATABASE_URL)" go run ./cmd/worker

compose-up:
	docker compose -f deployments/docker-compose.yml up --build

compose-down:
	docker compose -f deployments/docker-compose.yml down -v
