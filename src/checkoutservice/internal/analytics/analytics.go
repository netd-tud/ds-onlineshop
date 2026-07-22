package analytics

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	pb "github.com/GoogleCloudPlatform/microservices-demo/src/checkoutservice/genproto"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

type ProductEventType string
type OrderEventType string

const (
	EventOrder ProductEventType = "ORDER"

	EventCreate   OrderEventType = "CREATE"
	EventComplete OrderEventType = "COMPLETE"
)

type ProductEvent struct {
	EventID   string           `json:"event_id"`
	EventTime time.Time        `json:"event_time"`
	EventType ProductEventType `json:"event_type"`
	SKU       string           `json:"sku"`
	Qty       int32            `json:"qty,omitempty"`
	Price     pb.Money         `json:"price"`
	OrderID   string           `json:"order_id,omitempty"`
	SessionID string           `json:"session_id"`
	Producer  string           `json:"producer"`
}

type OrderEvent struct {
	EventID   string         `json:"event_id"`
	EventTime time.Time      `json:"event_time"`
	EventType OrderEventType `json:"event_type"`
	Price     pb.Money       `json:"price"`
	OrderID   string         `json:"order_id,omitempty"`
	SessionID string         `json:"session_id"`
	Producer  string         `json:"producer"`
}

type Publisher struct {
	writer   *kafka.Writer
	producer string
}

func NewPublisher(producerName string, topic string) *Publisher {
	brokers := os.Getenv("KAFKA_BOOTSTRAP_SERVERS")
	if brokers == "" {
		brokers = "analytics-kafka-kafka-bootstrap.kafka.svc.cluster.local:9092"
	}
	if topic == "" {
		topic = "product-events"
	}

	return &Publisher{
		producer: producerName,
		writer: &kafka.Writer{
			Addr:         kafka.TCP(brokers),
			Topic:        topic,
			Balancer:     &kafka.Hash{},
			BatchTimeout: 50 * time.Millisecond,
			RequiredAcks: kafka.RequireOne,
		},
	}
}

func (p *Publisher) Publish(evt any) {
	var payload []byte
	var partitionKey string
	var eventTypeStr string

	eventID := uuid.NewString()
	eventTime := time.Now().UTC()

	switch v := evt.(type) {
	case ProductEvent:
		v.EventID = eventID
		v.EventTime = eventTime
		v.Producer = p.producer
		partitionKey = v.SKU
		eventTypeStr = string(v.EventType)

		var err error
		payload, err = json.Marshal(v)
		if err != nil {
			log.Printf("analytics: marshal failed for ProductEvent: %v", err)
			return
		}

	case OrderEvent:
		v.EventID = eventID
		v.EventTime = eventTime
		v.Producer = p.producer
		partitionKey = v.OrderID
		eventTypeStr = string(v.EventType)

		var err error
		payload, err = json.Marshal(v)
		if err != nil {
			log.Printf("analytics: marshal failed for OrderEvent: %v", err)
			return
		}

	default:
		log.Printf("analytics: unsupported event type passed to Publish: %T", evt)
		return
	}

	go func(key string, data []byte, eType string) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		if err := p.writer.WriteMessages(ctx, kafka.Message{
			Key:   []byte(key),
			Value: data,
		}); err != nil {
			log.Printf("analytics: publish %s failed: %v", eType, err)
		}
	}(partitionKey, payload, eventTypeStr)
}

func (p *Publisher) Close() error {
	return p.writer.Close()
}
