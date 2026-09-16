package main

import (
	"context"

	commonpb "github.com/netd-tud/ds-onlineshop/src/productcatalogservice/genproto/common"
	productcatalogpb "github.com/netd-tud/ds-onlineshop/src/productcatalogservice/genproto/productcatalog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// XaPrepareCreateProduct handles the prepare phase of a distributed two-phase commit (XA) transaction for product creation.
//
// It thread-safely checks for idempotency using the global transaction ID (GID), stages the new product
// in the pending map if not already present, logs the staging event, and returns an empty response.
func (pcs *productCatalogService) XaPrepareCreateProduct(ctx context.Context, req *productcatalogpb.XaPrepareCreateProductRequest) (*commonpb.Empty, error) {
	callerID, err := pcs.getCallerID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not get caller id: %v", err)
	}

	pcs.xaMu.Lock()
	defer pcs.xaMu.Unlock()

	if pcs.xaPending == nil {
		pcs.xaPending = map[string]*catalogEntry{}
	}
	if _, exists := pcs.xaPending[req.Gid]; exists {
		// retried prepare for a gid already staged -> idempotent no-op
		return &commonpb.Empty{}, nil
	}

	product := &productcatalogpb.Product{
		Id:          req.Id,
		Name:        req.Name,
		Description: req.Description,
		PriceUsd:    req.PriceUsd,
		Categories:  req.Categories,
	}
	pcs.xaPending[req.Gid] = &catalogEntry{
		product: product,
		owner:   callerID,
	}
	log.Infof("XA: product %s prepared for gid %s", product.Id, req.Gid)
	return &commonpb.Empty{}, nil
}

// XaCommitCreateProduct handles the commit phase of a distributed two-phase commit (XA) transaction for product creation.
//
// It retrieves the staged product using the global transaction identifier (GID),
// stores it in the active catalog products map, clears the pending transaction state,
// and ensures idempotency for retried commit operations.
func (pcs *productCatalogService) XaCommitCreateProduct(ctx context.Context, req *commonpb.XaBranchRequest) (*commonpb.Empty, error) {
	callerID, err := pcs.getCallerID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not get caller id: %v", err)
	}

	pcs.xaMu.Lock()
	defer pcs.xaMu.Unlock()

	entry, ok := pcs.xaPending[req.Gid]
	if !ok {
		// retried commit for a gid already committed -> idempotent no-op
		return &commonpb.Empty{}, nil
	}

	if entry.owner != callerID {
		return nil, status.Error(codes.PermissionDenied, "owner mismatch")
	}

	pcs.mu.Lock()
	pcs.ensureCatalogLoadedLocked()
	pcs.catalog[entry.product.Id] = entry
	pcs.mu.Unlock()

	delete(pcs.xaPending, req.Gid)
	log.Infof("XA: product %s committed for gid %s", entry.product.Id, req.Gid)
	return &commonpb.Empty{}, nil
}

// XaRollbackCreateProduct handles the rollback phase of a distributed two-phase commit (XA) transaction for product creation.
//
// It removes the staged product using the global transaction identifier (GID), clears the pending transaction state,
// and ensures idempotency for retried rollback operations.
func (pcs *productCatalogService) XaRollbackCreateProduct(ctx context.Context, req *commonpb.XaBranchRequest) (*commonpb.Empty, error) {
	callerID, err := pcs.getCallerID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not get caller id: %v", err)
	}

	pcs.xaMu.Lock()
	defer pcs.xaMu.Unlock()

	// retried rollback for a gid already rolled back -> idempotent no-op
	entry, ok := pcs.xaPending[req.Gid]
	if !ok {
		return &commonpb.Empty{}, nil
	}

	if entry.owner != callerID {
		return nil, status.Error(codes.PermissionDenied, "owner mismatch")
	}

	delete(pcs.xaPending, req.Gid)
	log.Infof("XA: rolled back gid %s", req.Gid)
	return &commonpb.Empty{}, nil
}
