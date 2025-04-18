#!/bin/bash

set -e

# Check if protoc is installed
if ! command -v protoc &> /dev/null; then
    echo "⚠️ Protocol Buffer compiler (protoc) not found!"
    echo "Please install it with:"
    echo "  apt update && apt install -y protobuf-compiler"
    echo "  go install google.golang.org/protobuf/cmd/protoc-gen-go@latest"
    echo "  go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest"
    exit 1
fi

echo "🔄 Generating Protocol Buffer code..."

# Make sure proto directory exists
mkdir -p proto

# Generate Go code from proto files
protoc --go_out=. --go_opt=paths=source_relative \
  --go-grpc_out=. --go-grpc_opt=paths=source_relative \
  proto/agent.proto

echo "✅ Protocol Buffer code generated successfully!"
echo "📝 Remember to update your imports to use the generated code." 