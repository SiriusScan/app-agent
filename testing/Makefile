.PHONY: build-linux shell quick test-template clean help

# Default target
help:
	@echo "Template System Development Environment"
	@echo ""
	@echo "Available targets:"
	@echo "  build-linux    - Cross-compile agent for Linux"
	@echo "  shell          - Start interactive container shell"
	@echo "  quick          - Build + quick test"
	@echo "  test-template  - Run specific template (usage: make test-template TEMPLATE=file.yaml)"
	@echo "  clean          - Remove build artifacts"
	@echo ""

# Cross-compile agent for Linux
build-linux:
	@echo "Building agent for Linux..."
	@cd .. && GOOS=linux GOARCH=amd64 go build -o bin/agent cmd/agent/main.go
	@echo "✓ Binary created: bin/agent"

# Start interactive container shell
shell: build-linux
	@echo "Starting container..."
	@docker-compose -f docker-compose.dev.yaml run --rm agent-test

# Quick test (build + run version check)
quick: build-linux
	@echo "Quick test: checking agent version..."
	@docker-compose -f docker-compose.dev.yaml run --rm agent-test /app/bin/agent --version || echo "⚠ Agent not yet implementing --version flag"

# Run specific template
test-template: build-linux
	@if [ -z "$(TEMPLATE)" ]; then \
		echo "Usage: make test-template TEMPLATE=test-templates/01-file-hash.yaml"; \
		exit 1; \
	fi
	@echo "Running template: $(TEMPLATE)"
	@docker-compose -f docker-compose.dev.yaml run --rm agent-test /app/bin/agent template run /app/$(TEMPLATE)

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -f ../bin/agent
	@echo "✓ Clean complete"

