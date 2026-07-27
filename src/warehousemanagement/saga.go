package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"

	"github.com/dtm-labs/client/dtmgrpc"
	inventorypb "github.com/turt1z/microservices-demo/src/warehousemanagement/genproto/inventory"
	productcatalogpb "github.com/turt1z/microservices-demo/src/warehousemanagement/genproto/productcatalog"
	warehousemanagementpb "github.com/turt1z/microservices-demo/src/warehousemanagement/genproto/warehousemanagement"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (wm *warehouseManagement) createNewProductSaga(ctx context.Context, req *warehousemanagementpb.CreateWarehouseProductRequest) (*warehousemanagementpb.CreateWarehouseProductResponse, error) {
	productID, _ := generateID(10)
	gid := dtmgrpc.MustGenGid(wm.dtmSvcAddr)

	saga := dtmgrpc.NewSagaGrpc(wm.dtmSvcAddr, gid).
		Add(
			wm.productCatalogSvcAddr+"/hipstershop.ProductCatalogService/CreateNewProduct",
			wm.productCatalogSvcAddr+"/hipstershop.ProductCatalogService/CompensateCreateNewProduct",
			&productcatalogpb.CreateNewProductRequest{
				Id:          productID,
				Name:        req.Name,
				Description: req.Description,
				PriceUsd:    req.PriceUsd,
				Categories:  req.Categories,
			},
		).
		Add(
			wm.inventorySvcAddr+"/hipstershop.InventoryService/CreateNewInventoryProduct",
			wm.inventorySvcAddr+"/hipstershop.InventoryService/CompensateCreateNewInventoryProduct",
			&inventorypb.CreateNewInventoryProductRequest{
				Id:           productID,
				InitialStock: req.InitialStock,
			},
		)

	saga.WaitResult = true

	if err := saga.Submit(); err != nil {
		return nil, status.Errorf(codes.Internal, "DTM-SAGA: saga submission failed: %v", err)
	}

	catalogResp, err := productcatalogpb.NewProductCatalogServiceClient(wm.productCatalogSvcConn).GetProduct(ctx, &productcatalogpb.GetProductRequest{Id: productID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "DTM-SAGA: failed to retrieve created product: %v", err)
	}
	log.Infof("DTM-SAGA: Retrieved created product from catalog: %v", catalogResp)

	invResp, err := inventorypb.NewInventoryServiceClient(wm.inventorySvcConn).GetInventoryProduct(ctx, &inventorypb.GetInventoryProductRequest{Id: productID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "DTM-SAGA: failed to retrieve created inventory product: %v", err)
	}
	log.Infof("DTM-SAGA: Retrieved created product from inventory: %v", invResp)

	return &warehousemanagementpb.CreateWarehouseProductResponse{
		Product: catalogResp,
	}, nil
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
