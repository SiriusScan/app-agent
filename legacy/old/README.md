# Sirius App Agent

The Sirius App Agent is a service that manages scan tasks and communicates with various Sirius components through RabbitMQ, Valkey store, and gRPC.

## Project Structure

```
.
├── cmd/                 # Main applications
│   └── app-agent/      # Application entry points
├── internal/           # Private application code
│   ├── agent/         # Core business logic
│   └── config/        # Configuration management
└── pkg/               # Public library code
    ├── queue/         # Queue integration
    └── store/         # Storage integration
```

## Dependencies

- Go 1.21 or later
- RabbitMQ client (github.com/rabbitmq/amqp091-go)
- gRPC (google.golang.org/grpc)
- Protocol Buffers (google.golang.org/protobuf)

## Configuration

The application uses environment variables for configuration:

- `SIRIUS_RABBITMQ` - RabbitMQ connection URL
- `SIRIUS_RABBITMQ_QUEUE` - Queue name (default: "tasks")
- `SIRIUS_RABBITMQ_EXCHANGE` - Exchange name (default: "sirius")
- `SIRIUS_RABBITMQ_SSL` - Enable SSL for RabbitMQ (default: false)
- `SIRIUS_VALKEY` - Valkey store address
- `SIRIUS_VALKEY_TIMEOUT` - Valkey operation timeout (default: 30s)
- `SIRIUS_VALKEY_INSECURE` - Allow insecure connection (default: false)
- `SCAN_SERVER_ADDR` - Scan server gRPC address
- `SCAN_SERVER_TIMEOUT` - Scan server operation timeout (default: 60s)
- `SCAN_SERVER_INSECURE` - Allow insecure connection (default: false)

## Development

1. Clone the repository
2. Install dependencies: `go mod tidy`
3. Build the application: `go build ./cmd/app-agent`
4. Run tests: `go test ./...`

## License

Copyright © 2024 Sirius Project. All rights reserved.
