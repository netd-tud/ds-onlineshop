package main

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	pb "github.com/turt1z/microservices-demo/src/inventoryservice/genproto"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

const defaultPort = "50002"

var log *logrus.Logger

func init() {
	log = logrus.New()
	log.Level = logrus.DebugLevel
	log.Formatter = &logrus.JSONFormatter{
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "severity",
			logrus.FieldKeyMsg:   "message",
		},
		TimestampFormat: time.RFC3339Nano,
	}
	log.Out = os.Stdout
}

func main() {
	port := defaultPort
	if value, ok := os.LookupEnv("PORT"); ok {
		port = value
	}

	run(port)
	select {}
}

func run(port string) string {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatal(err)
	}

	// Propagate trace context
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{}, propagation.Baggage{}))
	var srv *grpc.Server
	srv = grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()))

	svc := &inventory{}
	err = loadInventory(&svc.inventory)
	if err != nil {
		log.Fatalf("could not parse inventory: %v", err)
	}

	pb.RegisterInventoryServiceServer(srv, svc)
	healthcheck := health.NewServer()
	healthpb.RegisterHealthServer(srv, healthcheck)
	go srv.Serve(listener)

	return listener.Addr().String()
}
