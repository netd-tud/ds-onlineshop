package main

import (
	"context"
	"encoding/json"
	"os"
	"slices"
	"sync"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	checkoutpb "github.com/netd-tud/ds-onlineshop/src/notificationservice/genproto/checkout"
	notificationpb "github.com/netd-tud/ds-onlineshop/src/notificationservice/genproto/notification"
	"google.golang.org/grpc"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
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

	channelMu sync.RWMutex
	subs      map[chan *notificationpb.StockAlert]map[string]struct{}
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

func (n *notification) setupMQTTSubscriber() {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(n.mqttBrokerAddr)

	hostname, _ := os.Hostname()
	opts.SetClientID("notification-service-client-" + hostname)

	stockTopic := "inventory/+/+/stock"
	orderTopic := "+/checkout/orders/completed/+"
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

	var createdAt *timestamppb.Timestamp

	if a, ok := n.alerts[p.Id]; ok {
		createdAt = a.CreatedAt
	} else {
		createdAt = timestamppb.Now()
	}

	alert := &notificationpb.StockAlert{
		ProductId: p.Id,
		Category:  p.Categories,
		Stock:     p.Stock,
		Severity:  p.Severity,
		CreatedAt: createdAt,
	}

	n.alerts[p.Id] = alert

	n.triggerNewAlert(alert)
}

func (n *notification) triggerNewAlert(alert *notificationpb.StockAlert) {
	n.channelMu.RLock()
	defer n.channelMu.RUnlock()

	for clientChan, categorySet := range n.subs {
		sendAlert := false
		for _, cat := range alert.GetCategory() {
			_, sendAlert = categorySet[cat]
		}

		if sendAlert || len(categorySet) == 0 {
			select {
			case clientChan <- alert:
			default:
				log.Printf("Client buffer full, dropping alert for %s", alert.GetProductId())
			}
		}
	}
}

func (n *notification) StreamStockAlerts(req *notificationpb.StreamStockAlertsRequest, stream grpc.ServerStreamingServer[notificationpb.StockAlert]) error {
	clientChan := make(chan *notificationpb.StockAlert, 100)

	categorySet := make(map[string]struct{})
	for _, c := range req.GetCategories() {
		categorySet[c] = struct{}{}
	}
	log.Info("Client subscribed to categories: ", req.GetCategories())

	n.channelMu.Lock()
	n.subs[clientChan] = categorySet
	n.channelMu.Unlock()

	defer func() {
		n.channelMu.Lock()
		delete(n.subs, clientChan)
		n.channelMu.Unlock()
	}()

	for {
		select {
		case <-stream.Context().Done():
			return nil
		case alert := <-clientChan:
			if err := stream.Send(alert); err != nil {
				return err
			}
		}
	}

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
