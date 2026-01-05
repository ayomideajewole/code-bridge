.PHONY: build run clean test help dev

# Binary name
BINARY_NAME=code-bridge
MAIN_PATH=cmd/server/main.go

# Build the application
build:
	@echo "Building..."
	@go build -o $(BINARY_NAME) $(MAIN_PATH)

# Run the application
run: build
	@echo "Running..."
	@./$(BINARY_NAME)

# Run without building (useful for development)
dev:
	@echo "Running in dev mode..."
	@go run $(MAIN_PATH)

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -f $(BINARY_NAME)
	@go clean

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	@go mod download

# Tidy dependencies
tidy:
	@echo "Tidying dependencies..."
	@go mod tidy

# Install the application
install:
	@echo "Installing..."
	@go install $(MAIN_PATH)

# Help
help:
	@echo "Available commands:"
	@echo "  make build    - Build the application"
	@echo "  make run      - Build and run the application"
	@echo "  make dev      - Run the application without building"
	@echo "  make clean    - Remove build artifacts"
	@echo "  make test     - Run tests"
	@echo "  make deps     - Download dependencies"
	@echo "  make tidy     - Tidy dependencies"
	@echo "  make install  - Install the application"
