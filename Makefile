-include .env
export

.PHONY: run migrate migrate-down generate lint test test-integration

run:
	go run ./cmd/trip-service

migrate:
	go tool goose -dir migrations postgres "$(DATABASE_URL)" up

migrate-down:
	go tool goose -dir migrations postgres "$(DATABASE_URL)" down

generate:
	go tool oapi-codegen -config api/oapi-codegen.yaml contracts/openapi/trip-service.openapi.yaml
	go generate ./internal/...

lint:
	golangci-lint run ./...

test:
	go test -race -count=1 -coverpkg=./internal/domain/...,./internal/usecase/... -coverprofile=coverage.out ./internal/...
	go tool cover -func=coverage.out | tail -n 1

test-integration:
	go test -race -count=2 -tags=integration ./internal/repository/...