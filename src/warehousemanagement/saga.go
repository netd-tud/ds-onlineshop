package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"

	"github.com/dtm-labs/client/dtmgrpc"
	pb "github.com/turt1z/microservices-demo/src/warehousemanagement/genproto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (wm *warehouseManagement) createNewProductSaga(ctx context.Context, req *pb.CreateWarehouseProductRequest) (*pb.CreateWarehouseProductResponse, error) {
	productID, _ := generateID(10)
	gid := dtmgrpc.MustGenGid(wm.dtmSvcAddr)

	saga := dtmgrpc.NewSagaGrpc(wm.dtmSvcAddr, gid).
		Add(
			wm.productCatalogSvcAddr+"/hipstershop.ProductCatalogService/CreateNewProduct",
			wm.productCatalogSvcAddr+"/hipstershop.ProductCatalogService/CompensateCreateNewProduct",
			&pb.CreateNewProductRequest{
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
			&pb.CreateNewInventoryProductRequest{
				Id:           productID,
				InitialStock: req.InitialStock,
			},
		)

	saga.WaitResult = true

	if err := saga.Submit(); err != nil {
		return nil, status.Errorf(codes.Internal, "DTM-SAGA: saga submission failed: %v", err)
	}

	catalogResp, err := pb.NewProductCatalogServiceClient(wm.productCatalogSvcConn).GetProduct(ctx, &pb.GetProductRequest{Id: productID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "DTM-SAGA: failed to retrieve created product: %v", err)
	}
	log.Infof("DTM-SAGA: Retrieved created product from catalog: %v", catalogResp)

	invResp, err := pb.NewInventoryServiceClient(wm.inventorySvcConn).GetInventoryProduct(ctx, &pb.GetInventoryProductRequest{Id: productID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "DTM-SAGA: failed to retrieve created inventory product: %v", err)
	}
	log.Infof("DTM-SAGA: Retrieved created product from inventory: %v", invResp)

	return &pb.CreateWarehouseProductResponse{
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
