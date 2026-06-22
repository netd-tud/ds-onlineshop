package main

import (
	"context"

	pb "github.com/turt1z/microservices-demo/src/inventoryservice/genproto"
	"google.golang.org/grpc/codes"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

type inventory struct {
	pb.UnimplementedInventoryServiceServer
	inventory pb.ListInventoryResponse
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

func (p *inventory) parseInventory() []*pb.InventoryProduct {
	if len(p.inventory.Products) == 0 {
		err := loadInventory(&p.inventory)
		if err != nil {
			return []*pb.InventoryProduct{}
		}
	}

	return p.inventory.Products
}
