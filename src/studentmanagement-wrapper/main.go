package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	pb "github.com/turt1z/microservices-demo/src/studentmanagement/genproto"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

const (
	wrapperPort = "50000"

	defaultInventoryAddr      = "inventoryservice:50002"
	defaultProductCatalogAddr = "productcatalogservice:3550"
)

type productManagement struct {
	productCatalogSvcAddr string
	productCatalogSvcConn *grpc.ClientConn

	inventorySvcAddr string
	inventorySvcConn *grpc.ClientConn

	pb.UnimplementedProductManagementServer
}

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
	svc := new(productManagement)

	srvPort := wrapperPort
	if os.Getenv("PORT") != "" {
		srvPort = os.Getenv("PORT")
	}

	mustMapEnv(&svc.productCatalogSvcAddr, "PRODUCT_CATALOG_SERVICE_ADDR")
	mustMapEnv(&svc.inventorySvcAddr, "INVENTORY_CATALOG_SERVICE_ADDR")

	ctx := context.Background()
	mustConnGRPC(ctx, &svc.productCatalogSvcConn, svc.productCatalogSvcAddr)
	mustConnGRPC(ctx, &svc.inventorySvcConn, svc.inventorySvcAddr)

	run(srvPort, svc)
	select {}
}

func run(port string, svc *productManagement) string {
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

	pb.RegisterProductManagementServer(srv, svc)
	healthcheck := health.NewServer()
	healthpb.RegisterHealthServer(srv, healthcheck)
	go srv.Serve(listener)

	return listener.Addr().String()
}

func mustMapEnv(target *string, envKey string) {
	v := os.Getenv(envKey)
	if v == "" {
		panic(fmt.Sprintf("environment variable %q not set", envKey))
	}
	*target = v
}

func mustConnGRPC(ctx context.Context, conn **grpc.ClientConn, addr string) {
	var err error
	_, cancel := context.WithTimeout(ctx, time.Second*3)
	defer cancel()
	*conn, err = grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()))
	if err != nil {
		panic(errors.Wrapf(err, "grpc: failed to connect %s", addr))
	}
}
