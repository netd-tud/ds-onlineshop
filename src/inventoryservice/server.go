package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/sirupsen/logrus"
	pb "github.com/turt1z/microservices-demo/src/inventoryservice/genproto"
	shared "github.com/turt1z/microservices-demo/src/shared"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
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

func run(port string) error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatal(err)
	}

	publicKeyPath := os.Getenv("AUTH_PUBLIC_KEY_PATH")
	if publicKeyPath == "" {
		publicKeyPath = "certs/auth_public.pem"
	}
	pubKey, err := os.ReadFile(publicKeyPath)
	if err != nil {
		log.Fatalf("failed to read public key from path %s: %v", publicKeyPath, err)
	}

	// Propagate trace context
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{}, propagation.Baggage{}))
	var srv *grpc.Server
	srv = grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.UnaryInterceptor(shared.NewAuthInterceptor(pubKey,
			// TODO: remove here once token validation implemented for each function
			"/hipstershop.InventoryService/ListInventory",
			"/hipstershop.InventoryService/GetInventoryProduct",
			//"/hipstershop.InventoryService/ChangeInventoryProductStock", already validated access right
			"/hipstershop.InventoryService/SetInventoryProductStock",
			"/hipstershop.InventoryService/CreateNewInventoryProduct",
			"/hipstershop.InventoryService/DeleteInventoryProduct",
			"/hipstershop.InventoryService/CompensateCreateNewInventoryProduct",
		)),
	)

	svc := &inventory{
		thresholds: struct {
			lowStock      int64
			criticalStock int64
		}{lowStock: 10, criticalStock: 3},
	}

	shared.MustMapEnv(&svc.productCatalogSvcAddr, "PRODUCT_CATALOG_SERVICE_ADDR")
	shared.MustMapEnv(&svc.mqttBrokerAddr, "MQTT_BROKER_ADDR")

	ctx := context.Background()
	shared.MustConnGRPC(ctx, &svc.productCatalogSvcConn, svc.productCatalogSvcAddr)

	err = loadInventory(&svc.inventory)
	if err != nil {
		log.Fatalf("could not parse inventory: %v", err)
	}

	opts := mqtt.NewClientOptions().AddBroker(svc.mqttBrokerAddr)
	opts.SetClientID("inventory-service")
	opts.SetConnectTimeout(time.Second * 5)

	svc.mqttClient = mqtt.NewClient(opts)
	if token := svc.mqttClient.Connect(); token.Wait() && token.Error() != nil {
		return status.Errorf(codes.Internal, "MQTT: Connection failed: %v", token.Error())
	}

	log.Debugln("MQTT: Connected successfully to broker")

	pb.RegisterInventoryServiceServer(srv, svc)
	healthcheck := health.NewServer()
	healthpb.RegisterHealthServer(srv, healthcheck)
	go srv.Serve(listener)

	return nil
}
