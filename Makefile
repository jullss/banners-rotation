.PHONY: run stop build test lint generate

run:
	docker compose up --build

stop:
	docker compose down

build:
	go build -o bin/banner-rotation ./cmd/banner-rotation

test:
	go test ./...

lint:
	golangci-lint run ./...

generate:
	go generate ./...
