.PHONY: build clean install test run deps

build:
	mkdir -p bin
	go build -o bin/postgresql-mcp-server .

install: build
	@echo "Built bin binary"

clean:
	rm -rf bin

test:
	go test ./...

run: build
	@echo "Usage: ./bin/postgresql-mcp-server \"your_connection_urls_here\""
	@echo "Example: ./bin/postgresql-mcp-server \"postgresql://user:pass@localhost:5432/db1,postgresql://user:pass@localhost:5433/db2\""

deps:
	go mod tidy
	go mod download

.DEFAULT_GOAL := build
