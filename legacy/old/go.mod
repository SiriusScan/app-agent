module github.com/SiriusScan/app-agent

go 1.24.1

replace github.com/SiriusScan/go-api => ../go-api

require (
	github.com/SiriusScan/go-api v0.0.0
	github.com/joho/godotenv v1.5.1
	github.com/prometheus/client_golang v1.19.0
	github.com/stretchr/testify v1.8.4
	go.uber.org/zap v1.27.0
	golang.org/x/time v0.11.0
	google.golang.org/grpc v1.65.0
	google.golang.org/grpc/cmd/protoc-gen-go-grpc v1.5.1
	google.golang.org/protobuf v1.34.1
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_model v0.5.0 // indirect
	github.com/prometheus/common v0.48.0 // indirect
	github.com/prometheus/procfs v0.12.0 // indirect
	github.com/streadway/amqp v1.1.0 // indirect
	github.com/stretchr/objx v0.5.0 // indirect
	github.com/valkey-io/valkey-go v1.0.54 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/net v0.34.0 // indirect
	golang.org/x/sys v0.29.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240604185151-ef581f913117 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
