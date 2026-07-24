// Copyright 2018 Google LLC
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
	"encoding/json"
	"fmt"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	cartpb "github.com/turt1z/microservices-demo/src/checkoutservice/genproto/cart"
	checkoutpb "github.com/turt1z/microservices-demo/src/checkoutservice/genproto/checkout"
	commonpb "github.com/turt1z/microservices-demo/src/checkoutservice/genproto/common"
	currencypb "github.com/turt1z/microservices-demo/src/checkoutservice/genproto/currency"
	emailpb "github.com/turt1z/microservices-demo/src/checkoutservice/genproto/email"
	paymentpb "github.com/turt1z/microservices-demo/src/checkoutservice/genproto/payment"
	productcatalogpb "github.com/turt1z/microservices-demo/src/checkoutservice/genproto/productcatalog"
	shippingpb "github.com/turt1z/microservices-demo/src/checkoutservice/genproto/shipping"
	"github.com/turt1z/microservices-demo/src/checkoutservice/internal/analytics"
	"github.com/turt1z/microservices-demo/src/checkoutservice/money"
	"google.golang.org/grpc/codes"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func (cs *checkoutService) Check(ctx context.Context, req *healthpb.HealthCheckRequest) (*healthpb.HealthCheckResponse, error) {
	return &healthpb.HealthCheckResponse{Status: healthpb.HealthCheckResponse_SERVING}, nil
}

func (cs *checkoutService) Watch(req *healthpb.HealthCheckRequest, ws healthpb.Health_WatchServer) error {
	return status.Errorf(codes.Unimplemented, "health check via Watch not implemented")
}

func (cs *checkoutService) PlaceOrder(ctx context.Context, req *checkoutpb.PlaceOrderRequest) (*checkoutpb.PlaceOrderResponse, error) {
	log.Infof("[PlaceOrder] user_id=%q user_currency=%q", req.UserId, req.UserCurrency)

	var sID string
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if vals := md.Get("session-id"); len(vals) > 0 {
			sID = vals[0]
		}
	}

	orderID, err := uuid.NewUUID()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate order uuid")
	}

	cs.analyticsOrderPublisher.Publish(analytics.OrderEvent{
		EventType: analytics.EventCreate,
		SessionID: sID,
		OrderID:   orderID.String(),
	})

	prep, err := cs.prepareOrderItemsAndShippingQuoteFromCart(ctx, req.UserId, req.UserCurrency, req.Address)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	total := commonpb.Money{CurrencyCode: req.UserCurrency,
		Units: 0,
		Nanos: 0}
	total = money.Must(money.Sum(total, *prep.shippingCostLocalized))
	for _, it := range prep.orderItems {
		multPrice := money.MultiplySlow(*it.Cost, uint32(it.GetItem().GetQuantity()))
		total = money.Must(money.Sum(total, multPrice))
	}

	txID, err := cs.chargeCard(ctx, &total, req.CreditCard)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to charge card: %+v", err)
	}
	log.Infof("payment went through (transaction_id: %s)", txID)

	shippingTrackingID, err := cs.shipOrder(ctx, req.Address, prep.cartItems)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "shipping error: %+v", err)
	}

	_ = cs.emptyUserCart(ctx, req.UserId)

	orderResult := &commonpb.OrderResult{
		OrderId:            orderID.String(),
		ShippingTrackingId: shippingTrackingID,
		ShippingCost:       prep.shippingCostLocalized,
		ShippingAddress:    req.Address,
		Items:              prep.orderItems,
	}

	for _, item := range orderResult.Items {
		cs.analyticsProductsPublisher.Publish(analytics.ProductEvent{
			EventType: analytics.EventOrder,
			SessionID: sID,
			SKU:       item.GetItem().GetProductId(),
			Price:     *item.GetCost(),
			Qty:       item.GetItem().GetQuantity(),
			OrderID:   orderResult.OrderId,
		})
	}

	cs.analyticsOrderPublisher.Publish(analytics.OrderEvent{
		EventType: analytics.EventComplete,
		SessionID: sID,
		OrderID:   orderResult.OrderId,
		Price:     total,
	})

	if err := cs.sendOrderConfirmation(ctx, req.Email, orderResult); err != nil {
		log.Warnf("failed to send order confirmation to %q: %+v", req.Email, err)
	} else {
		log.Infof("order confirmation email sent to %q", req.Email)
	}
	resp := &checkoutpb.PlaceOrderResponse{Order: orderResult}
	return resp, nil
}

type orderPrep struct {
	orderItems            []*commonpb.OrderItem
	cartItems             []*commonpb.CartItem
	shippingCostLocalized *commonpb.Money
}

func (cs *checkoutService) prepareOrderItemsAndShippingQuoteFromCart(ctx context.Context, userID, userCurrency string, address *commonpb.Address) (orderPrep, error) {
	var out orderPrep
	cartItems, err := cs.getUserCart(ctx, userID)
	if err != nil {
		return out, fmt.Errorf("cart failure: %+v", err)
	}
	orderItems, err := cs.prepOrderItems(ctx, cartItems, userCurrency)
	if err != nil {
		return out, fmt.Errorf("failed to prepare order: %+v", err)
	}
	shippingUSD, err := cs.quoteShipping(ctx, address, cartItems)
	if err != nil {
		return out, fmt.Errorf("shipping quote failure: %+v", err)
	}
	shippingPrice, err := cs.convertCurrency(ctx, shippingUSD, userCurrency)
	if err != nil {
		return out, fmt.Errorf("failed to convert shipping cost to currency: %+v", err)
	}

	out.shippingCostLocalized = shippingPrice
	out.cartItems = cartItems
	out.orderItems = orderItems
	return out, nil
}

