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

func (p *inventory) ChangeInventoryProductStock(ctx context.Context, req *pb.ChangeInventoryProductStockRequest) (*pb.ChangeInventoryProductStockResponse, error) {
	inventory := p.parseInventory()
	for _, product := range inventory {
		if req.Id == product.Id {
			newStock := product.Stock + req.Delta
			if newStock >= 0 {
				product.Stock = newStock
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
			return &pb.SetInventoryProductStockRequestResponse{Product: product}, nil
		}
	}
	// create product if non existent
	product := &pb.InventoryProduct{
		Id:    req.GetId(),
		Stock: req.GetNewStock(),
	}
	p.inventory.Products = append(p.parseInventory(), product)
	return &pb.SetInventoryProductStockRequestResponse{Product: product}, nil
}

func (p *inventory) CreateNewInventoryProduct(ctx context.Context, req *pb.CreateNewInventoryProductRequest) (*pb.CreateNewInventoryProductResponse, error) {
	product := &pb.InventoryProduct{
		Id:    req.GetId(),
		Stock: req.GetInitialStock(),
	}
	p.inventory.Products = append(p.parseInventory(), product)
	return &pb.CreateNewInventoryProductResponse{Product: product}, nil
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
