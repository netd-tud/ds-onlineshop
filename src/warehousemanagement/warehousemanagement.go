package main

import (
	"context"
	"os"
	"strings"

	pb "github.com/turt1z/microservices-demo/src/warehousemanagement/genproto"
	"google.golang.org/grpc/codes"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

func (wm *warehouseManagement) Check(ctx context.Context, req *healthpb.HealthCheckRequest) (*healthpb.HealthCheckResponse, error) {
	return &healthpb.HealthCheckResponse{Status: healthpb.HealthCheckResponse_SERVING}, nil
}

func (wm *warehouseManagement) Watch(req *healthpb.HealthCheckRequest, ws healthpb.Health_WatchServer) error {
	return status.Errorf(codes.Unimplemented, "health check via Watch not implemented")
}

func (wm *warehouseManagement) UpdateProductStock(ctx context.Context, req *pb.ChangeInventoryProductStockRequest) (*pb.InventoryProduct, error) {
	resp, err := pb.NewInventoryServiceClient(wm.inventorySvcConn).ChangeInventoryProductStock(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.Product, nil
}

func (wm *warehouseManagement) CreateNewProduct(ctx context.Context, req *pb.CreateWarehouseProductRequest) (*pb.CreateWarehouseProductResponse, error) {
	configPath := "/var/behavior-config/NAIVE_ROLLBACK_CREATE"
	configValue, _ := getConfigValue(configPath)
	log.Infof("Config Value for NAIVE_ROLLBACK_CREATE: %s", configValue)

	catalogResp, err := pb.NewProductCatalogServiceClient(wm.productCatalogSvcConn).CreateNewProduct(ctx, &pb.CreateNewProductRequest{
		Name:        req.Name,
		Description: req.Description,
		PriceUsd:    req.PriceUsd,
		Categories:  req.Categories,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create product in catalog: %v", err)
	}
	log.Infof("created product in catalog: %s", catalogResp.Product.Id)

	_, err = pb.NewInventoryServiceClient(wm.inventorySvcConn).CreateNewInventoryProduct(ctx, &pb.CreateNewInventoryProductRequest{
		Id:           catalogResp.Product.Id,
		InitialStock: req.InitialStock,
	})

	if configValue == "false" {
		// no compensating action executed
	} else {
		// naive/manual compensation if inventory is not reachable; rolling back product catalog creation
		if err != nil {
			log.Error("failed to set initial stock: %v, rolling back")
			_, invErr := pb.NewInventoryServiceClient(wm.inventorySvcConn).DeleteInventoryProduct(ctx, &pb.DeleteInventoryProductRequest{Id: catalogResp.Product.Id})
			if invErr != nil {
				log.Errorf("failed to rollback inventory creation for product: %s", catalogResp.Product.Id)
			}
			_, catErr := pb.NewProductCatalogServiceClient(wm.productCatalogSvcConn).DeleteProduct(ctx, &pb.DeleteProductRequest{Id: catalogResp.Product.Id})
			if catErr != nil {
				return nil, status.Errorf(codes.Internal, "failed to set initial stock: %v, failed to rollback catalog creation for product: %s, manual intervention needed", err, catalogResp.Product.Id)
			}
			return nil, status.Errorf(codes.Internal, "failed to set initial stock: %v, rolled back creation for product: %s", err, catalogResp.Product.Id)
		}
	}

	return &pb.CreateWarehouseProductResponse{Product: catalogResp.Product}, nil
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
