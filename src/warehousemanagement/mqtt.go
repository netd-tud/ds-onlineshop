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
	pb "github.com/turt1z/microservices-demo/src/warehousemanagement/genproto"
)

// MQTT Structural Mappings
type MqttMoney struct {
	CurrencyCode string `json:"currency_code"`
	Units        int64  `json:"units"`
	Nanos        int32  `json:"nanos"`
}

type MqttCreateProductPayload struct {
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	PriceUsd     MqttMoney `json:"price_usd"`
	Categories   []string  `json:"categories"`
	InitialStock int64     `json:"initial_stock"`
}

type MqttUpdateStockPayload struct {
	ID    string `json:"id"`
	Delta int64  `json:"delta"`
}

var mqttMsgChan = make(chan mqtt.Message)

var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	mqttMsgChan <- msg
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	fmt.Println("Connected to MQTT Broker")
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	fmt.Printf("Connection lost: %v", err)
}

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

func setupMqttServer(svc *warehouseManagement) {
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
	wg.Add(1)

	go func() {
		defer wg.Done()
		finalChan := processMsg(ctx, mqttMsgChan)

		for msg := range finalChan {
			reqCtx, reqCancel := context.WithTimeout(ctx, 5*time.Second)

			switch msg.Topic() {
			case createTopic:
				var payload MqttCreateProductPayload
				if err := json.Unmarshal(msg.Payload(), &payload); err != nil {
					log.Errorf("MQTT Worker: Failed to parse JSON creation payload: %v", err)
					reqCancel()
					continue
				}

				grpcReq := &pb.CreateWarehouseProductRequest{
					Name:        payload.Name,
					Description: payload.Description,
					PriceUsd: &pb.Money{
						CurrencyCode: payload.PriceUsd.CurrencyCode,
						Units:        payload.PriceUsd.Units,
						Nanos:        payload.PriceUsd.Nanos,
					},
					Categories:   payload.Categories,
					InitialStock: payload.InitialStock,
				}

				resp, err := svc.CreateNewProduct(reqCtx, grpcReq)
				if err != nil {
					log.Errorf("MQTT Worker: CreateNewProduct execution failed: %v", err)
					reqCancel()
					continue
				}
				log.Infof("MQTT Worker: Product successfully created via MQTT. Allocated ID: %s", resp.GetProduct().GetId())
			case updateTopic:
				var payload MqttUpdateStockPayload
				if err := json.Unmarshal(msg.Payload(), &payload); err != nil {
					log.Errorf("MQTT Worker: Failed to parse JSON stock update payload: %v", err)
					reqCancel()
					continue
				}

				log.Infof("MQTT Worker: Processing stock update for item '%s' with delta %d", payload.ID, payload.Delta)

				grpcReq := &pb.ChangeInventoryProductStockRequest{
					Id:    payload.ID,
					Delta: payload.Delta,
				}

				resp, err := svc.UpdateProductStock(reqCtx, grpcReq)
				if err != nil {
					log.Errorf("MQTT Worker: UpdateProductStock execution failed: %v", err)
					reqCancel()
					continue
				}
				log.Infof("MQTT Worker: Stock updated successfully via MQTT. Product ID: %s", resp.GetId())
			}

			reqCancel()
		}
	}()

	if token := client.Subscribe(createTopic, 1, messagePubHandler); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	fmt.Printf("Subscribed to topic: %s\n", createTopic)

	if token := client.Subscribe(updateTopic, 1, messagePubHandler); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	fmt.Printf("Subscribed to topic: %s\n", updateTopic)

	// Wait for interrupt signal to gracefully shutdown the subscriber
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
