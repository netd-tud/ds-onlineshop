package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	commonpb "github.com/netd-tud/ds-onlineshop/src/inventoryservice/genproto/common"
	inventorypb "github.com/netd-tud/ds-onlineshop/src/inventoryservice/genproto/inventory"
	productcatalogpb "github.com/netd-tud/ds-onlineshop/src/inventoryservice/genproto/productcatalog"
	shared "github.com/netd-tud/ds-onlineshop/src/shared"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

// inventoryService represents the inventory microservice backend.
//
// It manages product stock levels, integrates with product catalog service,
// publishes real-time event updates via an MQTT broker, tracks alert thresholds,
// and maintains thread safety using internal mutexes for concurrent access.
type inventoryService struct {
	inventorypb.UnimplementedInventoryServiceServer

	stockMu   sync.Mutex
	inventory inventorypb.ListInventoryResponse

	productCatalogSvcAddr string
	productCatalogSvcConn *grpc.ClientConn

	mqttBrokerAddr string
	mqttClient     mqtt.Client

	thresholds struct {
		lowStock      int64
		criticalStock int64
	}

	xaMu      sync.Mutex
	xaPending map[string]*inventorypb.InventoryProduct

	resolvedAlerts   map[string]time.Time
	resolvedAlertTTL time.Duration
}

func (is *inventoryService) Check(ctx context.Context, req *healthpb.HealthCheckRequest) (*healthpb.HealthCheckResponse, error) {
	return &healthpb.HealthCheckResponse{Status: healthpb.HealthCheckResponse_SERVING}, nil
}

func (is *inventoryService) Watch(req *healthpb.HealthCheckRequest, ws healthpb.Health_WatchServer) error {
	return status.Errorf(codes.Unimplemented, "health check via Watch not implemented")
}

// ListInventory handles the gRPC request to retrieve the full list of inventory products.
//
// It returns a populated ListInventoryResponse containing the parsed inventory items
// managed by the service.
func (is *inventoryService) ListInventory(context.Context, *commonpb.Empty) (*inventorypb.ListInventoryResponse, error) {
	return &inventorypb.ListInventoryResponse{Products: is.parseInventory()}, nil
}

// GetInventoryProduct handles the gRPC request to retrieve a specific inventory product.
//
// It returns the requested InventoryProduct if found, otherwise an error.
func (is *inventoryService) GetInventoryProduct(ctx context.Context, req *inventorypb.GetInventoryProductRequest) (*inventorypb.InventoryProduct, error) {
	inventory := is.parseInventory()
	for _, product := range inventory {
		if req.Id == product.Id {
			return product, nil
		}
	}

	return nil, status.Errorf(codes.NotFound, "no product with ID %s", req.Id)
}

// applyStockDeltaLocked modifies the stock quantity of a specified product under lock
// and publishes the updated state over MQTT.
//
// It searches for the product by identifier, adjusts its stock by the given delta,
// ensures stock levels do not drop below zero, triggers an MQTT stock event notification,
// and returns the updated product. An error is returned if the product is not found or stock
// is insufficient.
func (is *inventoryService) applyStockDeltaLocked(productID string, delta int64) (*inventorypb.InventoryProduct, error) {
	for _, product := range is.parseInventory() {
		if product.GetId() != productID {
			continue
		}
		newStock := product.Stock + delta
		if newStock < 0 {
			return nil, status.Errorf(codes.Internal, "insufficient stock for product with ID %s", productID)
		}
		product.Stock = newStock
		is.publishStockEventOverMQTT(is.mqttBrokerAddr, product)
		return product, nil
	}
	return nil, status.Errorf(codes.NotFound, "no product with ID %s", productID)
}