func (cs *checkoutService) quoteShipping(ctx context.Context, address *commonpb.Address, items []*commonpb.CartItem) (*commonpb.Money, error) {
	shippingQuote, err := shippingpb.NewShippingServiceClient(cs.shippingSvcConn).
		GetQuote(ctx, &shippingpb.GetQuoteRequest{
			Address: address,
			Items:   items})
	if err != nil {
		return nil, fmt.Errorf("failed to get shipping quote: %+v", err)
	}
	return shippingQuote.GetCostUsd(), nil
}

func (cs *checkoutService) getUserCart(ctx context.Context, userID string) ([]*commonpb.CartItem, error) {
	cart, err := cartpb.NewCartServiceClient(cs.cartSvcConn).GetCart(ctx, &cartpb.GetCartRequest{UserId: userID})
	if err != nil {
		return nil, fmt.Errorf("failed to get user cart during checkout: %+v", err)
	}
	return cart.GetItems(), nil
}

func (cs *checkoutService) emptyUserCart(ctx context.Context, userID string) error {
	if _, err := cartpb.NewCartServiceClient(cs.cartSvcConn).EmptyCart(ctx, &cartpb.EmptyCartRequest{UserId: userID}); err != nil {
		return fmt.Errorf("failed to empty user cart during checkout: %+v", err)
	}
	return nil
}

func (cs *checkoutService) prepOrderItems(ctx context.Context, items []*commonpb.CartItem, userCurrency string) ([]*commonpb.OrderItem, error) {
	out := make([]*commonpb.OrderItem, len(items))
	cl := productcatalogpb.NewProductCatalogServiceClient(cs.productCatalogSvcConn)

	for i, item := range items {
		product, err := cl.GetProduct(ctx, &productcatalogpb.GetProductRequest{Id: item.GetProductId()})
		if err != nil {
			return nil, fmt.Errorf("failed to get product #%q", item.GetProductId())
		}
		price, err := cs.convertCurrency(ctx, product.GetPriceUsd(), userCurrency)
		if err != nil {
			return nil, fmt.Errorf("failed to convert price of %q to %s", item.GetProductId(), userCurrency)
		}
		out[i] = &commonpb.OrderItem{
			Item: item,
			Cost: price}
	}
	return out, nil
}

func (cs *checkoutService) convertCurrency(ctx context.Context, from *commonpb.Money, toCurrency string) (*commonpb.Money, error) {
	result, err := currencypb.NewCurrencyServiceClient(cs.currencySvcConn).Convert(context.TODO(), &currencypb.CurrencyConversionRequest{
		From:   from,
		ToCode: toCurrency})
	if err != nil {
		return nil, fmt.Errorf("failed to convert currency: %+v", err)
	}
	return result, err
}

func (cs *checkoutService) chargeCard(ctx context.Context, amount *commonpb.Money, paymentInfo *paymentpb.CreditCardInfo) (string, error) {
	paymentResp, err := paymentpb.NewPaymentServiceClient(cs.paymentSvcConn).Charge(ctx, &paymentpb.ChargeRequest{
		Amount:     amount,
		CreditCard: paymentInfo})
	if err != nil {
		return "", fmt.Errorf("could not charge the card: %+v", err)
	}
	return paymentResp.GetTransactionId(), nil
}

func (cs *checkoutService) initializeMQTTClient() mqtt.Client {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s", cs.mqttBrokerAddr))
	opts.SetClientID("checkoutservice")
	opts.OnConnect = func(client mqtt.Client) {
		log.Info("Connected to MQTT")
	}
	opts.OnConnectionLost = func(client mqtt.Client, err error) {
		log.Errorf("Connection to MQTT lost: %v", err)
	}
	client := mqtt.NewClient(opts)
	log.Infof("Client: %v", client)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("MQTT Connection Error: %v", token.Error())
	}
	return client
}

func (cs *checkoutService) sendOrderConfirmation(ctx context.Context, email string, order *commonpb.OrderResult) error {
	if cs.mqttBrokerAddr != "" {
		return cs.sendOrderConfirmationMQTT(ctx, email, order)
	}
	return cs.sendOrderConfirmationgRPC(ctx, email, order)
}

func (cs *checkoutService) sendOrderConfirmationMQTT(ctx context.Context, email string, order *commonpb.OrderResult) error {
	type OrderEvent struct {
		Email string `json:"email"`
		Order string `json:"order"`
	}

	eventData, _ := json.Marshal(OrderEvent{
		Email: email,
		Order: order.OrderId,
	})

	token := cs.mqttClient.Publish("orders/checkout-complete", 1, false, eventData)
	if token.Error() != nil {
		return token.Error()
	}
	token.Wait()

	return nil
}

func (cs *checkoutService) sendOrderConfirmationgRPC(ctx context.Context, email string, order *commonpb.OrderResult) error {
	_, err := emailpb.NewEmailServiceClient(cs.emailSvcConn).SendOrderConfirmation(ctx, &emailpb.SendOrderConfirmationRequest{
		Email: email,
		Order: order})
	return err
}

func (cs *checkoutService) shipOrder(ctx context.Context, address *commonpb.Address, items []*commonpb.CartItem) (string, error) {
	resp, err := shippingpb.NewShippingServiceClient(cs.shippingSvcConn).ShipOrder(ctx, &shippingpb.ShipOrderRequest{
		Address: address,
		Items:   items})
	if err != nil {
		return "", fmt.Errorf("shipment failed: %+v", err)
	}
	return resp.GetTrackingId(), nil
}
