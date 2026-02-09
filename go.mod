module github.com/SiriusScan/app-agent

go 1.24.0

toolchain go1.24.1

require (
	github.com/SiriusScan/go-api v0.0.10
	github.com/olekukonko/tablewriter v0.0.5
	github.com/spf13/cobra v1.10.1
	github.com/valkey-io/valkey-go v1.0.60
	go.uber.org/zap v1.27.0
	golang.org/x/term v0.37.0
	google.golang.org/grpc v1.71.1
	google.golang.org/protobuf v1.36.6
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/mattn/go-runewidth v0.0.9 // indirect
	github.com/spf13/pflag v1.0.9 // indirect
	github.com/streadway/amqp v1.1.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/net v0.39.0 // indirect
	golang.org/x/sys v0.38.0 // indirect
	golang.org/x/text v0.24.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250414145226-207652e42e2e // indirect
	gorm.io/gorm v1.25.12 // indirect
)

replace github.com/SiriusScan/go-api => ../go-api
