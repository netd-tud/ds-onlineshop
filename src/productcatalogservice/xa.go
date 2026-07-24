package main

import (
	"context"

	commonpb "github.com/turt1z/microservices-demo/src/productcatalogservice/genproto/common"
	productcatalogpb "github.com/turt1z/microservices-demo/src/productcatalogservice/genproto/productcatalog"
)

func (p *productCatalog) XaPrepareCreateProduct(ctx context.Context, req *productcatalogpb.XaPrepareCreateProductRequest) (*commonpb.Empty, error) {
	p.xaMu.Lock()
	defer p.xaMu.Unlock()

	if p.xaPending == nil {
		p.xaPending = map[string]*productcatalogpb.Product{}
	}
	if _, exists := p.xaPending[req.Gid]; exists {
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
	p.xaPending[req.Gid] = product
	log.Infof("XA: product %s prepared for gid %s", product.Id, req.Gid)
	return &commonpb.Empty{}, nil
}

func (p *productCatalog) XaCommitCreateProduct(ctx context.Context, req *commonpb.XaBranchRequest) (*commonpb.Empty, error) {
	p.xaMu.Lock()
	defer p.xaMu.Unlock()

	product, ok := p.xaPending[req.Gid]
	if !ok {
		// retried commit for a gid already committed -> idempotent no-op
		return &commonpb.Empty{}, nil
	}
	p.catalog.Products = append(p.parseCatalog(), product)
	delete(p.xaPending, req.Gid)
	log.Infof("XA: product %s committed for gid %s", product.Id, req.Gid)
	return &commonpb.Empty{}, nil
}

func (p *productCatalog) XaRollbackCreateProduct(ctx context.Context, req *commonpb.XaBranchRequest) (*commonpb.Empty, error) {
	p.xaMu.Lock()
	defer p.xaMu.Unlock()

	// retried rollback for a gid already rolled back -> idempotent no-op
	if _, ok := p.xaPending[req.Gid]; !ok {
		return &commonpb.Empty{}, nil
	}
	delete(p.xaPending, req.Gid)
	log.Infof("XA: rolled back gid %s", req.Gid)
	return &commonpb.Empty{}, nil
}
