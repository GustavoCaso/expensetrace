.PHONY: build test run_web lint generate-test-coverage clean help

.DEFAULT_GOAL := help

build:
	CGO_ENABLED=1 go build -o expensetrace ./cmd/

test:
	go test ./...

run_web:
	EXPENSE_LIVERELOAD=true go run cmd/main.go web

lint:
	golangci-lint run

format:
	golangci-lint fmt .

generate-test-coverage:
	@echo "Generating coverage report"
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	open coverage.html

clean:
	rm -f coverage.out coverage.html

help:
	@echo "Available targets:"
	@echo "  build                  - Build the CLI binary"
	@echo "  test                   - Run all tests"
	@echo "  run_web                - Run the web server using EXPENSE_LIVERELOAD=true set"
	@echo "  generate-test-coverage - Generate coverage report"
	@echo "  lint                   - Run golangci-lint"
	@echo "  format                 - Format Go code"
	@echo "  clean                  - Clean build artifacts"
	@echo "  help                   - Show this help message"