// ChangeInventoryProductStock handles the gRPC request to adjust a product's stock level by a given delta.
//
// It extracts and verifies user authorization claims from the incoming context,
// validates whether the user has permission to modify the specific product,
// acquires a stock mutex lock, applies the requested delta update, and returns the modified product response.
func (is *inventoryService) ChangeInventoryProductStock(ctx context.Context, req *inventorypb.ChangeInventoryProductStockRequest) (*inventorypb.ChangeInventoryProductStockResponse, error) {
	claims, ok := shared.GetClaims(ctx)
	log.Infof("ChangeInventoryProductStock called for product with ID %s with claims: %v", req.Id, claims)
	if !ok {
		return nil, status.Error(codes.Internal, "failed to resolve user identity data from context")
	}
	log.Printf("ChangeInventoryProductStock called by user: %s, roles: %v", claims.Username, claims.Roles)

	if !is.userAllowedToModifyProduct(ctx, req.GetId(), *claims) {
		return nil, status.Error(codes.Unauthenticated, "user not allowed to modify product")
	}

	is.stockMu.Lock()
	defer is.stockMu.Unlock()

	product, err := is.applyStockDeltaLocked(req.GetId(), req.GetDelta())
	if err != nil {
		return nil, err
	}
	return &inventorypb.ChangeInventoryProductStockResponse{Product: product}, nil
}

// CompensateChangeInventoryProductStock handles the gRPC compensation request to revert a previous stock modification.
//
// It negates the requested stock delta and delegates the adjustment back to ChangeInventoryProductStock
// to roll back transactional stock changes.
func (is *inventoryService) CompensateChangeInventoryProductStock(ctx context.Context, req *inventorypb.ChangeInventoryProductStockRequest) (*inventorypb.ChangeInventoryProductStockResponse, error) {
	return is.ChangeInventoryProductStock(ctx, &inventorypb.ChangeInventoryProductStockRequest{
		Id:    req.GetId(),
		Delta: -req.GetDelta(),
	})
}

// SetInventoryProductStock handles the gRPC request to set an absolute stock level for a product.
//
// It searches the existing inventory for a matching product ID and updates its stock directly.
// If the product does not exist, it creates and appends a new inventory product entry,
// publishes the updated state over MQTT, and returns the modified product response.
func (is *inventoryService) SetInventoryProductStock(ctx context.Context, req *inventorypb.SetInventoryProductStockRequest) (*inventorypb.SetInventoryProductStockRequestResponse, error) {
	inventory := is.parseInventory()
	for _, product := range inventory {
		if req.GetId() == product.GetId() {
			product.Stock = req.GetNewStock()
			is.publishStockEventOverMQTT(is.mqttBrokerAddr, product)
			return &inventorypb.SetInventoryProductStockRequestResponse{Product: product}, nil
		}
	}
	// create product if non-existent
	product := &inventorypb.InventoryProduct{
		Id:    req.GetId(),
		Stock: req.GetNewStock(),
	}
	is.inventory.Products = append(is.parseInventory(), product)
	log.Infof("Inventory product updated: %s", product.Id)
	is.publishStockEventOverMQTT(is.mqttBrokerAddr, product)
	return &inventorypb.SetInventoryProductStockRequestResponse{Product: product}, nil
}

// CreateNewInventoryProduct handles the gRPC request to create a new product in the inventory.
//
// It checks for a simulated demo failure configuration file to conditionally abort the operation,
// instantiates a new inventory product with the requested initial stock, appends it to the inventory list,
// and returns the newly created product response.
func (is *inventoryService) CreateNewInventoryProduct(ctx context.Context, req *inventorypb.CreateNewInventoryProductRequest) (*inventorypb.CreateNewInventoryProductResponse, error) {
	// Simulate inventory failure if enabled
	configPath := "/var/behavior-config/FAIL_INVENTORY"

	configValue, _ := getConfigValue(configPath)
	if configValue != "" {
		if configValue == "true" {
			log.Warn("DEMO MODE ACTIVE: Returning gRPC Aborted code!")
			return nil, status.Error(codes.Aborted, "inventory allocation failed permanently")
		}
	}

	product := &inventorypb.InventoryProduct{
		Id:    req.GetId(),
		Stock: req.GetInitialStock(),
	}
	is.inventory.Products = append(is.parseInventory(), product)
	log.Infof("Inventory product created: %s", product.Id)
	return &inventorypb.CreateNewInventoryProductResponse{Product: product}, nil
}

// getConfigValue reads and retrieves a configuration string from the specified file path,
// returning an appropriate gRPC status error if the file does not exist or cannot be read.
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

