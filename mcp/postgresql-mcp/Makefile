.PHONY: build clean install test run deps

build:
	@mkdir -p bin
	go build -o bin/postgresql-mcp .

install: build
	@echo "Built bin/postgresql-mcp binary"

clean:
	rm -f bin/postgresql-mcp

test:
	go test ./...

run: build
	./bin/postgresql-mcp

deps:
	go mod tidy
	go mod download

.DEFAULT_GOAL := build
