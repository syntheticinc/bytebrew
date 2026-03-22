.PHONY: build build-release run test test-integration lint docker-up docker-down migrate-up migrate-down

build:
	go build -o bin/server ./cmd/server
	go build -o bin/keygen ./cmd/keygen

build-release: ## Build obfuscated release binaries (requires: go install mvdan.cc/garble@latest)
	garble -literals -tiny build -ldflags "-s -w" -o bin/server ./cmd/server
	garble -literals -tiny build -ldflags "-s -w" -o bin/keygen ./cmd/keygen

run: docker-up
	go run ./cmd/server

test:
	go test ./... -v

test-integration:
	go test -tags integration -v -timeout 300s ./...

lint:
	golangci-lint run

docker-up:
	docker compose up -d

docker-down:
	docker compose down

migrate-up:
	go run ./cmd/server --migrate-only

migrate-down:
	@echo "Use golang-migrate CLI for down migrations"
