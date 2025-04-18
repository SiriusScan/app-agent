.PHONY: all server agent clean proto run-server run-agent

all: server agent

# Build the server
server:
	@echo "Building server..."
	@mkdir -p bin
	@go build -o bin/agent-server ./cmd/agent-server
	@echo "Server built successfully!"

# Build the agent
agent:
	@echo "Building agent..."
	@mkdir -p bin
	@go build -o bin/agent ./cmd/agent
	@echo "Agent built successfully!"

# Generate protobuf code
proto:
	@echo "Generating protocol buffer code..."
	@mkdir -p proto/scan
	@./scripts/generate-protos.sh
	@echo "Protocol buffer code generated successfully!"

# Run the server
run-server: server
	@echo "Running server..."
	@bin/agent-server

# Run the agent
run-agent: agent
	@echo "Running agent..."
	@bin/agent

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@echo "Clean complete!"

# Run tests
test:
	@echo "Running tests..."
	@go test ./...
	@echo "Tests complete!"

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod download
	@go get -u google.golang.org/protobuf/cmd/protoc-gen-go
	@go get -u google.golang.org/grpc/cmd/protoc-gen-go-grpc
	@echo "Dependencies installed!"

# Show help
help:
	@echo "Available targets:"
	@echo "  make server     - Build the server"
	@echo "  make agent      - Build the agent"
	@echo "  make proto      - Generate protocol buffer code"
	@echo "  make run-server - Build and run the server"
	@echo "  make run-agent  - Build and run the agent"
	@echo "  make clean      - Clean build artifacts"
	@echo "  make test       - Run tests"
	@echo "  make deps       - Install dependencies"
	@echo "  make all        - Build both server and agent"
	@echo "  make help       - Show this help message" 