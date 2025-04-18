#!/bin/bash

# Exit on error
set -e

# Install protoc plugins if not present
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.31.0
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0

# Create output directory if it doesn't exist
mkdir -p internal/proto/scan

# Clean up any existing generated files
rm -f internal/proto/scan/*.pb.go

# Generate Go code from proto files
cd internal/proto && \
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    scan/agent.proto

# Fix permissions
chmod -R 644 scan/*.pb.go

echo "✅ Proto generation complete" 