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

	pb "github.com/turt1z/microservices-demo/src/productcatalogservice/genproto"
	auth "github.com/turt1z/microservices-demo/src/shared"
	"google.golang.org/grpc/codes"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

type productCatalog struct {
	pb.UnimplementedProductCatalogServiceServer
	catalog   pb.ListProductsResponse
	xaMu      sync.Mutex
	xaPending map[string]*pb.Product
}

func (p *productCatalog) Check(ctx context.Context, req *healthpb.HealthCheckRequest) (*healthpb.HealthCheckResponse, error) {
	return &healthpb.HealthCheckResponse{Status: healthpb.HealthCheckResponse_SERVING}, nil
}

func (p *productCatalog) Watch(req *healthpb.HealthCheckRequest, ws healthpb.Health_WatchServer) error {
	return status.Errorf(codes.Unimplemented, "health check via Watch not implemented")
}

func (p *productCatalog) ListProducts(context.Context, *pb.Empty) (*pb.ListProductsResponse, error) {
	time.Sleep(extraLatency)

	return &pb.ListProductsResponse{Products: p.parseCatalog()}, nil
}

func (p *productCatalog) GetProduct(ctx context.Context, req *pb.GetProductRequest) (*pb.Product, error) {
	claims, ok := auth.GetClaims(ctx)
	log.Infof("GetProduct called with claims: %v", claims)
	if !ok {
		return nil, status.Error(codes.Internal, "failed to resolve user identity data from context")
	}
	log.Printf("GetProduct called by user: %s, roles: %v", claims.Username, claims.Roles)

	time.Sleep(extraLatency)

	catalog := p.parseCatalog()
	for _, product := range catalog {
		if req.Id == product.Id {
			return product, nil
		}
	}

	return nil, status.Errorf(codes.NotFound, "no product with ID %s", req.Id)
}

func (p *productCatalog) SearchProducts(ctx context.Context, req *pb.SearchProductsRequest) (*pb.SearchProductsResponse, error) {
	time.Sleep(extraLatency)

	var ps []*pb.Product
	for _, product := range p.parseCatalog() {
		if strings.Contains(strings.ToLower(product.Name), strings.ToLower(req.Query)) ||
			strings.Contains(strings.ToLower(product.Description), strings.ToLower(req.Query)) {
			ps = append(ps, product)
		}
	}

	return &pb.SearchProductsResponse{Results: ps}, nil
}

func (p *productCatalog) CreateNewProduct(ctx context.Context, req *pb.CreateNewProductRequest) (*pb.CreateNewProductResponse, error) {
	if req.Id == "" {
		newId, _ := generateID(10)
		req.Id = newId
	}
	product := &pb.Product{
		Id:          req.Id,
		Name:        req.Name,
		Description: req.Description,
		Picture:     "",
		PriceUsd:    req.PriceUsd,
		Categories:  req.Categories,
	}
	p.catalog.Products = append(p.parseCatalog(), product)
	log.Infof("Product created: %s", product.Id)
	return &pb.CreateNewProductResponse{Product: product}, nil
}

func (p *productCatalog) DeleteProduct(ctx context.Context, req *pb.DeleteProductRequest) (*pb.DeleteProductResponse, error) {
	catalog := p.parseCatalog()
	for i, product := range catalog {
		if req.GetId() == product.GetId() {
			p.catalog.Products = append(catalog[:i], catalog[i+1:]...)
			log.Infof("Product deleted: %s", product.Id)
			return &pb.DeleteProductResponse{Product: product}, nil
		}
	}
	return nil, status.Errorf(codes.NotFound, "no product with ID %s", req.Id)
}

func (p *productCatalog) CompensateCreateNewProduct(ctx context.Context, req *pb.CreateNewProductRequest) (*pb.DeleteProductResponse, error) {
	res, err := p.DeleteProduct(ctx, &pb.DeleteProductRequest{Id: req.GetId()})
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

func (p *productCatalog) parseCatalog() []*pb.Product {
	if reloadCatalog || len(p.catalog.Products) == 0 {
		err := loadCatalog(&p.catalog)
		if err != nil {
			return []*pb.Product{}
		}
	}

	return p.catalog.Products
}
