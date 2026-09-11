package main

import (
	"context"

	commonpb "github.com/netd-tud/ds-onlineshop/src/productcatalogservice/genproto/common"
	productcatalogpb "github.com/netd-tud/ds-onlineshop/src/productcatalogservice/genproto/productcatalog"
)

// XaPrepareCreateProduct handles the prepare phase of a distributed two-phase commit (XA) transaction for product creation.
//
// It thread-safely checks for idempotency using the global transaction ID (GID), stages the new product
// in the pending map if not already present, logs the staging event, and returns an empty response.
func (pcs *productCatalogService) XaPrepareCreateProduct(ctx context.Context, req *productcatalogpb.XaPrepareCreateProductRequest) (*commonpb.Empty, error) {
	pcs.xaMu.Lock()
	defer pcs.xaMu.Unlock()

	if pcs.xaPending == nil {
		pcs.xaPending = map[string]*productcatalogpb.Product{}
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
	pcs.xaPending[req.Gid] = product
	log.Infof("XA: product %s prepared for gid %s", product.Id, req.Gid)
	return &commonpb.Empty{}, nil
}

// XaCommitCreateProduct handles the commit phase of a distributed two-phase commit (XA) transaction for product creation.
//
// It retrieves the staged product using the global transaction identifier (GID),
// appends it to the active catalog products list, clears the pending transaction state,
// and ensures idempotency for retried commit operations.
func (pcs *productCatalogService) XaCommitCreateProduct(ctx context.Context, req *commonpb.XaBranchRequest) (*commonpb.Empty, error) {
	pcs.xaMu.Lock()
	defer pcs.xaMu.Unlock()

	product, ok := pcs.xaPending[req.Gid]
	if !ok {
		// retried commit for a gid already committed -> idempotent no-op
		return &commonpb.Empty{}, nil
	}
	pcs.catalog.Products = append(pcs.parseCatalog(), product)
	delete(pcs.xaPending, req.Gid)
	log.Infof("XA: product %s committed for gid %s", product.Id, req.Gid)
	return &commonpb.Empty{}, nil
}

// XaRollbackCreateProduct handles the rollback phase of a distributed two-phase commit (XA) transaction for product creation.
//
// It removes the staged product using the global transaction identifier (GID), clears the pending transaction state,
// and ensures idempotency for retried rollback operations.
func (pcs *productCatalogService) XaRollbackCreateProduct(ctx context.Context, req *commonpb.XaBranchRequest) (*commonpb.Empty, error) {
	pcs.xaMu.Lock()
	defer pcs.xaMu.Unlock()

	// retried rollback for a gid already rolled back -> idempotent no-op
	if _, ok := pcs.xaPending[req.Gid]; !ok {
		return &commonpb.Empty{}, nil
	}
	delete(pcs.xaPending, req.Gid)
	log.Infof("XA: rolled back gid %s", req.Gid)
	return &commonpb.Empty{}, nil
}
