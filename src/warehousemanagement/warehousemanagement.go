package main

import (
	"context"

	pb "github.com/turt1z/microservices-demo/src/warehousemanagement/genproto"
	"google.golang.org/grpc/codes"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

func (pm *productManagement) Check(ctx context.Context, req *healthpb.HealthCheckRequest) (*healthpb.HealthCheckResponse, error) {
	return &healthpb.HealthCheckResponse{Status: healthpb.HealthCheckResponse_SERVING}, nil
}

func (pm *productManagement) Watch(req *healthpb.HealthCheckRequest, ws healthpb.Health_WatchServer) error {
	return status.Errorf(codes.Unimplemented, "health check via Watch not implemented")
}

func (pm *productManagement) UpdateProductStock(ctx context.Context, req *pb.ChangeInventoryProductStockRequest) (*pb.InventoryProduct, error) {
	resp, err := pb.NewInventoryServiceClient(pm.inventorySvcConn).ChangeInventoryProductStock(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.Product, nil
}

func (pm *productManagement) CreateNewProduct(ctx context.Context, req *pb.CreateWarehouseProductRequest) (*pb.CreateWarehouseProductResponse, error) {
	catalogResp, err := pb.NewProductCatalogServiceClient(pm.productCatalogSvcConn).CreateNewProduct(ctx, &pb.CreateNewProductRequest{
		Name:        req.Name,
		Description: req.Description,
		PriceUsd:    req.PriceUsd,
		Categories:  req.Categories,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create product in catalog: %v", err)
	}

	_, err = pb.NewInventoryServiceClient(pm.inventorySvcConn).SetInventoryProductStock(ctx, &pb.SetInventoryProductStockRequest{
		Id:       catalogResp.Product.Id,
		NewStock: req.InitialStock,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to set initial stock: %v", err)
	}

	return &pb.CreateWarehouseProductResponse{Product: catalogResp.Product}, nil
}
