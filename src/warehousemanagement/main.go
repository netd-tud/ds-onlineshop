package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/dtm-labs/client/workflow"
	shared "github.com/netd-tud/ds-onlineshop/src/shared"
	warehousemanagementpb "github.com/netd-tud/ds-onlineshop/src/warehousemanagement/genproto/warehousemanagement"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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

// ServedFunction defines the transactional execution mode for the service.
type ServedFunction int

const (
	// NAIVE executes requests standardly without distributed transaction coordination and manual rollback.
	NAIVE ServedFunction = iota
	// SAGA executes requests using Saga-based distributed transaction management.
	SAGA
	// XA executes requests using 2-phase commit (XA) distributed transactions via DTM.
	XA
)

// warehouseManagement implements the gRPC WarehouseManagement server.
//
// It maintains client connections and address configurations for external microservices,
// including product catalog, inventory, DTM (Distributed Transaction Manager), and MQTT messaging infrastructure.
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

	warehousemanagementpb.UnimplementedWarehouseManagementServer
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

// main initializes and boots the warehouse management microservice.
//
// It parses required environment variables for service addresses, determines the distributed
// transaction mode (SAGA, XA, or NAIVE), establishes gRPC connections to dependencies, starts
// the server on the designated port, and sets up the MQTT subscriber.
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
	setupMqttSubscriber(svc)
}

// run configures and starts the gRPC server for the warehouse management service.
//
// It creates a TCP listener, configures OpenTelemetry tracing, registers the service and
// health check handlers, initializes DTM workflow support, begins serving requests, and registers
// the XA product creation workflow before returning the listener address.
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

	warehousemanagementpb.RegisterWarehouseManagementServer(srv, svc)
	healthcheck := health.NewServer()
	healthpb.RegisterHealthServer(srv, healthcheck)

	workflow.InitGrpc(svc.dtmSvcAddr, svc.ownAddr, srv)

	go srv.Serve(listener)

	if err := svc.registerXaCreateProductWorkflow(); err != nil {
		log.Fatal(errors.Wrap(err, "workflow: failed to register xa-create-product"))
	}
	return listener.Addr().String()
}

// mustConnGRPC establishes a blocking gRPC client connection to the specified address with default insecure
// credentials and OpenTelemetry telemetry handling.
//
// It accepts optional extraOpts (e.g., custom unary interceptors like workflow.Interceptor) and panics if
// connection initialization fails within the context timeout.
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
