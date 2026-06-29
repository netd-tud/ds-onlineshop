package main

import (
	"context"

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
	catalogResp, err := pb.NewProductCatalogServiceClient(wm.productCatalogSvcConn).CreateNewProduct(ctx, &pb.CreateNewProductRequest{
		Name:        req.Name,
		Description: req.Description,
		PriceUsd:    req.PriceUsd,
		Categories:  req.Categories,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create product in catalog: %v", err)
	}

	_, err = pb.NewInventoryServiceClient(wm.inventorySvcConn).SetInventoryProductStock(ctx, &pb.SetInventoryProductStockRequest{
		Id:       catalogResp.Product.Id,
		NewStock: req.InitialStock,
	})

	// naive/manual compensation if inventory is not reachable; rolling back product catalog creation
	if err != nil {
		log.Error("failed to set initial stock: %v, rolling back catalog creation")
		_, delErr := pb.NewProductCatalogServiceClient(wm.productCatalogSvcConn).DeleteProduct(ctx, &pb.DeleteProductRequest{Id: catalogResp.Product.Id})
		if delErr != nil {
			log.Error("failed to rollback catalog creation: %v. Manual intervention needed", delErr)
		}
		return nil, status.Errorf(codes.Internal, "failed to set initial stock: %v, rolled back catalog creation for product: %s", err, catalogResp.Product.Id)
	}

	return &pb.CreateWarehouseProductResponse{Product: catalogResp.Product}, nil
}
