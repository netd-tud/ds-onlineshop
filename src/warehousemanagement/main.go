package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/dtm-labs/client/workflow"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	shared "github.com/turt1z/microservices-demo/src/shared"
	pb "github.com/turt1z/microservices-demo/src/warehousemanagement/genproto"
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
)

type ServedFunction int

const (
	NAIVE ServedFunction = iota
	SAGA
	XA
)

type warehouseManagement struct {
	servedFunction ServedFunction

	productCatalogSvcAddr string
	productCatalogSvcConn *grpc.ClientConn

	inventorySvcAddr string
	inventorySvcConn *grpc.ClientConn

	xaProductCatalogConn *grpc.ClientConn
	xaInventoryConn      *grpc.ClientConn

	dtmSvcAddr string
	dtmSvcConn *grpc.ClientConn

	ownAddr string

	mqttBrokerAddr string

	pb.UnimplementedWarehouseManagementServer
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
	svc := new(warehouseManagement)

	srvPort := wrapperPort
	if os.Getenv("PORT") != "" {
		srvPort = os.Getenv("PORT")
	}

	shared.MustMapEnv(&svc.productCatalogSvcAddr, "PRODUCT_CATALOG_SERVICE_ADDR")
	shared.MustMapEnv(&svc.inventorySvcAddr, "INVENTORY_CATALOG_SERVICE_ADDR")
	shared.MustMapEnv(&svc.mqttBrokerAddr, "MQTT_BROKER_ADDR")
	shared.MustMapEnv(&svc.dtmSvcAddr, "DTM_SERVICE_ADDR")
	shared.MustMapEnv(&svc.ownAddr, "WAREHOUSE_MANAGEMENT_SVC_ADDR")

	sf := strings.ToUpper(os.Getenv("SERVED_FUNCTION"))
	switch sf {
	case "SAGA":
		log.Infof("Warehouse Management Service is running in SAGA mode")
		svc.servedFunction = SAGA
	case "XA":
		log.Infof("Warehouse Management Service is running in XA mode")
		svc.servedFunction = XA
	default:
		log.Infof("Warehouse Management Service is running in NAIVE mode")
		svc.servedFunction = NAIVE
	}

	ctx := context.Background()
	shared.MustConnGRPC(ctx, &svc.productCatalogSvcConn, svc.productCatalogSvcAddr)
	shared.MustConnGRPC(ctx, &svc.inventorySvcConn, svc.inventorySvcAddr)
	shared.MustConnGRPC(ctx, &svc.dtmSvcConn, svc.dtmSvcAddr)
	mustConnGRPC(ctx, &svc.xaProductCatalogConn, svc.productCatalogSvcAddr, grpc.WithUnaryInterceptor(workflow.Interceptor))
	mustConnGRPC(ctx, &svc.xaInventoryConn, svc.inventorySvcAddr, grpc.WithUnaryInterceptor(workflow.Interceptor))

	run(srvPort, svc)
	setupMqttServer(svc)
}

func run(port string, svc *warehouseManagement) string {
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

	pb.RegisterWarehouseManagementServer(srv, svc)
	healthcheck := health.NewServer()
	healthpb.RegisterHealthServer(srv, healthcheck)

	workflow.InitGrpc(svc.dtmSvcAddr, svc.ownAddr, srv)

	go srv.Serve(listener)

	if err := svc.registerXaCreateProductWorkflow(); err != nil {
		log.Fatal(errors.Wrap(err, "workflow: failed to register xa-create-product"))
	}
	return listener.Addr().String()
}

func mustConnGRPC(ctx context.Context, conn **grpc.ClientConn, addr string, extraOpts ...grpc.DialOption) {
	var err error
	_, cancel := context.WithTimeout(ctx, time.Second*3)
	defer cancel()
	opts := append([]grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	}, extraOpts...)
	*conn, err = grpc.NewClient(addr, opts...)
	if err != nil {
		panic(errors.Wrapf(err, "grpc: failed to connect %s", addr))
	}
}