// DeleteInventoryProduct handles the gRPC request to delete a product from the inventory.
//
// It searches the existing inventory for a matching product ID and removes it.
// If the product does not exist, it returns an appropriate error.
func (is *inventoryService) DeleteInventoryProduct(ctx context.Context, req *inventorypb.DeleteInventoryProductRequest) (*inventorypb.DeleteInventoryProductResponse, error) {
	inventory := is.parseInventory()
	for i, product := range inventory {
		if req.GetId() == product.GetId() {
			is.inventory.Products = append(inventory[:i], inventory[i+1:]...)
			log.Infof("Inventory product deleted: %s", product.Id)
			return &inventorypb.DeleteInventoryProductResponse{Product: product}, nil
		}
	}
	// return empty response
	return &inventorypb.DeleteInventoryProductResponse{}, nil
}

// ResolveStockAlert handles the gRPC request to resolve a product stock alert by restocking inventory.
//
// It verifies user claims and modification permissions, validates the reorder parameters and timestamp,
// checks for duplicate resolution requests via a deduplication key cache. Upon successful verification
// it applies the stock increase and returns the updated product along with a flag
// indicating whether the alert was already resolved before.
func (is *inventoryService) ResolveStockAlert(ctx context.Context, req *inventorypb.ResolveStockAlertRequest) (*inventorypb.ResolveStockAlertResponse, error) {
	claims, ok := shared.GetClaims(ctx)
	if !ok {
		return nil, status.Error(codes.Internal, "failed to resolve user identity data from context")
	}
	if !is.userAllowedToModifyProduct(ctx, req.GetProductId(), *claims) {
		return nil, status.Error(codes.Unauthenticated, "user not allowed to modify product")
	}
	if req.GetReorderAmount() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "reorder_amount must be positive")
	}
	if req.GetCreatedAt() == nil {
		return nil, status.Error(codes.InvalidArgument, "created_at is required")
	}

	key := fmt.Sprintf("%s|%d", req.GetProductId(), req.GetCreatedAt().AsTime().Unix())

	is.stockMu.Lock()
	defer is.stockMu.Unlock()

	if _, seen := is.resolvedAlerts[key]; seen {
		product, err := is.GetInventoryProduct(ctx, &inventorypb.GetInventoryProductRequest{Id: req.GetProductId()})
		if err != nil {
			return nil, err
		}
		return &inventorypb.ResolveStockAlertResponse{Product: product, AlreadyResolved: true}, nil
	}

	product, err := is.applyStockDeltaLocked(req.GetProductId(), req.GetReorderAmount())
	if err != nil {
		return nil, err
	}
	is.resolvedAlerts[key] = time.Now()

	return &inventorypb.ResolveStockAlertResponse{Product: product, AlreadyResolved: false}, nil
}

// reapResolvedAlerts periodically cleans up expired resolved stock alerts
// from the internal tracking cache based on a configured time-to-live duration.
func (is *inventoryService) reapResolvedAlerts(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		cutoff := time.Now().Add(-is.resolvedAlertTTL)
		is.stockMu.Lock()
		for key, resolvedAt := range is.resolvedAlerts {
			if resolvedAt.Before(cutoff) {
				delete(is.resolvedAlerts, key)
			}
		}
		is.stockMu.Unlock()
	}
}

// CompensateCreateNewInventoryProduct handles the gRPC compensation request to roll back a product creation.
//
// It invokes DeleteInventoryProduct using the requested product identifier to undo the creation operation.
func (is *inventoryService) CompensateCreateNewInventoryProduct(ctx context.Context, req *inventorypb.CreateNewInventoryProductRequest) (*inventorypb.DeleteInventoryProductResponse, error) {
	res, err := is.DeleteInventoryProduct(ctx, &inventorypb.DeleteInventoryProductRequest{Id: req.GetId()})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to compensate create inventory product: %v", err)
	}
	log.Infof("Create Inventory product compensated: %s", req.GetId())
	return res, nil
}

// parseInventory retrieves the list of inventory products, lazily loading them
// from the underlying storage if the internal product cache is uninitialized.
func (is *inventoryService) parseInventory() []*inventorypb.InventoryProduct {
	if len(is.inventory.Products) == 0 {
		err := loadInventory(&is.inventory)
		if err != nil {
			return []*inventorypb.InventoryProduct{}
		}
	}

	return is.inventory.Products
}

