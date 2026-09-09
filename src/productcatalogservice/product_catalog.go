// Copyright 2023 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"strings"
	"sync"
	"time"

	commonpb "github.com/netd-tud/ds-onlineshop/src/productcatalogservice/genproto/common"
	productcatalogpb "github.com/netd-tud/ds-onlineshop/src/productcatalogservice/genproto/productcatalog"
	"google.golang.org/grpc/codes"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

type productCatalogService struct {
	productcatalogpb.UnimplementedProductCatalogServiceServer
	catalog   productcatalogpb.ListProductsResponse
	xaMu      sync.Mutex
	xaPending map[string]*productcatalogpb.Product
}

func (pcs *productCatalogService) Check(ctx context.Context, req *healthpb.HealthCheckRequest) (*healthpb.HealthCheckResponse, error) {
	return &healthpb.HealthCheckResponse{Status: healthpb.HealthCheckResponse_SERVING}, nil
}

func (pcs *productCatalogService) Watch(req *healthpb.HealthCheckRequest, ws healthpb.Health_WatchServer) error {
	return status.Errorf(codes.Unimplemented, "health check via Watch not implemented")
}

func (pcs *productCatalogService) ListProducts(context.Context, *commonpb.Empty) (*productcatalogpb.ListProductsResponse, error) {
	time.Sleep(extraLatency)

	return &productcatalogpb.ListProductsResponse{Products: pcs.parseCatalog()}, nil
}

func (pcs *productCatalogService) GetProduct(ctx context.Context, req *productcatalogpb.GetProductRequest) (*productcatalogpb.Product, error) {
	time.Sleep(extraLatency)

	catalog := pcs.parseCatalog()
	for _, product := range catalog {
		if req.Id == product.Id {
			return product, nil
		}
	}

	return nil, status.Errorf(codes.NotFound, "no product with ID %s", req.Id)
}

func (pcs *productCatalogService) SearchProducts(ctx context.Context, req *productcatalogpb.SearchProductsRequest) (*productcatalogpb.SearchProductsResponse, error) {
	time.Sleep(extraLatency)

	var ps []*productcatalogpb.Product
	for _, product := range pcs.parseCatalog() {
		if strings.Contains(strings.ToLower(product.Name), strings.ToLower(req.Query)) ||
			strings.Contains(strings.ToLower(product.Description), strings.ToLower(req.Query)) {
			ps = append(ps, product)
		}
	}

	return &productcatalogpb.SearchProductsResponse{Results: ps}, nil
}

func (pcs *productCatalogService) CreateNewProduct(ctx context.Context, req *productcatalogpb.CreateNewProductRequest) (*productcatalogpb.CreateNewProductResponse, error) {
	if req.Id == "" {
		newId, _ := generateID(10)
		req.Id = newId
	}
	product := &productcatalogpb.Product{
		Id:          req.Id,
		Name:        req.Name,
		Description: req.Description,
		Picture:     "",
		PriceUsd:    req.PriceUsd,
		Categories:  req.Categories,
	}
	pcs.catalog.Products = append(pcs.parseCatalog(), product)
	log.Infof("Product created: %s", product.Id)
	return &productcatalogpb.CreateNewProductResponse{Product: product}, nil
}

func (pcs *productCatalogService) DeleteProduct(ctx context.Context, req *productcatalogpb.DeleteProductRequest) (*productcatalogpb.DeleteProductResponse, error) {
	catalog := pcs.parseCatalog()
	for i, product := range catalog {
		if req.GetId() == product.GetId() {
			pcs.catalog.Products = append(catalog[:i], catalog[i+1:]...)
			log.Infof("Product deleted: %s", product.Id)
			return &productcatalogpb.DeleteProductResponse{Product: product}, nil
		}
	}
	return nil, status.Errorf(codes.NotFound, "no product with ID %s", req.Id)
}

func (pcs *productCatalogService) CompensateCreateNewProduct(ctx context.Context, req *productcatalogpb.CreateNewProductRequest) (*productcatalogpb.DeleteProductResponse, error) {
	res, err := pcs.DeleteProduct(ctx, &productcatalogpb.DeleteProductRequest{Id: req.GetId()})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to compensate create product: %v", err)
	}
	log.Infof("Create Product compensated: %s", req.GetId())
	return res, nil
}

func generateID(length int) (string, error) {
	// 6 bytes → 8 base64url chars, scale accordingly
	numBytes := (length*6)/8 + 1
	b := make([]byte, numBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b)[:length], nil
}

func (pcs *productCatalogService) parseCatalog() []*productcatalogpb.Product {
	if reloadCatalog || len(pcs.catalog.Products) == 0 {
		err := loadCatalog(&pcs.catalog)
		if err != nil {
			return []*productcatalogpb.Product{}
		}

		log.Info("Inserting into database...")
		if err := loadCatalogIntoPostgres(&pcs.catalog); err != nil {
			log.Warn("failed to insert product details into Postgres database: %v", err)
		}
	}

	return pcs.catalog.Products
}
