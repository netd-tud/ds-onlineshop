package main

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	notificationpb "github.com/netd-tud/ds-onlineshop/src/notificationservice/genproto/notification"
	shared "github.com/netd-tud/ds-onlineshop/src/shared"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

const defaultPort = "50051"

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

	var srv *grpc.Server
	srv = grpc.NewServer()

	svc := &notification{
		thresholds: struct {
			lowStock      int64
			criticalStock int64
		}{lowStock: 10, criticalStock: 3},

		alerts: make(map[string]*notificationpb.StockAlert),

		orderQueues:   make(map[string]*OrderQueue),
		queueCapacity: 20,

		subs: make(map[chan *notificationpb.StockAlert]map[string]struct{}),
	}

	shared.MustMapEnv(&svc.mqttBrokerAddr, "MQTT_BROKER_ADDR")
	if value, ok := os.LookupEnv("QUEUE_CAPACITY"); ok {
		if capacity, err := strconv.Atoi(value); err == nil {
			svc.queueCapacity = capacity
		}
	}

	opts := mqtt.NewClientOptions().AddBroker(svc.mqttBrokerAddr)
	opts.SetClientID("inventory-service")
	opts.SetConnectTimeout(time.Second * 5)

	svc.setupMQTTSubscriber()

	notificationpb.RegisterNotificationServiceServer(srv, svc)
	healthcheck := health.NewServer()
	healthpb.RegisterHealthServer(srv, healthcheck)
	go srv.Serve(listener)

	return nil
}
