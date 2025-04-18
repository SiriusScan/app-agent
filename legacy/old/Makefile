.PHONY: tools proto test

# Install development tools locally
tools:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Generate protocol buffer code
proto:
	PATH="$(shell go env GOPATH)/bin:$(PATH)" protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/scan/v1/scan.proto

# Run tests in container
test:
	docker exec sirius-engine sh -c 'cd /app-agent && go test -v ./...'

# Run tests for a specific package in container
test-pkg:
	docker exec sirius-engine sh -c 'cd /app-agent && go test -v ./$(pkg)' 