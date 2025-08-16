.PHONY: build clean install test run deps

build:
	go build -o bin .

install: build
	@echo "Built bin binary"

clean:
	rm -f bin

test:
	go test ./...

run: build
	./bin

deps:
	go mod tidy
	go mod download

.DEFAULT_GOAL := build
