package main

import (
	"context"
	"encoding/json"

	"github.com/dtm-labs/client/dtmcli"
	"github.com/dtm-labs/client/dtmgrpc"
	"github.com/dtm-labs/client/workflow"
	commonpb "github.com/turt1z/microservices-demo/src/warehousemanagement/genproto/common"
	inventorypb "github.com/turt1z/microservices-demo/src/warehousemanagement/genproto/inventory"
	productcatalogpb "github.com/turt1z/microservices-demo/src/warehousemanagement/genproto/productcatalog"
	warehousemanagementpb "github.com/turt1z/microservices-demo/src/warehousemanagement/genproto/warehousemanagement"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const xaCreateProductWorkflow = "xa-create-product"

type XaCreateProductInput struct {
	ProductId string                                               `json:"product_id"`
	Req       *warehousemanagementpb.CreateWarehouseProductRequest `json:"req"`
}

func (wm *warehouseManagement) registerXaCreateProductWorkflow() error {
	return workflow.Register(xaCreateProductWorkflow, func(wf *workflow.Workflow, data []byte) error {
		input := XaCreateProductInput{}
		if err := json.Unmarshal(data, &input); err != nil {
			return err
		}
		req := input.Req
		productID := input.ProductId

		catalogCli := productcatalogpb.NewProductCatalogServiceClient(wm.xaProductCatalogConn)
		inventoryCli := inventorypb.NewInventoryServiceClient(wm.xaInventoryConn)

		// Branch 1: Product Catalog
		wf.NewBranch().OnCommit(func(bb *dtmcli.BranchBarrier) error {
			log.Info("XA: Committing product creation in catalog")
			_, err := catalogCli.XaCommitCreateProduct(wf.Context, &commonpb.XaBranchRequest{Gid: wf.Gid})
			return err
		}).OnRollback(func(bb *dtmcli.BranchBarrier) error {
			log.Info("XA: Rolling back product creation in catalog")
			_, err := catalogCli.XaRollbackCreateProduct(wf.Context, &commonpb.XaBranchRequest{Gid: wf.Gid})
			return err
		})
		log.Info("XA: Preparing product creation in catalog")
		if _, err := catalogCli.XaPrepareCreateProduct(wf.Context, &productcatalogpb.XaPrepareCreateProductRequest{
			Gid:         wf.Gid,
			Id:          productID,
			Name:        req.Name,
			Description: req.Description,
			PriceUsd:    req.PriceUsd,
			Categories:  req.Categories,
		}); err != nil {
			return err
		}

		// Branch 2: Inventory
		wf.NewBranch().OnCommit(func(bb *dtmcli.BranchBarrier) error {
			log.Info("XA: Committing inventory product creation")
			_, err := inventoryCli.XaCommitCreateInventoryProduct(wf.Context, &commonpb.XaBranchRequest{Gid: wf.Gid})
			return err
		}).OnRollback(func(bb *dtmcli.BranchBarrier) error {
			log.Info("XA: Rolling back inventory product creation")
			_, err := inventoryCli.XaRollbackCreateInventoryProduct(wf.Context, &commonpb.XaBranchRequest{Gid: wf.Gid})
			return err
		})
		log.Info("XA: Preparing inventory product creation")
		_, err := inventoryCli.XaPrepareCreateInventoryProduct(wf.Context, &inventorypb.XaPrepareCreateInventoryProductRequest{
			Gid:          wf.Gid,
			Id:           productID,
			InitialStock: req.InitialStock,
		})
		return err
	})
}

func (wm *warehouseManagement) createNewProductXa(ctx context.Context, req *warehousemanagementpb.CreateWarehouseProductRequest) (*warehousemanagementpb.CreateWarehouseProductResponse, error) {
	gid := dtmgrpc.MustGenGid(wm.dtmSvcAddr)
	productID, _ := generateID(10)

	data, err := json.Marshal(XaCreateProductInput{ProductId: productID, Req: req})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal request: %v", err)
	}

	log.Info("XA: Initiating workflow for product creation")
	if err := workflow.Execute(xaCreateProductWorkflow, gid, data); err != nil {
		return nil, status.Errorf(codes.Internal, "XA workflow failed: %v", err)
	}

	catalogResp, err := productcatalogpb.NewProductCatalogServiceClient(wm.productCatalogSvcConn).GetProduct(ctx, &productcatalogpb.GetProductRequest{Id: productID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "XA: failed to retrieve created product: %v", err)
	}
	log.Infof("XA: Retrieved created product from catalog: %v", catalogResp)

	invResp, err := inventorypb.NewInventoryServiceClient(wm.inventorySvcConn).GetInventoryProduct(ctx, &inventorypb.GetInventoryProductRequest{Id: productID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "XA: failed to retrieve created inventory product: %v", err)
	}
	log.Infof("XA: Retrieved created product from inventory: %v", invResp)

	return &warehousemanagementpb.CreateWarehouseProductResponse{Product: catalogResp}, nil
}
