#!/bin/bash

set -e

echo "Generating protocol buffer code..."

# Create output directory if it doesn't exist
mkdir -p proto/scan

# Generate Go code from protocol buffers
protoc --go_out=. \
       --go_opt=paths=source_relative \
       --go-grpc_out=. \
       --go-grpc_opt=paths=source_relative \
       proto/scan/agent.proto

echo "Protocol buffer code generation complete." 