APP_NAME=ledger-service

.PHONY: build test vet run migrate-up migrate-down

build:
	go build -o bin/$(APP_NAME) ./cmd/api

test:
	go test ./...

vet:
	go vet ./...

run:
	go run ./cmd/api

migrate-up:
	migrate -path migrations -database "$$DATABASE_URL" up

migrate-down:
	migrate -path migrations -database "$$DATABASE_URL" down 1
