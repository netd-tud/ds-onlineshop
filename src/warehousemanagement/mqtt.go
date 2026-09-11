package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	commonpb "github.com/netd-tud/ds-onlineshop/src/warehousemanagement/genproto/common"
	inventorypb "github.com/netd-tud/ds-onlineshop/src/warehousemanagement/genproto/inventory"
	warehousemanagementpb "github.com/netd-tud/ds-onlineshop/src/warehousemanagement/genproto/warehousemanagement"
	"google.golang.org/grpc/metadata"
)

// MqttMoney represents a monetary amount formatted for JSON payload serialization over MQTT topics.
type MqttMoney struct {
	CurrencyCode string `json:"currency_code"`
	Units        int64  `json:"units"`
	Nanos        int32  `json:"nanos"`
}

// MqttCreateProductPayload defines the JSON message structure received via MQTT to trigger product creation.
//
// It contains metadata, pricing details, category listings, initial stock allocations,
// and an authorization token required to process the creation request.
type MqttCreateProductPayload struct {
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	PriceUsd     MqttMoney `json:"price_usd"`
	Categories   []string  `json:"categories"`
	InitialStock int64     `json:"initial_stock"`
	Token        string    `json:"token"`
}

// MqttUpdateStockPayload defines the JSON message structure received via MQTT to update inventory stock levels.
//
// It specifies the target product ID, the relative quantity change (delta), and an authorization token.
type MqttUpdateStockPayload struct {
	ID    string `json:"id"`
	Delta int64  `json:"delta"`
	Token string `json:"token"`
}

// mqttMsgChan is a channel used to queue incoming MQTT messages
// for asynchronous consumption and processing by background workers.
var mqttMsgChan = make(chan mqtt.Message)

// messagePubHandler is the default MQTT callback function invoked when a message is published to subscribed topics.
//
// It forwards incoming MQTT messages directly to the internal mqttMsgChan channel for asynchronous processing.
var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	mqttMsgChan <- msg
}

// connectHandler is the callback function triggered upon successfully establishing a connection to the MQTT broker.
//
// It logs a message confirming that the MQTT client connection is active.
var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	fmt.Println("Connected to MQTT Broker")
}

// connectLostHandler is the callback function triggered upon losing connection to the MQTT broker.
//
// It logs a message indicating the connection loss and the associated error.
var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	fmt.Printf("Connection lost: %v", err)
}

// processMsg starts a background goroutine that reads MQTT messages from an input channel,
// logs their topic and payload, and pipes them to an output channel.
//
// It returns the output channel for further pipeline processing and automatically closes
// it when the input channel is closed or the context is canceled.
func processMsg(ctx context.Context, input <-chan mqtt.Message) chan mqtt.Message {
	out := make(chan mqtt.Message)
	go func() {
		defer close(out)
		for {
			select {
			case msg, ok := <-input:
				if !ok {
					return
				}
				fmt.Printf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())
				out <- msg
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}

// setupMqttSubscriber initializes and manages the MQTT client connection, topic subscriptions, and background message processing loop.
//
// It connects to the configured broker, subscribes to product creation and stock update topics,
// and routes incoming messages to internal gRPC handlers with context-propagated authorization tokens.
// It also listens for termination signals to perform clean unsubscriptions and graceful shutdowns.
func setupMqttSubscriber(svc *warehouseManagement) {
	createTopic := "inventory/create-item"
	updateTopic := "inventory/update-product-stock"

	opts := mqtt.NewClientOptions()
	opts.AddBroker(svc.mqttBrokerAddr)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	wg.Go(func() {
		finalChan := processMsg(ctx, mqttMsgChan)

		for msg := range finalChan {
			func(msg mqtt.Message) {
				reqCtx, reqCancel := context.WithTimeout(ctx, 5*time.Second)
				defer reqCancel()

				switch msg.Topic() {
				case createTopic:
					var payload MqttCreateProductPayload
					if err := json.Unmarshal(msg.Payload(), &payload); err != nil {
						log.Errorf("MQTT Worker: Failed to parse JSON creation payload: %v", err)
						return
					}

					grpcReq := &warehousemanagementpb.CreateWarehouseProductRequest{
						Name:        payload.Name,
						Description: payload.Description,
						PriceUsd: &commonpb.Money{
							CurrencyCode: payload.PriceUsd.CurrencyCode,
							Units:        payload.PriceUsd.Units,
							Nanos:        payload.PriceUsd.Nanos,
						},
						Categories:   payload.Categories,
						InitialStock: payload.InitialStock,
					}

					if payload.Token != "" {
						reqCtx = metadata.NewOutgoingContext(reqCtx, metadata.Pairs("authorization", "Bearer "+payload.Token))
					}
					resp, err := svc.CreateNewProduct(reqCtx, grpcReq)
					if err != nil {
						log.Errorf("MQTT Worker: CreateNewProduct execution failed: %v", err)
						return
					}
					log.Infof("MQTT Worker: Product successfully created via MQTT. Allocated ID: %s", resp.GetProduct().GetId())
				case updateTopic:
					var payload MqttUpdateStockPayload
					if err := json.Unmarshal(msg.Payload(), &payload); err != nil {
						log.Errorf("MQTT Worker: Failed to parse JSON stock update payload: %v", err)
						return
					}

					log.Infof("MQTT Worker: Processing stock update for item '%s' with delta %d", payload.ID, payload.Delta)

					grpcReq := &inventorypb.ChangeInventoryProductStockRequest{
						Id:    payload.ID,
						Delta: payload.Delta,
					}

					if payload.Token != "" {
						reqCtx = metadata.NewOutgoingContext(reqCtx, metadata.Pairs("authorization", "Bearer "+payload.Token))
					}
					resp, err := svc.UpdateProductStock(reqCtx, grpcReq)
					if err != nil {
						log.Errorf("MQTT Worker: UpdateProductStock execution failed: %v", err)
						return
					}
					log.Infof("MQTT Worker: Stock updated successfully via MQTT. Product ID: %s", resp.GetId())
				}
			}(msg)
		}
	})

	if token := client.Subscribe(createTopic, 1, messagePubHandler); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	fmt.Printf("Subscribed to topic: %s\n", createTopic)

	if token := client.Subscribe(updateTopic, 1, messagePubHandler); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	fmt.Printf("Subscribed to topic: %s\n", updateTopic)

	// Wait for interrupt signal to gracefully shut down the subscriber
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	// Cancel the context to signal the goroutine to stop
	cancel()

	// Unsubscribe and disconnect
	fmt.Println("Unsubscribing and disconnecting...")
	client.Unsubscribe(createTopic, updateTopic)
	client.Disconnect(250)

	// Wait for the goroutine to finish
	wg.Wait()
	fmt.Println("MQTT cleanup complete, exiting...")
}
