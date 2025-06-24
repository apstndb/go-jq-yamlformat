.PHONY: test
test:
	go test -v -race ./...

.PHONY: test-coverage
test-coverage:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

.PHONY: lint
lint:
	golangci-lint run

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: examples
examples:
	@echo "Running examples..."
	@echo "\n=== Basic Example ==="
	go run examples/basic/main.go
	@echo "\n=== Variables Example ==="
	go run examples/variables/main.go
	@echo "\n=== Custom Types Example ==="
	go run examples/custom-types/main.go
	@echo "\n=== Streaming Example ==="
	go run examples/streaming/main.go
	@echo "\n=== Errors Example ==="
	go run examples/errors/main.go

.PHONY: clean
clean:
	rm -f coverage.out coverage.html
	rm -f examples/streaming/output.jsonl

.PHONY: all
all: fmt vet test