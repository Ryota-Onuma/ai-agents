.PHONY: build clean install test

build:
	go build -o bin main.go

install: build
	@echo "Built postgresql-mcp binary"

clean:
	rm -f postgresql-mcp

test:
	go test ./...

run: build
	./postgresql-mcp

deps:
	go mod tidy
	go mod download

.DEFAULT_GOAL := build