package main

import (
	"strings"

	"github.com/dtm-labs/client/dtmgrpc"
	checkoutpb "github.com/netd-tud/ds-onlineshop/src/checkoutservice/genproto/checkout"
	commonpb "github.com/netd-tud/ds-onlineshop/src/checkoutservice/genproto/common"
	inventorypb "github.com/netd-tud/ds-onlineshop/src/checkoutservice/genproto/inventory"
	paymentpb "github.com/netd-tud/ds-onlineshop/src/checkoutservice/genproto/payment"
	shared "github.com/netd-tud/ds-onlineshop/src/shared"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (cs *checkoutService) placeOrderSaga(items []*checkoutpb.OrderItem, amount *commonpb.Money, cc *paymentpb.CreditCardInfo) error {
	systemToken, err := shared.GenerateSystemToken("checkout-service", []string{"SYSTEM_SERVICE"})
	if err != nil {
		return err
	}

	headers := map[string]string{
		"Authorization": "Bearer " + systemToken,
	}

	gid := dtmgrpc.MustGenGid(cs.dtmSvcAddr)
	saga := dtmgrpc.NewSagaGrpc(cs.dtmSvcAddr, gid, dtmgrpc.WithBranchHeaders(headers))

	for _, item := range items {
		saga.Add(
			cs.inventorySvcAddr+"/hipstershop.InventoryService/ChangeInventoryProductStock",
			cs.inventorySvcAddr+"/hipstershop.InventoryService/CompensateChangeInventoryProductStock",
			&inventorypb.ChangeInventoryProductStockRequest{
				Id:    item.GetItem().GetProductId(),
				Delta: -int64(item.GetItem().GetQuantity()),
			},
		)
	}

	saga.Add(
		cs.paymentSvcAddr+"/hipstershop.PaymentService/Charge",
		cs.paymentSvcAddr+"/hipstershop.PaymentService/CompensateCharge",
		&paymentpb.ChargeRequest{Amount: amount, CreditCard: cc},
	)

	saga.WaitResult = true
	if err := saga.Submit(); err != nil {
		if strings.Contains(err.Error(), "FAILURE") {
			return status.Errorf(codes.Aborted, "Order declined: insufficient stock or payment failed")
		}
		return status.Errorf(codes.Internal, "DTM-SAGA error: %v", err)
	}
	return nil
}
