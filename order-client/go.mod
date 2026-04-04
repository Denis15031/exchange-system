module exchange-system/order-client

go 1.25.1

require (
	exchange-system/proto v0.0.0-00010101000000-000000000000
	exchange-system/shared v0.0.0-00010101000000-000000000000
	github.com/spf13/cobra v1.8.1
	go.uber.org/zap v1.27.1
	google.golang.org/grpc v1.79.3
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/net v0.52.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.35.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251202230838-ff82c1b0f217 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace (
	exchange-system/proto => ../proto
	exchange-system/shared => ../shared
)
