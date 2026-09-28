-include .env
export

.PHONY: run migrate migrate-down

run:
	go run ./cmd/trip-service

migrate:
	go tool goose -dir migrations postgres "$(DATABASE_URL)" up

migrate-down:
	go tool goose -dir migrations postgres "$(DATABASE_URL)" down