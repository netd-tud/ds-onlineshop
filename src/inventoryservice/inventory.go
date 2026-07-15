package main

import (
	"context"
	"encoding/json"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	pb "github.com/turt1z/microservices-demo/src/inventoryservice/genproto"
	shared "github.com/turt1z/microservices-demo/src/shared"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

type inventory struct {
	pb.UnimplementedInventoryServiceServer
	inventory pb.ListInventoryResponse

	productCatalogSvcAddr string
	productCatalogSvcConn *grpc.ClientConn

	mqttBrokerAddr string
	mqttClient     mqtt.Client

	thresholds struct {
		lowStock      int64
		criticalStock int64
	}

	xaMu      sync.Mutex
	xaPending map[string]*pb.InventoryProduct
}

func (p *inventory) Check(ctx context.Context, req *healthpb.HealthCheckRequest) (*healthpb.HealthCheckResponse, error) {
	return &healthpb.HealthCheckResponse{Status: healthpb.HealthCheckResponse_SERVING}, nil
}

func (p *inventory) Watch(req *healthpb.HealthCheckRequest, ws healthpb.Health_WatchServer) error {
	return status.Errorf(codes.Unimplemented, "health check via Watch not implemented")
}

func (p *inventory) ListInventory(context.Context, *pb.Empty) (*pb.ListInventoryResponse, error) {
	return &pb.ListInventoryResponse{Products: p.parseInventory()}, nil
}

func (p *inventory) GetInventoryProduct(ctx context.Context, req *pb.GetInventoryProductRequest) (*pb.InventoryProduct, error) {
	inventory := p.parseInventory()
	for _, product := range inventory {
		if req.Id == product.Id {
			return product, nil
		}
	}

	return nil, status.Errorf(codes.NotFound, "no product with ID %s", req.Id)
}

func (p *inventory) ChangeInventoryProductStock(ctx context.Context, req *pb.ChangeInventoryProductStockRequest) (*pb.ChangeInventoryProductStockResponse, error) {
	claims, ok := shared.GetClaims(ctx)
	log.Infof("ChangeInventoryProductStock called for product with ID %s with claims: %v", req.Id, claims)
	if !ok {
		return nil, status.Error(codes.Internal, "failed to resolve user identity data from context")
	}
	log.Printf("ChangeInventoryProductStock called by user: %s, roles: %v", claims.Username, claims.Roles)

	if !p.userAllowedToModifyProduct(ctx, req.GetId(), *claims) {
		return nil, status.Error(codes.Unauthenticated, "user not allowed to modify product")
	}

	inventory := p.parseInventory()
	for _, product := range inventory {
		if req.Id == product.Id {
			newStock := product.Stock + req.Delta
			if newStock >= 0 {
				product.Stock = newStock
				p.publishStockEventOverMQTT(p.mqttBrokerAddr, product)
				return &pb.ChangeInventoryProductStockResponse{Product: product}, nil
			} else {
				return nil, status.Errorf(codes.Internal, "insufficient stock for product with ID %s", req.Id)
			}
		}
	}
	return nil, status.Errorf(codes.NotFound, "no product with ID %s", req.Id)
}

func (p *inventory) SetInventoryProductStock(ctx context.Context, req *pb.SetInventoryProductStockRequest) (*pb.SetInventoryProductStockRequestResponse, error) {
	inventory := p.parseInventory()
	for _, product := range inventory {
		if req.GetId() == product.GetId() {
			product.Stock = req.GetNewStock()
			p.publishStockEventOverMQTT(p.mqttBrokerAddr, product)
			return &pb.SetInventoryProductStockRequestResponse{Product: product}, nil
		}
	}
	// create product if non existent
	product := &pb.InventoryProduct{
		Id:    req.GetId(),
		Stock: req.GetNewStock(),
	}
	p.inventory.Products = append(p.parseInventory(), product)
	log.Infof("Inventory product updated: %s", product.Id)
	p.publishStockEventOverMQTT(p.mqttBrokerAddr, product)
	return &pb.SetInventoryProductStockRequestResponse{Product: product}, nil
}

func (p *inventory) CreateNewInventoryProduct(ctx context.Context, req *pb.CreateNewInventoryProductRequest) (*pb.CreateNewInventoryProductResponse, error) {
	// Simulate inventory failure if enabled
	configPath := "/var/behavior-config/FAIL_INVENTORY"

	configValue, _ := getConfigValue(configPath)
	if configValue != "" {
		if configValue == "true" {
			log.Warn("DEMO MODE ACTIVE: Returning gRPC Aborted code!")
			return nil, status.Error(codes.Aborted, "inventory allocation failed permanently")
		}
	}

	product := &pb.InventoryProduct{
		Id:    req.GetId(),
		Stock: req.GetInitialStock(),
	}
	p.inventory.Products = append(p.parseInventory(), product)
	log.Infof("Inventory product created: %s", product.Id)
	return &pb.CreateNewInventoryProductResponse{Product: product}, nil
}

func getConfigValue(configPath string) (string, error) {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Infof("Behavior file not found at %s, proceeding normally", configPath)
		return "", status.Error(codes.NotFound, "file not found")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Errorf("Error reading file stream: %v", err)
		return "", status.Error(codes.Internal, "failed to read behavior config")
	}

	configValue := strings.TrimSpace(string(data))
	log.Infof("Config Value read : '%s'", configValue)
	return configValue, nil
}

