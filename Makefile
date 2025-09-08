.PHONY: build run clean test deps

# Application name
APP_NAME = domain-exporter

# Build application
build:
	go build -o $(APP_NAME) .

# Run application
run:
	go run . -config=config.yaml

# Install dependencies
deps:
	go mod tidy
	go mod download

# Clean build files
clean:
	rm -f $(APP_NAME)

# Run tests
test:
	go test ./...

# Format code
fmt:
	go fmt ./...

# Check code
vet:
	go vet ./...

# Full check
check: fmt vet test

# Build Docker image
docker-build:
	docker build -t $(APP_NAME) .

# Test application functionality
test-app:
	./test-app.sh

# Docker compose commands
docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

# Show help
help:
	@echo "Available commands:"
	@echo "  build       - Build application"
	@echo "  run         - Run application"
	@echo "  deps        - Install dependencies"
	@echo "  clean       - Clean build files"
	@echo "  test        - Run tests"
	@echo "  test-app    - Test application functionality"
	@echo "  fmt         - Format code"
	@echo "  vet         - Check code"
	@echo "  check       - Full check (fmt+vet+test)"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-up   - Start with Docker Compose"
	@echo "  docker-down - Stop Docker Compose"