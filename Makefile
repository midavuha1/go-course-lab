-include .env
export

.PHONY: run migrate migrate-down generate

run:
	go run ./cmd/trip-service

migrate:
	go tool goose -dir migrations postgres "$(DATABASE_URL)" up

migrate-down:
	go tool goose -dir migrations postgres "$(DATABASE_URL)" down

generate:
	go tool oapi-codegen -config api/oapi-codegen.yaml contracts/openapi/trip-service.openapi.yaml