package main

import (
	"context"

	pb "github.com/turt1z/microservices-demo/src/inventoryservice/genproto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (p *inventory) XaPrepareCreateInventoryProduct(ctx context.Context, req *pb.XaPrepareCreateInventoryProductRequest) (*pb.Empty, error) {
	configPath := "/var/behavior-config/FAIL_INVENTORY"

	configValue, _ := getConfigValue(configPath)
	if configValue != "" {
		if configValue == "true" {
			log.Warn("DEMO MODE ACTIVE: Returning gRPC Aborted code!")
			return nil, status.Error(codes.Aborted, "inventory allocation failed permanently")
		}
	}

	p.xaMu.Lock()
	defer p.xaMu.Unlock()

	if p.xaPending == nil {
		p.xaPending = map[string]*pb.InventoryProduct{}
	}
	if _, exists := p.xaPending[req.Gid]; exists {
		// retried prepare for a gid already staged -> idempotent no-op
		return &pb.Empty{}, nil
	}
	if req.InitialStock < 0 {
		return nil, status.Error(codes.Aborted, "initial stock cannot be negative")
	}

	p.xaPending[req.Gid] = &pb.InventoryProduct{Id: req.Id, Stock: req.InitialStock}
	log.Infof("XA: inventory product %s prepared for gid %s", req.Id, req.Gid)
	return &pb.Empty{}, nil
}

func (p *inventory) XaCommitCreateInventoryProduct(ctx context.Context, req *pb.XaBranchRequest) (*pb.Empty, error) {
	p.xaMu.Lock()
	defer p.xaMu.Unlock()

	product, ok := p.xaPending[req.Gid]
	if !ok {
		// retried commit for a gid already committed -> idempotent no-op
		return &pb.Empty{}, nil
	}
	p.inventory.Products = append(p.parseInventory(), product)
	delete(p.xaPending, req.Gid)
	log.Infof("XA: inventory product %s committed for gid %s", product.Id, req.Gid)
	return &pb.Empty{}, nil
}

func (p *inventory) XaRollbackCreateInventoryProduct(ctx context.Context, req *pb.XaBranchRequest) (*pb.Empty, error) {
	p.xaMu.Lock()
	defer p.xaMu.Unlock()

	// retried rollback for a gid already rolled back -> idempotent no-op
	if _, ok := p.xaPending[req.Gid]; !ok {
		return &pb.Empty{}, nil
	}
	delete(p.xaPending, req.Gid)
	log.Infof("XA: rolled back gid %s", req.Gid)
	return &pb.Empty{}, nil
}
