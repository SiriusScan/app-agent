#!/bin/bash

# Generate Go code from protobuf definitions
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       proto/hello/hello.proto

echo "Proto files generated successfully" 