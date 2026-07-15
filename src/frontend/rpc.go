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
	"net/http"
	"time"

	pb "github.com/turt1z/microservices-demo/src/frontend/genproto"
	"google.golang.org/grpc/metadata"

	"github.com/pkg/errors"
)

const (
	avoidNoopCurrencyConversionRPC = false
)

func (fe *frontendServer) getCurrencies(ctx context.Context) ([]string, error) {
	currs, err := pb.NewCurrencyServiceClient(fe.currencySvcConn).
		GetSupportedCurrencies(ctx, &pb.Empty{})
	if err != nil {
		return nil, err
	}
	var out []string
	for _, c := range currs.CurrencyCodes {
		if _, ok := whitelistedCurrencies[c]; ok {
			out = append(out, c)
		}
	}
	return out, nil
}

func (fe *frontendServer) getProducts(ctx context.Context) ([]*pb.Product, error) {
	resp, err := pb.NewProductCatalogServiceClient(fe.productCatalogSvcConn).
		ListProducts(ctx, &pb.Empty{})
	return resp.GetProducts(), err
}

func (fe *frontendServer) getProduct(ctx context.Context, id string, cookie *http.Cookie) (*pb.Product, error) {
	tokenString := ""
	if cookie != nil {
		tokenString = cookie.Value
	}

	ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+tokenString)
	resp, err := pb.NewProductCatalogServiceClient(fe.productCatalogSvcConn).
		GetProduct(ctx, &pb.GetProductRequest{Id: id})
	return resp, err
}

func (fe *frontendServer) getCart(ctx context.Context, userID string) ([]*pb.CartItem, error) {
	resp, err := pb.NewCartServiceClient(fe.cartSvcConn).GetCart(ctx, &pb.GetCartRequest{UserId: userID})
	return resp.GetItems(), err
}

func (fe *frontendServer) emptyCart(ctx context.Context, userID string) error {
	_, err := pb.NewCartServiceClient(fe.cartSvcConn).EmptyCart(ctx, &pb.EmptyCartRequest{UserId: userID})
	return err
}

func (fe *frontendServer) insertCart(ctx context.Context, userID, productID string, quantity int32) error {
	_, err := pb.NewCartServiceClient(fe.cartSvcConn).AddItem(ctx, &pb.AddItemRequest{
		UserId: userID,
		Item: &pb.CartItem{
			ProductId: productID,
			Quantity:  quantity},
	})
	return err
}

func (fe *frontendServer) convertCurrency(ctx context.Context, money *pb.Money, currency string) (*pb.Money, error) {
	if avoidNoopCurrencyConversionRPC && money.GetCurrencyCode() == currency {
		return money, nil
	}
	return pb.NewCurrencyServiceClient(fe.currencySvcConn).
		Convert(ctx, &pb.CurrencyConversionRequest{
			From:   money,
			ToCode: currency})
}

func (fe *frontendServer) getShippingQuote(ctx context.Context, items []*pb.CartItem, currency string) (*pb.Money, error) {
	quote, err := pb.NewShippingServiceClient(fe.shippingSvcConn).GetQuote(ctx,
		&pb.GetQuoteRequest{
			Address: nil,
			Items:   items})
	if err != nil {
		return nil, err
	}
	localized, err := fe.convertCurrency(ctx, quote.GetCostUsd(), currency)
	return localized, errors.Wrap(err, "failed to convert currency for shipping cost")
}

func (fe *frontendServer) getRecommendations(ctx context.Context, userID string, productIDs []string) ([]*pb.Product, error) {
	resp, err := pb.NewRecommendationServiceClient(fe.recommendationSvcConn).ListRecommendations(ctx,
		&pb.ListRecommendationsRequest{UserId: userID, ProductIds: productIDs})
	if err != nil {
		return nil, err
	}
	out := make([]*pb.Product, len(resp.GetProductIds()))
	for i, v := range resp.GetProductIds() {
		p, err := fe.getProduct(ctx, v, nil)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get recommended product info (#%s)", v)
		}
		out[i] = p
	}
	if len(out) > 4 {
		out = out[:4] // take only first four to fit the UI
	}
	return out, err
}

func (fe *frontendServer) getAd(ctx context.Context, ctxKeys []string) ([]*pb.Ad, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Millisecond*100)
	defer cancel()

	resp, err := pb.NewAdServiceClient(fe.adSvcConn).GetAds(ctx, &pb.AdRequest{
		ContextKeys: ctxKeys,
	})
	return resp.GetAds(), errors.Wrap(err, "failed to get ads")
}

func (fe *frontendServer) listInventory(ctx context.Context) ([]*pb.InventoryProduct, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Millisecond*100)
	defer cancel()

	resp, err := pb.NewInventoryServiceClient(fe.inventorySvcConn).ListInventory(ctx, &pb.Empty{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get inventory")
	}
	inventory := make([]*pb.InventoryProduct, 0)
	for _, item := range resp.Products {
		inventory = append(inventory, item)
	}
	return inventory, nil
}

func (fe *frontendServer) getStock(ctx context.Context, productID string) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Millisecond*100)
	defer cancel()

	resp, err := pb.NewInventoryServiceClient(fe.inventorySvcConn).GetInventoryProduct(ctx,
		&pb.GetInventoryProductRequest{
			Id: productID,
		})
	if err != nil {
		return 0, errors.Wrapf(err, "failed to get stock for product #%s", productID)
	}
	return resp.GetStock(), nil
}

func (fe *frontendServer) reorderProduct(ctx context.Context, productID string, quantity int64, cookie *http.Cookie) (*pb.InventoryProduct, error) {
	tokenString := ""
	if cookie != nil {
		tokenString = cookie.Value
	}

	ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+tokenString)
	resp, err := pb.NewInventoryServiceClient(fe.inventorySvcConn).ChangeInventoryProductStock(ctx,
		&pb.ChangeInventoryProductStockRequest{
			Id:    productID,
			Delta: quantity,
		})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to reorder product #%s", productID)
	}
	return resp.Product, nil
}
