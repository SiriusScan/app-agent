module github.com/SiriusScan/app-agent

go 1.23.0

toolchain go1.24.1

require (
	github.com/SiriusScan/go-api v0.0.10
	github.com/rabbitmq/amqp091-go v1.10.0
	go.uber.org/zap v1.27.0
	google.golang.org/grpc v1.71.1
	google.golang.org/protobuf v1.36.6
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/cobra v1.10.1 // indirect
	github.com/spf13/pflag v1.0.9 // indirect
	github.com/streadway/amqp v1.1.0 // indirect
	github.com/valkey-io/valkey-go v1.0.60 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/net v0.39.0 // indirect
	golang.org/x/sys v0.32.0 // indirect
	golang.org/x/text v0.24.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250414145226-207652e42e2e // indirect
)

// replace github.com/SiriusScan/go-api => ../go-api
