package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/turt1z/microservices-demo/src/client/genproto"
)

const (
	grpcAddress = "ds-exercise-06.netd.cs.tu-dresden.de:30050"
	mqttBroker  = "tcp://ds-exercise-06.netd.cs.tu-dresden.de:31883"
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
	Delta int32  `json:"delta"`
}

func main() {
	// =================================================================
	// gRPC EXECUTION
	// =================================================================
	conn, err := grpc.NewClient(grpcAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("gRPC: Did not connect: %v", err)
	}
	defer conn.Close()

	grpcClient := pb.NewWarehouseManagementClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	log.Println("--- Calling CreateNewProduct via gRPC ---")
	createRes, err := grpcClient.CreateNewProduct(ctx, &pb.CreateWarehouseProductRequest{
		Name:        "Hat",
		Description: "A high-quality piece of clothing.",
		PriceUsd: &pb.Money{
			CurrencyCode: "USD",
			Units:        15,
			Nanos:        990000000,
		},
		Categories:   []string{"clothing", "accessories"},
		InitialStock: 100,
	})
	if err != nil {
		log.Fatalf("gRPC: Could not create product: %v", err)
	}
	gRPCProductID := createRes.GetProduct().GetId()
	log.Printf("gRPC: Product Created Successfully! ID: %s, Name: %s\n", gRPCProductID, createRes.GetProduct().GetName())

	log.Println("\n--- Calling UpdateProductStock via gRPC ---")
	updateRes, err := grpcClient.UpdateProductStock(ctx, &pb.ChangeInventoryProductStockRequest{
		Id:    gRPCProductID,
		Delta: 150,
	})
	if err != nil {
		log.Fatalf("gRPC: Could not update product stock: %v", err)
	}
	log.Printf("gRPC: Stock Updated Successfully! Product ID: %s, New Stock Level: %d\n", updateRes.GetId(), updateRes.GetStock())

	// =================================================================
	// MQTT PUBLISHING
	// =================================================================
	opts := mqtt.NewClientOptions().AddBroker(mqttBroker)
	opts.SetClientID("student_management_test_client")
	opts.SetConnectTimeout(time.Second * 5)

	mqttClient := mqtt.NewClient(opts)
	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("MQTT: Connection failed: %v", token.Error())
	}
	defer mqttClient.Disconnect(250)
	log.Println("MQTT: Connected successfully to broker")

	// Publish 1: Create  different item via MQTT
	createTopic := "inventory/create-item"
	newProductPayload := MqttCreateProductPayload{
		Name:        "Lighter",
		Description: "Simple, light Lighter.",
		PriceUsd: MqttMoney{
			CurrencyCode: "USD",
			Units:        1,
			Nanos:        500000000, // $1.50
		},
		Categories:   []string{"utility"},
		InitialStock: 50,
	}

	createBytes, err := json.Marshal(newProductPayload)
	if err != nil {
		log.Fatalf("MQTT: Failed to marshal creation payload: %v", err)
	}

	log.Printf("MQTT: Publishing creation request for '%s' to topic '%s'...\n", newProductPayload.Name, createTopic)
	token := mqttClient.Publish(createTopic, 1, false, createBytes)
	token.Wait()
	if token.Error() != nil {
		log.Printf("MQTT: Publishing creation failed: %v\n", token.Error())
	} else {
		log.Println("MQTT: Creation request published successfully")
	}

	// Publish 2: Update stock level for a product via MQTT
	// NOTE: Because MQTT is fire-and-forget, there currently is no information about the ID of the "Wireless Lab Mouse"
	// Updating the grpc Item for demonstration here
	updateTopic := "inventory/update-product-stock"
	updatePayload := MqttUpdateStockPayload{
		ID:    gRPCProductID,
		Delta: 75,
	}

	updateBytes, err := json.Marshal(updatePayload)
	if err != nil {
		log.Fatalf("MQTT: Failed to marshal update payload: %v", err)
	}

	log.Printf("MQTT: Publishing stock alteration (delta: %d) for ID '%s' to topic '%s'...\n", updatePayload.Delta, updatePayload.ID, updateTopic)
	token = mqttClient.Publish(updateTopic, 1, false, updateBytes)
	token.Wait()
	if token.Error() != nil {
		log.Printf("MQTT: Publishing stock update failed: %v\n", token.Error())
	} else {
		log.Println("MQTT: Stock update published successfully")
	}
}
