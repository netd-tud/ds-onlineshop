module github.com/turt1z/microservices-demo/src/inventoryservice

go 1.26.4

require (
	github.com/eclipse/paho.mqtt.golang v1.5.1
	github.com/golang/protobuf v1.5.4
	github.com/sirupsen/logrus v1.9.4
	github.com/turt1z/microservices-demo/src/shared v0.0.0-20260715091515-425d812108a6
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.69.0
	go.opentelemetry.io/otel v1.44.0
	google.golang.org/grpc v1.82.0
	google.golang.org/protobuf v1.36.11
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/golang-jwt/jwt/v5 v5.3.1 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel/metric v1.44.0 // indirect
	go.opentelemetry.io/otel/trace v1.44.0 // indirect
	golang.org/x/net v0.55.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
	golang.org/x/text v0.37.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260526163538-3dc84a4a5aaa // indirect
)

replace github.com/turt1z/microservices-demo/src/inventoryservice/genproto => ./genproto
