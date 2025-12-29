.PHONY: build run clean deps test help run-fast run-no-smtp run-verbose

# Binary name
BINARY=email-checker

# Data directory
DATA_DIR=data

# Default input/output files
INPUT_FILE=$(DATA_DIR)/data.json
OUTPUT_FILE=$(DATA_DIR)/invalid_emails.json

# Default settings
WORKERS=16
BATCH_SIZE=1000
RATE_LIMIT=10ms

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

deps: ## Download dependencies
	go mod tidy

build: deps ## Build the binary (optimized)
	go build -ldflags="-s -w" -o $(BINARY) main.go

run: deps ## Run with default settings
	go run main.go -input=$(INPUT_FILE) -output=$(OUTPUT_FILE) -workers=$(WORKERS) -batch=$(BATCH_SIZE) -rate=$(RATE_LIMIT)

run-fast: deps ## Run with maximum speed (no rate limiting)
	go run main.go -input=$(INPUT_FILE) -output=$(OUTPUT_FILE) -workers=32 -batch=5000 -rate=0

run-no-smtp: deps ## Run without SMTP verification (faster)
	go run main.go -input=$(INPUT_FILE) -output=$(OUTPUT_FILE) -workers=$(WORKERS) -smtp=false

run-verbose: deps ## Run with verbose logging
	go run main.go -input=$(INPUT_FILE) -output=$(OUTPUT_FILE) -workers=$(WORKERS) -verbose

run-build: build ## Run the compiled binary
	./$(BINARY) -input=$(INPUT_FILE) -output=$(OUTPUT_FILE) -workers=$(WORKERS)

clean: ## Remove binary and output files
	rm -f $(BINARY)
	rm -f $(OUTPUT_FILE)

init: ## Create data directory if it doesn't exist
	mkdir -p $(DATA_DIR)

test: ## Run tests
	go test -v ./...

bench: ## Run benchmarks
	go test -bench=. -benchmem ./...

lint: ## Run linter (requires golangci-lint)
	golangci-lint run

fmt: ## Format code
	go fmt ./...

vet: ## Run go vet
	go vet ./...
