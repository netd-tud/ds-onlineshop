package main

import (
	"context"
	"encoding/json"

	"github.com/dtm-labs/client/dtmcli"
	"github.com/dtm-labs/client/dtmgrpc"
	"github.com/dtm-labs/client/workflow"
	commonpb "github.com/netd-tud/ds-onlineshop/src/warehousemanagement/genproto/common"
	inventorypb "github.com/netd-tud/ds-onlineshop/src/warehousemanagement/genproto/inventory"
	productcatalogpb "github.com/netd-tud/ds-onlineshop/src/warehousemanagement/genproto/productcatalog"
	warehousemanagementpb "github.com/netd-tud/ds-onlineshop/src/warehousemanagement/genproto/warehousemanagement"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const xaCreateProductWorkflow = "xa-create-product"

// XaCreateProductInput defines the input payload used for executing two-phase commit (XA) workflows managed by DTM.
//
// It pairs a pre-generated product ID with the original gRPC request payload containing product creation metadata.
type XaCreateProductInput struct {
	ProductId string                                               `json:"product_id"`
	Req       *warehousemanagementpb.CreateWarehouseProductRequest `json:"req"`
}

// registerXaCreateProductWorkflow registers the DTM XA two-phase commit workflow for atomic product creation.
//
// It unmarshals the input payload, initializes XA branches for both the product catalog and inventory services,
// and defines their respective Prepare, Commit, and Rollback phase handlers using DTM branch barriers.
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

// createNewProductXa executes product creation using a 2-phase commit (XA) workflow coordinated by DTM.
//
// It generates a global transaction ID (GID) and product ID, serializes the workflow input payload,
// triggers the registered XA workflow, and retrieves the verified product and inventory records upon success.
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
