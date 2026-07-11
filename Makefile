.PHONY: run stop build test test-integration lint generate

run:
	docker compose up --build

stop:
	docker compose down

build:
	go build -o bin/banner-rotation ./cmd/banner-rotation

test:
	go test -race -count=1 ./...

test-integration:
	go test -v -tags integration -count=1 ./tests/integration/...

lint:
	golangci-lint run ./...

generate:
	go generate ./...
