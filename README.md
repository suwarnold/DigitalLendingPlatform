# Digital Lending Platform

Phase 1 foundation for a Go + PostgreSQL digital lending platform. This slice is a modular monolith with an API process, a background worker, borrower registration/login, and durable outbox scaffolding.

The product design lives in the [project wiki](https://github.com/suwarnold/DigitalLendingPlatform/wiki/Digital-Lending-Platform). Loan origination, payments, and the ledger are later phases.

## What is included

- Go API (`cmd/api`) and worker (`cmd/worker`)
- PostgreSQL migrations for borrowers, idempotency keys, and outbox events
- Borrower register / login / profile / stub identity verification
- JWT authentication, health probes, Prometheus metrics, and Swagger UI
- Docker Compose for local development
- Unit tests, Postgres-backed integration tests, and GitHub Actions CI

## Prerequisites

- Go 1.23+
- Docker (for Compose and local PostgreSQL)

## Local run

```bash
docker compose -f deployments/docker-compose.yml up --build
```

This starts PostgreSQL, the API on `http://localhost:8080`, and the worker. The API applies migrations on startup.

Without Compose, start PostgreSQL yourself, then:

```bash
make run-api
make run-worker
```

Those targets default to the values in `.env.example`. Override with `DATABASE_URL` / `JWT_SECRET` in the environment if needed.

## Try the API

Phone numbers are normalized to Indonesian E.164 (`0812...` becomes `+62812...`).

```bash
curl -sS -X POST http://localhost:8080/api/v1/borrowers \
  -H 'Content-Type: application/json' \
  -H "Idempotency-Key: $(uuidgen)" \
  -d '{
    "full_name": "Ada Lovelace",
    "phone_number": "081234567890",
    "email": "ada@example.com",
    "password": "supersecret"
  }'

curl -sS -X POST http://localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{
    "phone_number": "081234567890",
    "password": "supersecret"
  }'

TOKEN='<access_token>'

curl -sS http://localhost:8080/api/v1/borrowers/me \
  -H "Authorization: Bearer $TOKEN"

curl -sS -X POST http://localhost:8080/api/v1/borrowers/verify-identity \
  -H "Authorization: Bearer $TOKEN"
```

Probes and docs:

- `GET /healthz` — process liveness
- `GET /readyz` — PostgreSQL ping
- `GET /metrics` — Prometheus metrics
- `GET /docs` — Swagger UI (also `/swagger`)
- `GET /openapi.yaml` — OpenAPI 3 spec

OpenAPI source of truth is [`api/openapi.yaml`](api/openapi.yaml).

## Tests

```bash
make vet
make test
```

Unit tests always run. Integration tests in `tests/integration` skip unless `DATABASE_URL` is set:

```bash
export DATABASE_URL='postgres://lending:lending@localhost:5432/lending?sslmode=disable'
go test ./tests/integration
```

CI starts PostgreSQL as a service and runs the full suite.

## Layout

```
cmd/api                HTTP process
cmd/worker             Outbox poller
internal/app           HTTP router wiring
internal/borrower      Registration, login, profile, stub KYC
internal/outbox        Event append + worker + no-op publisher
internal/platform      Config, DB, auth, HTTP, metrics
migrations             Versioned SQL
deployments            Docker Compose
api                    OpenAPI spec + Swagger UI assets
```

## Notes

- Money is not handled in Phase 1. Later tables will use `BIGINT` amounts, never floating point.
- The worker publishes outbox events with a no-op publisher so later queue integration has a consumer already in place.
- This is not a production-ready lending system. Review accounting, regulatory, and provider requirements before launch.
