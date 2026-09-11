package main

import (
	"context"

	commonpb "github.com/netd-tud/ds-onlineshop/src/inventoryservice/genproto/common"
	inventorypb "github.com/netd-tud/ds-onlineshop/src/inventoryservice/genproto/inventory"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// XaPrepareCreateInventoryProduct handles the prepare phase of a distributed two-phase commit (XA) transaction for product creation.
//
// It simulates failure conditions if demo mode is enabled, validates the initial stock level,
// stages the pending product creation mapped to the given global transaction identifier (GID), and ensures idempotency for retried calls.
func (is *inventoryService) XaPrepareCreateInventoryProduct(ctx context.Context, req *inventorypb.XaPrepareCreateInventoryProductRequest) (*commonpb.Empty, error) {
	configPath := "/var/behavior-config/FAIL_INVENTORY"

	configValue, _ := getConfigValue(configPath)
	if configValue != "" {
		if configValue == "true" {
			log.Warn("DEMO MODE ACTIVE: Returning gRPC Aborted code!")
			return nil, status.Error(codes.Aborted, "inventory allocation failed permanently")
		}
	}

	is.xaMu.Lock()
	defer is.xaMu.Unlock()

	if is.xaPending == nil {
		is.xaPending = map[string]*inventorypb.InventoryProduct{}
	}
	if _, exists := is.xaPending[req.Gid]; exists {
		// retried prepare for a gid already staged -> idempotent no-op
		return &commonpb.Empty{}, nil
	}
	if req.InitialStock < 0 {
		return nil, status.Error(codes.Aborted, "initial stock cannot be negative")
	}

	is.xaPending[req.Gid] = &inventorypb.InventoryProduct{Id: req.Id, Stock: req.InitialStock}
	log.Infof("XA: inventory product %s prepared for gid %s", req.Id, req.Gid)
	return &commonpb.Empty{}, nil
}

// XaCommitCreateInventoryProduct handles the commit phase of a distributed two-phase commit (XA) transaction for product creation.
//
// It retrieves the staged product using the global transaction identifier (GID),
// appends it to the active inventory list, clears the pending transaction state,
// and ensures idempotency for retried commit operations.
func (is *inventoryService) XaCommitCreateInventoryProduct(ctx context.Context, req *commonpb.XaBranchRequest) (*commonpb.Empty, error) {
	is.xaMu.Lock()
	defer is.xaMu.Unlock()

	product, ok := is.xaPending[req.Gid]
	if !ok {
		// retried commit for a gid already committed -> idempotent no-op
		return &commonpb.Empty{}, nil
	}
	is.inventory.Products = append(is.parseInventory(), product)
	delete(is.xaPending, req.Gid)
	log.Infof("XA: inventory product %s committed for gid %s", product.Id, req.Gid)
	return &commonpb.Empty{}, nil
}

// XaRollbackCreateInventoryProduct handles the rollback phase of a distributed two-phase commit (XA) transaction for product creation.
//
// It removes the staged product using the global transaction identifier (GID), clears the pending transaction state,
// and ensures idempotency for retried rollback operations.
func (is *inventoryService) XaRollbackCreateInventoryProduct(ctx context.Context, req *commonpb.XaBranchRequest) (*commonpb.Empty, error) {
	is.xaMu.Lock()
	defer is.xaMu.Unlock()

	// retried rollback for a gid already rolled back -> idempotent no-op
	if _, ok := is.xaPending[req.Gid]; !ok {
		return &commonpb.Empty{}, nil
	}
	delete(is.xaPending, req.Gid)
	log.Infof("XA: rolled back gid %s", req.Gid)
	return &commonpb.Empty{}, nil
}
