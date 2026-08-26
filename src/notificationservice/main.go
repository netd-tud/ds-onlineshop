package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"slices"
	"strconv"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	checkoutpb "github.com/netd-tud/ds-onlineshop/src/notificationservice/genproto/checkout"
	notificationpb "github.com/netd-tud/ds-onlineshop/src/notificationservice/genproto/notification"
	shared "github.com/netd-tud/ds-onlineshop/src/shared"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/protobuf/encoding/protojson"
)

type notification struct {
	notificationpb.UnimplementedNotificationServiceServer

	mqttBrokerAddr string
	mqttClient     mqtt.Client

	thresholds struct {
		lowStock      int64
		criticalStock int64
	}

	alertsMu sync.RWMutex
	alerts   map[string]*notificationpb.StockAlert

	ordersMu      sync.RWMutex
	orderQueues   map[string]*OrderQueue
	queueCapacity int
}

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

func (n *notification) setupMQTTSubscriber() {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(n.mqttBrokerAddr)

	hostname, _ := os.Hostname()
	opts.SetClientID("notification-service-client-" + hostname)

	stockTopic := "inventory/+/+/stock"
	orderTopic := "+/checkout/orders/completed"
	qos := byte(1)

	var stockMessageHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
		n.onStockUpdate(client, msg)
	}

	var orderMessageHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
		n.onOrderCompleted(client, msg)
	}

	opts.OnConnect = func(client mqtt.Client) {
		log.Println("MQTT connected successfully")

		if token := client.Subscribe(stockTopic, qos, stockMessageHandler); token.Wait() && token.Error() != nil {
			log.Errorf("Error subscribing to topic %s: %v", stockTopic, token.Error())
		} else {
			log.Printf("Successfully subscribed to %s", stockTopic)
		}

		if token := client.Subscribe(orderTopic, qos, orderMessageHandler); token.Wait() && token.Error() != nil {
			log.Errorf("Error subscribing to topic %s: %v", orderTopic, token.Error())
		} else {
			log.Printf("Successfully subscribed to %s", orderTopic)
		}
	}

	opts.OnConnectionLost = func(client mqtt.Client, err error) {
		log.Printf("MQTT connection lost: %v", err)
	}

	client := mqtt.NewClient(opts)
	n.mqttClient = client

	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("Error connecting to MQTT broker: %v", token.Error())
	}

	log.Printf("Successfully subscribed to %s", stockTopic)
	log.Printf("Successfully subscribed to %s", orderTopic)
}

func (n *notification) onStockUpdate(_ mqtt.Client, msg mqtt.Message) {
	var p struct {
		Id         string   `json:"id"`
		Stock      int64    `json:"stock"`
		Severity   string   `json:"severity"`
		Categories []string `json:"categories"`
	}
	if err := json.Unmarshal(msg.Payload(), &p); err != nil {
		log.WithError(err).Error("Failed to unmarshal MQTT message")
		return
	}

	n.alertsMu.Lock()
	defer n.alertsMu.Unlock()
	if p.Severity == "normal" {
		delete(n.alerts, p.Id)
		return
	}

	n.alerts[p.Id] = &notificationpb.StockAlert{ProductId: p.Id, Category: p.Categories, Stock: p.Stock, Severity: p.Severity}
}

func (n *notification) Check(ctx context.Context, req *healthpb.HealthCheckRequest) (*healthpb.HealthCheckResponse, error) {
	return &healthpb.HealthCheckResponse{Status: healthpb.HealthCheckResponse_SERVING}, nil
}

func (n *notification) ListOpenAlerts(ctx context.Context, req *notificationpb.ListOpenAlertsRequest) (*notificationpb.ListOpenAlertsResponse, error) {
	reqCats := make(map[string]bool, len(req.GetCategories()))
	for _, c := range req.GetCategories() {
		reqCats[c] = true
	}

	n.alertsMu.RLock()
	defer n.alertsMu.RUnlock()

	response := &notificationpb.ListOpenAlertsResponse{}

	for _, alert := range n.alerts {
		for _, cat := range alert.Category {
			if reqCats[cat] || slices.Contains(req.GetCategories(), "all") {
				response.Alerts = append(response.Alerts, alert)
				break
			}
		}
	}

	return response, nil
}

func (n *notification) onOrderCompleted(_ mqtt.Client, msg mqtt.Message) {
	order := &checkoutpb.OrderResult{}

	if err := protojson.Unmarshal(msg.Payload(), order); err != nil {
		log.WithError(err).Error("Failed to unmarshal order MQTT message")
		return
	}

	n.PushNewOrder(order)
}

func (n *notification) PushNewOrder(order *checkoutpb.OrderResult) {
	currency := order.GetShippingCost().GetCurrencyCode()
	if currency == "" {
		currency = "UNKNOWN"
	}

	n.ordersMu.Lock()
	defer n.ordersMu.Unlock()

	q, exists := n.orderQueues[currency]
	if !exists {
		q = NewOrderQueue(n.queueCapacity)
		n.orderQueues[currency] = q
	}

	q.Push(order)
}

func (n *notification) ListRecentOrdersByCurrency(ctx context.Context, req *notificationpb.ListRecentOrdersByCurrencyRequest) (*notificationpb.ListRecentOrdersByCurrencyResponse, error) {
	n.ordersMu.RLock()
	defer n.ordersMu.RUnlock()

	response := &notificationpb.ListRecentOrdersByCurrencyResponse{}
	response.OrdersByCurrency = make(map[string]*notificationpb.OrderList)
	for _, currency := range req.GetCurrencies() {
		if q, exists := n.orderQueues[currency]; exists {
			response.OrdersByCurrency[currency] = &notificationpb.OrderList{Orders: q.GetAll()}
		}
	}

	return response, nil
}