func (p *inventory) DeleteInventoryProduct(ctx context.Context, req *pb.DeleteInventoryProductRequest) (*pb.DeleteInventoryProductResponse, error) {
	inventory := p.parseInventory()
	for i, product := range inventory {
		if req.GetId() == product.GetId() {
			p.inventory.Products = append(inventory[:i], inventory[i+1:]...)
			log.Infof("Inventory product deleted: %s", product.Id)
			return &pb.DeleteInventoryProductResponse{Product: product}, nil
		}
	}
	return &pb.DeleteInventoryProductResponse{}, nil
}

func (p *inventory) CompensateCreateNewInventoryProduct(ctx context.Context, req *pb.CreateNewInventoryProductRequest) (*pb.DeleteInventoryProductResponse, error) {
	res, err := p.DeleteInventoryProduct(ctx, &pb.DeleteInventoryProductRequest{Id: req.GetId()})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to compensate create inventory product: %v", err)
	}
	log.Infof("Create Inventory product compensated: %s", req.GetId())
	return res, nil
}

func (p *inventory) parseInventory() []*pb.InventoryProduct {
	if len(p.inventory.Products) == 0 {
		err := loadInventory(&p.inventory)
		if err != nil {
			return []*pb.InventoryProduct{}
		}
	}

	return p.inventory.Products
}

type inventoryProductWithCategory struct {
	Id         string   `json:"id"`
	Stock      int64    `json:"stock"`
	Categories []string `json:"categories"`
}

func (p *inventory) publishStockEventOverMQTT(brokerAddr string, product *pb.InventoryProduct) {
	stock := product.GetStock()
	var subTopic string
	switch {
	case stock <= p.thresholds.criticalStock:
		subTopic = product.GetId() + "/stock/critical"
	case stock <= p.thresholds.lowStock:
		subTopic = product.GetId() + "/stock/low"
	default:
		subTopic = product.GetId() + "/stock/normal"
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()

	catalogResp, err := pb.NewProductCatalogServiceClient(p.productCatalogSvcConn).GetProduct(ctx, &pb.GetProductRequest{Id: product.GetId()})

	var categories []string
	if err != nil {
		log.Errorf("failed to connect to catalog: %s. Defaulting to uncategorized.", err)
		categories = []string{"uncategorized"}
	} else if catalogResp != nil {
		categories = catalogResp.Categories
	}

	combinedProduct := inventoryProductWithCategory{
		Id:         product.Id,
		Stock:      product.Stock,
		Categories: categories,
	}

	log.Infof("Retrieved following categories for product: %s", combinedProduct.Categories)

	payload, _ := json.Marshal(combinedProduct)
	for _, category := range combinedProduct.Categories {
		fullTopic := "inventory/" + category + "/" + subTopic
		log.Infof("Publishing event for topic '%s'...", fullTopic)
		go func(t string, pld []byte) {
			_ = p.publishEventOverMQTT(brokerAddr, t, pld)
		}(fullTopic, payload)
	}
}

func (p *inventory) publishEventOverMQTT(brokerAddr string, topic string, payload []byte) error {
	log.Infof("Attempting to publish event for topic '%s'...", topic)
	if p.mqttClient == nil || !p.mqttClient.IsConnected() {
		log.Errorf("MQTT client is not connected")
		return status.Error(codes.Internal, "MQTT client is not connected")
	}

	log.Printf("Publishing event for topic '%s'...", topic)
	token := p.mqttClient.Publish(topic, 1, false, payload)

	if finished := token.WaitTimeout(time.Second * 2); !finished {
		return status.Error(codes.DeadlineExceeded, "MQTT publish timed out")
	}
	if token.Error() != nil {
		return status.Errorf(codes.Internal, "MQTT publishing failed: %v", token.Error())
	}
	log.Printf("Published event for topic '%s' successfully", topic)
	return nil
}

func (p *inventory) userAllowedToModifyProduct(ctx context.Context, productId string, claims shared.UserClaims) bool {
	product, err := pb.NewProductCatalogServiceClient(p.productCatalogSvcConn).GetProduct(ctx, &pb.GetProductRequest{Id: productId})
	if err != nil {
		log.Errorf("failed to get product from catalog: %v", err)
		return false
	}

	categories := shared.ClaimsToCategories(&claims)
	for _, cat := range product.GetCategories() {
		target := shared.CategoryAccess{Category: shared.Category(cat), Permission: shared.PermissionWrite}
		if slices.Contains(categories, target) {
			return true
		}
	}

	return false
}
