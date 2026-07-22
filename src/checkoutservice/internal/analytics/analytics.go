package analytics

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

type EventType string

const (
	EventView    EventType = "VIEW"
	EventATC     EventType = "ATC"
	EventOrder   EventType = "ORDER"
	EventReorder EventType = "REORDER"
)

type ProductEvent struct {
	EventID   string    `json:"event_id"`
	EventTime time.Time `json:"event_time"`
	EventType EventType `json:"event_type"`
	SKU       string    `json:"sku"`
	Qty       int32     `json:"qty,omitempty"`
	Price     float64   `json:"price,omitempty"`
	OrderID   string    `json:"order_id,omitempty"`
	SessionID string    `json:"session_id"`
	Producer  string    `json:"producer"`
}

type Publisher struct {
	writer   *kafka.Writer
	producer string
}

func NewPublisher(producerName string) *Publisher {
	brokers := os.Getenv("KAFKA_BOOTSTRAP_SERVERS")
	if brokers == "" {
		brokers = "analytics-kafka-kafka-bootstrap.kafka.svc.cluster.local:9092"
	}
	topic := os.Getenv("KAFKA_PRODUCT_EVENTS_TOPIC")
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

func (p *Publisher) Publish(evt ProductEvent) {
	evt.EventID = uuid.NewString()
	evt.EventTime = time.Now().UTC()
	evt.Producer = p.producer

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		payload, err := json.Marshal(evt)
		if err != nil {
			log.Printf("analytics: marshal failed: %v", err)
			return
		}
		if err := p.writer.WriteMessages(ctx, kafka.Message{
			Key:   []byte(evt.SKU),
			Value: payload,
		}); err != nil {
			log.Printf("analytics: publish %s for %s failed: %v", evt.EventType, evt.SKU, err)
		}
	}()
}

func (p *Publisher) Close() error {
	return p.writer.Close()
}
