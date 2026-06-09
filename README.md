# Ledger Service

[![Go](https://img.shields.io/badge/Go-1.22-00ADD8?logo=go)](https://go.dev/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-4169E1?logo=postgresql)](https://www.postgresql.org/)
[![Docker](https://img.shields.io/badge/Docker-ready-2496ED?logo=docker)](https://www.docker.com/)

Ledger Service is a small transactional wallet API written in Go. It keeps account balances in PostgreSQL and performs transfers inside a single database transaction with pessimistic row locks.

The project is intentionally compact: standard `net/http`, `chi` for routing, `pgx/v5` for PostgreSQL, and a clean split between delivery, usecase, domain, and repository layers.

## Features

- Create accounts with currency and initial balance
- Read account balance
- Transfer money between accounts atomically
- Store debit and credit ledger entries for every transfer
- Protect concurrent withdrawals with `SELECT ... FOR UPDATE`
- Graceful HTTP shutdown on `SIGINT` and `SIGTERM`
- Docker Compose setup with PostgreSQL and migrations

## Architecture

```text
cmd/api
  main.go

internal/domain
  account.go
  errors.go

internal/usecase
  account.go

internal/repository/postgres
  account.go

internal/delivery/http
  handler.go
  router.go

pkg/postgres
  postgres.go
```

The domain package owns the core types and repository contract. The usecase layer validates business rules before calling storage. The PostgreSQL repository owns transactions, row locks, balance updates, and ledger writes. HTTP handlers only translate JSON requests into usecase calls and map domain errors to status codes.

## Requirements

- Go 1.22+
- Docker and Docker Compose
- `golang-migrate` if you want to run migrations outside Docker

## Run With Docker

```bash
docker compose up --build
```

The API will listen on `http://localhost:8080`.

PostgreSQL is exposed on `localhost:5432` with:

```text
database: ledger
user: ledger
password: ledger
```

## Run Locally

Start PostgreSQL and apply migrations:

```bash
export DATABASE_URL="postgres://ledger:ledger@localhost:5432/ledger?sslmode=disable"
migrate -path migrations -database "$DATABASE_URL" up
go run ./cmd/api
```

Optional environment variables:

| Name | Default |
| --- | --- |
| `HTTP_ADDR` | `:8080` |
| `HTTP_READ_TIMEOUT` | `5s` |
| `HTTP_WRITE_TIMEOUT` | `10s` |
| `HTTP_IDLE_TIMEOUT` | `1m` |
| `POSTGRES_MAX_CONNS` | `10` |
| `POSTGRES_MIN_CONNS` | `2` |
| `POSTGRES_MAX_CONN_LIFETIME` | `1h` |

## API

### Create Account

```bash
curl -X POST http://localhost:8080/accounts \
  -H "Content-Type: application/json" \
  -d '{"user_id":"user-1","currency":"USD","balance":10000}'
```

Response:

```json
{
  "id": "d35f62f0-3f92-4096-8a70-ec8201dd2c7d",
  "user_id": "user-1",
  "currency": "USD",
  "balance": 10000,
  "created_at": "2026-06-09T12:40:00Z",
  "updated_at": "2026-06-09T12:40:00Z"
}
```

### Get Account

```bash
curl http://localhost:8080/accounts/d35f62f0-3f92-4096-8a70-ec8201dd2c7d
```

### Create Transfer

```bash
curl -X POST http://localhost:8080/transfers \
  -H "Content-Type: application/json" \
  -d '{"from_account_id":"source-id","to_account_id":"target-id","amount":1500}'
```

Response:

```json
{
  "entries": [
    {
      "id": "entry-id-1",
      "account_id": "source-id",
      "transfer_id": "transfer-id",
      "direction": "debit",
      "amount": 1500,
      "balance_after": 8500,
      "created_at": "2026-06-09T12:45:00Z"
    },
    {
      "id": "entry-id-2",
      "account_id": "target-id",
      "transfer_id": "transfer-id",
      "direction": "credit",
      "amount": 1500,
      "balance_after": 1500,
      "created_at": "2026-06-09T12:45:00Z"
    }
  ]
}
```

### List Account Ledger

```bash
curl http://localhost:8080/transfers/source-id
```

## Development

```bash
make test
make vet
make build
```

The service stores money as integer minor units. For example, `10000` can represent `100.00 USD`.

## Error Responses

```json
{
  "error": "insufficient funds"
}
```

Common statuses:

| Status | Reason |
| --- | --- |
| `400` | Invalid account data, invalid amount, same account transfer, currency mismatch |
| `404` | Account not found |
| `409` | Insufficient funds |
| `500` | Unexpected server error |
