.PHONY: build test lint clean install

# Binary name
BINARY_NAME=srs

# Build the CLI binary
build:
	@echo "Building $(BINARY_NAME)..."
	@go build -o $(BINARY_NAME) .

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Run golangci-lint
lint:
	@echo "Running linter..."
	@golangci-lint run

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -f $(BINARY_NAME)
	@go clean

# Install binary to GOPATH/bin
install:
	@echo "Installing $(BINARY_NAME)..."
	@go install .