// inventoryProductWithCategory represents an enriched inventory product record,
// including its identifier, stock level, calculated severity, and associated categories.
type inventoryProductWithCategory struct {
	Id         string   `json:"id"`
	Stock      int64    `json:"stock"`
	Severity   string   `json:"severity"`
	Categories []string `json:"categories"`
}

// publishStockEventOverMQTT determines product stock severity levels, retrieves associated categories
// from the product catalog service, and asynchronously publishes serialized
// inventory status updates over MQTT to category-specific topics.
func (is *inventoryService) publishStockEventOverMQTT(brokerAddr string, product *inventorypb.InventoryProduct) {
	stock := product.GetStock()
	var severity string
	switch {
	case stock <= is.thresholds.criticalStock:
		severity = "critical"
	case stock <= is.thresholds.lowStock:
		severity = "low"
	default:
		severity = "normal"
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()

	catalogResp, err := productcatalogpb.NewProductCatalogServiceClient(is.productCatalogSvcConn).GetProduct(ctx, &productcatalogpb.GetProductRequest{Id: product.GetId()})

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
		Severity:   severity,
		Categories: categories,
	}

	log.Infof("Retrieved following categories for product: %s", combinedProduct.Categories)

	payload, _ := json.Marshal(combinedProduct)
	for _, category := range combinedProduct.Categories {
		fullTopic := "inventory/" + category + "/" + product.GetId() + "/stock"
		log.Infof("Publishing event for topic '%s'...", fullTopic)
		go func(t string, pld []byte) {
			_ = is.publishEventOverMQTT(brokerAddr, t, pld)
		}(fullTopic, payload)
	}
}

// publishEventOverMQTT sends a payload to a specified MQTT topic via the connected client,
// enforcing a publish timeout and verifying successful transmission.
func (is *inventoryService) publishEventOverMQTT(brokerAddr string, topic string, payload []byte) error {
	log.Infof("Attempting to publish event for topic '%s'...", topic)
	if is.mqttClient == nil || !is.mqttClient.IsConnected() {
		log.Errorf("MQTT client is not connected")
		return status.Error(codes.Internal, "MQTT client is not connected")
	}

	log.Printf("Publishing event for topic '%s'...", topic)
	token := is.mqttClient.Publish(topic, 1, true, payload)

	if finished := token.WaitTimeout(time.Second * 2); !finished {
		return status.Error(codes.DeadlineExceeded, "MQTT publish timed out")
	}
	if token.Error() != nil {
		return status.Errorf(codes.Internal, "MQTT publishing failed: %v", token.Error())
	}
	log.Printf("Published event for topic '%s' successfully", topic)
	return nil
}

// userAllowedToModifyProduct verifies whether a user is authorized to modify a specific product.
//
// It checks if the user possesses system or administrator roles, holds global write permissions,
// or has category-specific write permissions validated against the product catalog service.
func (is *inventoryService) userAllowedToModifyProduct(ctx context.Context, productId string, claims shared.UserClaims) bool {
	for _, role := range claims.Roles {
		if role == "SYSTEM_SERVICE" || role == "ADMIN" {
			log.WithField("role", role).Info("User has system service or admin role, allowing modification")
			return true
		}
	}

	categories := shared.ClaimsToCategories(&claims)
	log.WithField("categories", categories).Info("Checking user permissions")

	if slices.Contains(categories, shared.CategoryAccess{Category: "all", Permission: shared.PermissionWrite}) {
		return true
	}

	product, err := productcatalogpb.NewProductCatalogServiceClient(is.productCatalogSvcConn).GetProduct(ctx, &productcatalogpb.GetProductRequest{Id: productId})
	if err != nil {
		log.Errorf("failed to get product from catalog: %v", err)
		return false
	}

	for _, cat := range product.GetCategories() {
		target := shared.CategoryAccess{Category: shared.Category(cat), Permission: shared.PermissionWrite}
		if slices.Contains(categories, target) {
			return true
		}
	}

	return false
}
