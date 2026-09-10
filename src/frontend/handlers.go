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
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"math/rand"
	"net"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	adpb "github.com/netd-tud/ds-onlineshop/src/frontend/genproto/ad"
	authpb "github.com/netd-tud/ds-onlineshop/src/frontend/genproto/auth"
	cartpb "github.com/netd-tud/ds-onlineshop/src/frontend/genproto/cart"
	checkoutpb "github.com/netd-tud/ds-onlineshop/src/frontend/genproto/checkout"
	commonpb "github.com/netd-tud/ds-onlineshop/src/frontend/genproto/common"
	notificationpb "github.com/netd-tud/ds-onlineshop/src/frontend/genproto/notification"
	paymentpb "github.com/netd-tud/ds-onlineshop/src/frontend/genproto/payment"
	productcatalogpb "github.com/netd-tud/ds-onlineshop/src/frontend/genproto/productcatalog"
	"github.com/netd-tud/ds-onlineshop/src/frontend/internal/analytics"
	"github.com/netd-tud/ds-onlineshop/src/frontend/money"
	"github.com/netd-tud/ds-onlineshop/src/frontend/validator"
	shared "github.com/netd-tud/ds-onlineshop/src/shared"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/metadata"
)

type platformDetails struct {
	css      string
	provider string
}

// ratings represents the aggregated review payload returned for a product,
// including the list of individual ratings and the computed average score.
type ratings struct {
	Ratings []rating `json:"ratings"`
	Average float32  `json:"average"`
}

// rating describes a single product review submitted to or returned from the
// rating microservice.
type rating struct {
	ID        string  `json:"id"`
	UserID    string  `json:"user_id"`
	Score     float32 `json:"score"`
	Body      string  `json:"body"`
	ProductID string  `json:"product_id"`
}

var (
	frontendMessage  = strings.TrimSpace(os.Getenv("FRONTEND_MESSAGE"))
	isCymbalBrand    = "true" == strings.ToLower(os.Getenv("CYMBAL_BRANDING"))
	assistantEnabled = "true" == strings.ToLower(os.Getenv("ENABLE_ASSISTANT"))

	// flags which enable/disable the specific services in the frontend ui
	authEnabled         = false
	inventoryEnabled    = false
	notificationEnabled = false

	templates = template.Must(template.New("").
			Funcs(template.FuncMap{
			"renderMoney":         renderMoney,
			"renderCurrencyLogo":  renderCurrencyLogo,
			"calculateOrderTotal": calculateOrderTotal,
		}).ParseGlob("templates/*.html"))
	plat platformDetails
)

var validEnvs = []string{"local", "gcp", "azure", "aws", "onprem", "alibaba"}

// heavyLoadHandler is an HTTP handler designed to intentionally consume
// CPU cycles for load testing and monitoring validation.
//
// It looks for an "iters" query parameter in the request URL to determine
// how many hashing iterations to perform, defaulting to 500,000 if omitted.
func (fe *frontendServer) heavyLoadHandler(w http.ResponseWriter, r *http.Request) {
	iters := r.URL.Query().Get("iters")
	if iters == "" {
		iters = "500000"
	}
	iterations, err := strconv.Atoi(iters)
	log.WithField("iterations", iterations).Info("heavy load request")
	if err != nil {
		renderHTTPError(log, r, w, errors.Wrap(err, "failed to parse iterations"), http.StatusBadRequest)
		return
	}
	w.Write([]byte(computeHeavyLoad(iterations)))
}

// computeHeavyLoad simulates heavy CPU utilization by performing sequential
// SHA256 hashing.
func computeHeavyLoad(iterations int) string {
	data := []byte("monitoring-load-test-payload-data")
	hash := sha256.Sum256(data)

	for range iterations {
		// Chain the hashes together to force serial CPU computation
		hash = sha256.Sum256(hash[:])
	}

	return fmt.Sprintf("%x", hash)
}

// openAlertsForRequest retrieves the stock alerts for the authenticated user
// based on their claims and sorts them based on the stock value.
func (fe *frontendServer) openAlertsForRequest(w http.ResponseWriter, r *http.Request) ([]*notificationpb.StockAlert, error) {
	cookie, err := r.Cookie(cookieAuth)
	if err != nil {
		return nil, err
	}
	claims, token, err := fe.claimsFromCookie(cookie)
	if err != nil || !token.Valid {
		fe.invalidateCookie(w, r, cookieAuth, err)
		return nil, err
	}

	categories := shared.ClaimsToCategories(claims)
	productAlerts, _ := fe.getProductAlerts(r.Context(), categories)

	slices.SortFunc(productAlerts, func(a, b *notificationpb.StockAlert) int {
		if a.Stock < b.Stock {
			return -1
		}
		if a.Stock > b.Stock {
			return 1
		}
		return 0
	})

	return productAlerts, nil
}

// recentOrdersForRequest retrieves the recent orders for the authenticated user
// based on their claims and partitions them by currencies.
func (fe *frontendServer) recentOrdersForRequest(w http.ResponseWriter, r *http.Request) (map[string]*notificationpb.OrderList, error) {
	cookie, err := r.Cookie(cookieAuth)
	if err != nil {
		return nil, err
	}
	claims, token, err := fe.claimsFromCookie(cookie)
	if err != nil || !token.Valid {
		fe.invalidateCookie(w, r, cookieAuth, err)
		return nil, err
	}

	currencies := shared.ClaimsToCurrencies(claims)
	recentOrdersPerCurrency, _ := fe.getRecentOrders(r.Context(), currencies)

	return recentOrdersPerCurrency, nil
}

// notificationHandler is an HTTP handler for rendering stock-alert and order notifications.
func (fe *frontendServer) notificationHandler(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(ctxKeyLog{}).(logrus.FieldLogger)

	productAlerts, err := fe.openAlertsForRequest(w, r)
	if err != nil {
		log.Warn("unauthenticated access attempt for product alerts")
		http.Redirect(w, r, baseUrl+"/login?next=/notifications", http.StatusFound)
		return
	}
	recentOrdersPerCurrency, err := fe.recentOrdersForRequest(w, r)
	if err != nil {
		log.Warn("unauthenticated access attempt for recent orders")
		http.Redirect(w, r, baseUrl+"/login?next=/notifications", http.StatusFound)
		return
	}

	if err := templates.ExecuteTemplate(w, "notifications", injectCommonTemplateData(r, map[string]any{
		"stock_alerts": productAlerts,
		"order_groups": recentOrdersPerCurrency,
	})); err != nil {
		log.Error(err)
	}
}

// product wraps inventory related information around a product catalog item,
// including its stock, severity level and weither the item is reorderable.
type product struct {
	Item        *productcatalogpb.Product
	Stock       int64
	Severity    string
	Reorderable bool
}

// inventoryHandler is an HTTP handler for rendering a list of inventory products
// the user is responsible for.
//
// It verifies the user is logged-in and redirects to the login page if not. Afterward,
// it retrieves product information from both productcatalog and inventory services, combines them,
// filters them based on the user's category access and sorts based on stock.
func (fe *frontendServer) inventoryHandler(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(ctxKeyLog{}).(logrus.FieldLogger)

	cookie, err := r.Cookie(cookieAuth)
	if err != nil {
		log.Warn("unauthenticated access attempt to inventory page: missing cookie")
		http.Redirect(w, r, baseUrl+"/login?next=/inventory", http.StatusFound)
		return
	}

	claims, token, err := fe.claimsFromCookie(cookie)

	if err != nil || !token.Valid {
		fe.invalidateCookie(w, r, cookieAuth, err)
		http.Redirect(w, r, baseUrl+"/login?next=/inventory", http.StatusFound)
		return
	}
	categoryAccess := shared.ClaimsToCategories(claims)

	products, _ := fe.getProducts(r.Context())
	inventoryProducts, _ := fe.listInventory(r.Context())

	combinedMap := make(map[string]*product, len(products)+len(inventoryProducts))
	for _, p := range products {
		combinedMap[p.GetId()] = &product{Item: p, Stock: 0, Reorderable: false}
	}
	for _, inventoryProduct := range inventoryProducts {
		if cp, ok := combinedMap[inventoryProduct.GetId()]; ok {
			cp.Stock = inventoryProduct.GetStock()
		} else {
			log.Warn("Could not find catalog Product corresponding to inventory Product with ID: %s", inventoryProduct.GetId())
		}
	}

	combinedList := make([]*product, 0, len(combinedMap))
	for _, cp := range combinedMap {
		combinedList = append(combinedList, cp)
	}

	filtered := combinedList
	if categoryAccess != nil {
		tmp := make([]*product, 0, len(combinedList))
		for _, cp := range combinedList {
			if cp == nil || cp.Item == nil {
				continue
			}
			severity := severityFromStock(cp.Stock)

			if slices.Contains(categoryAccess, shared.CategoryAccess{Category: shared.CategoryAll, Permission: shared.PermissionWrite}) {
				cp = &product{
					Item:        cp.Item,
					Stock:       cp.Stock,
					Severity:    severity,
					Reorderable: true,
				}
				tmp = append(tmp, cp)
				continue
			}
			for _, cat := range cp.Item.Categories {
				targetW := shared.CategoryAccess{Category: shared.Category(cat), Permission: shared.PermissionWrite}
				targetRO := shared.CategoryAccess{Category: shared.Category(cat), Permission: shared.PermissionRead}
				if slices.Contains(categoryAccess, targetW) {
					cp = &product{
						Item:        cp.Item,
						Stock:       cp.Stock,
						Severity:    severity,
						Reorderable: true,
					}
				} else if !slices.Contains(categoryAccess, targetRO) {
					break
				}
				tmp = append(tmp, cp)
				break
			}
		}
		filtered = tmp
	}

	slices.SortFunc(filtered, func(a, b *product) int {
		if a.Reorderable != b.Reorderable {
			if a.Reorderable {
				return -1
			}
			return 1
		}
		if a.Stock < b.Stock {
			return -1
		}
		if a.Stock > b.Stock {
			return 1
		}
		return 0
	})

	log.Infof("User %s has access to categories: %v, filtered inventory list has the following content: %v", claims.Username, categoryAccess, filtered)

	if err := templates.ExecuteTemplate(w, "reorder", injectCommonTemplateData(r, map[string]interface{}{
		"show_currency": false,
		"products":      filtered,
		"banner_color":  os.Getenv("BANNER_COLOR"),
	})); err != nil {
		log.Error(err)
	}
}

// claimsFromCookie extracts the user claims and jwt token from a http cookie.
func (fe *frontendServer) claimsFromCookie(cookie *http.Cookie) (*shared.UserClaims, *jwt.Token, error) {
	tokenString := cookie.Value
	claims := &shared.UserClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		return fe.publicKey, nil
	})
	return claims, token, err
}

// invalidateCookie invalidates a cookie by the given name and request.
//
// The cookie is invalidated by setting MaxAge to -1.
func (fe *frontendServer) invalidateCookie(w http.ResponseWriter, r *http.Request, cookieName string, err error) {
	log.WithError(err).Warn("invalidating cookie")
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		MaxAge:   -1,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		// Secure: true,
	})
	http.Redirect(w, r, baseUrl+"/login", http.StatusFound)
}

// profileHandler is an HTTP handler, which redirects the user to /account if they are logged-in or to /login if not.
func (fe *frontendServer) profileHandler(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(ctxKeyLog{}).(logrus.FieldLogger)

	cookie, err := r.Cookie(cookieAuth)
	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			log.Info("no auth cookie found, redirecting to login page")
		} else {
			log.WithError(err).Error("error retrieving auth cookie")
		}
		http.Redirect(w, r, baseUrl+"/login", http.StatusFound)
		return
	}

	tokenString := cookie.Value
	claims := &shared.UserClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return fe.publicKey, nil
	})

	if err != nil || !token.Valid {
		log.WithError(err).Warn("stale or invalid token detected, clearing session")

		fe.invalidateCookie(w, r, cookieAuth, err)
		return
	}

	log.WithField("username", claims.Username).Info("valid token confirmed, directing to account")
	http.Redirect(w, r, baseUrl+"/account", http.StatusFound)
}

// loginHandler is an HTTP handler for processeing HTTP requests for the user authentication flow.
//
// For GET requests, it renders the login form and preserves the user's
// intended post-login destination via the "next" query parameter.
//
// For POST requests, it validates the submitted credentials against the
// backend authentication gRPC service. Upon success, it establishes the
// user session by setting an HTTP-only JWT cookie and a short-lived
// notification flag. It then redirects the user to their original
// destination, strictly sanitizing the redirect target to prevent external
// routing exploits.
func (fe *frontendServer) loginHandler(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(ctxKeyLog{}).(logrus.FieldLogger)

	if r.Method == http.MethodGet {
		nextTarget := r.URL.Query().Get("next")
		if err := templates.ExecuteTemplate(w, "login", injectCommonTemplateData(r, map[string]interface{}{
			"next": nextTarget,
		})); err != nil {
			log.Error(err)
		}
		return
	}

	// Handle POST
	nextTarget := r.FormValue("next")
	username := r.FormValue("uid")
	password := r.FormValue("password")

	log.WithField("username", username).Info("login attempt")
	resp, err := authpb.NewAuthServiceClient(fe.authSvcConn).Login(r.Context(), &authpb.LoginRequest{
		Username: username,
		Password: password,
	})

	if err != nil {
		log.WithError(err).Warn("login failed")
		if err := templates.ExecuteTemplate(w, "login", injectCommonTemplateData(r, map[string]interface{}{
			"error": "Invalid username or password",
			"next":  nextTarget,
		})); err != nil {
			log.Error(err)
		}
		return
	}

	log.WithField("username", username).Info("login successful")

	http.SetCookie(w, &http.Cookie{
		Name:     cookieAuth,
		Value:    resp.GetToken(),
		MaxAge:   cookieMaxAge,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		// Secure: true,
	})

	// Security check: ensure the redirect path is strictly local to prevent open redirect vulnerabilities.
	if nextTarget == "" || !strings.HasPrefix(nextTarget, "/") || strings.HasPrefix(nextTarget, "//") {
		nextTarget = "/"
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "notif_check",
		Value:    "1",
		MaxAge:   10,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	log.Warn("Setting notif_check cookie for post-login alerts")

	http.Redirect(w, r, baseUrl+nextTarget, http.StatusFound)
}

// reorderHandler processes HTTP requests to trigger a product restock.
//
// It verifies that the user is currently authenticated, extracts the target
// product ID and requested quantity from the form payload, and delegates
// the reorder operation to the backend.
func (fe *frontendServer) reorderHandler(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(ctxKeyLog{}).(logrus.FieldLogger)

	cookie, err := r.Cookie(cookieAuth)
	if err != nil {
		log.Warn("unauthenticated access attempt to reorder product: missing cookie")
		http.Redirect(w, r, baseUrl+"/login", http.StatusFound)
		return
	}

	_ = cookie

	productId := r.FormValue("product_id")
	quantity, err := strconv.ParseInt(r.FormValue("quantity"), 10, 64)
	if err != nil {
		log.WithError(err).Warn("invalid quantity value")
		http.Redirect(w, r, baseUrl+"/inventory", http.StatusFound)
		return
	}

	log.Infof("Reorder request for product %s with quantity %s", productId, quantity)

	resp, err := fe.reorderProduct(r.Context(), productId, quantity, cookie)
	if err != nil {
		log.WithError(err).Warn("reorder failed")
	}

	log.Infof("Reorder response: %v", resp)

	http.Redirect(w, r, baseUrl+"/inventory", http.StatusFound)
}

// accountHandler processes HTTP requests to display the user's account profile.
//
// It requires an active authenticated session. It extracts and validates
// the JWT from the user's cookie, mapping the embedded identity claims
// (such as name, title, and roles) directly into the view. If the session
// is missing, expired, or invalid, the request is safely aborted and the
// user is redirected to re-authenticate.
func (fe *frontendServer) accountHandler(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(ctxKeyLog{}).(logrus.FieldLogger)

	cookie, err := r.Cookie(cookieAuth)
	if err != nil {
		log.Warn("unauthenticated access attempt to account page: missing cookie")
		http.Redirect(w, r, baseUrl+"/login", http.StatusFound)
		return
	}

	claims, token, err := fe.claimsFromCookie(cookie)

	log.Infof("Claims: %s", claims)

	if err != nil || !token.Valid {
		fe.invalidateCookie(w, r, cookieAuth, err)
		return
	}

	templateData := map[string]interface{}{
		"Username": claims.Username,
		"UserID":   claims.UserID,
		"Roles":    claims.Roles,
		"Name":     claims.Name,
		"Title":    claims.Title,
	}

	if err := templates.ExecuteTemplate(w, "account", injectCommonTemplateData(r, templateData)); err != nil {
		log.Error(err)
	}
}

// homeHandler processes HTTP requests for the application landing page.
//
// It aggregates core data by fetching available currencies, the
// product catalog, and the user's active shopping cart. It converts product
// prices to the user's preferred currency, checks for post-login
// stock notifications, and renders the home template.
func (fe *frontendServer) homeHandler(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(ctxKeyLog{}).(logrus.FieldLogger)
	log.WithField("currency", currentCurrency(r)).Info("home")
	currencies, err := fe.getCurrencies(r.Context())
	if err != nil {
		renderHTTPError(log, r, w, errors.Wrap(err, "could not retrieve currencies"), http.StatusInternalServerError)
		return
	}
	products, err := fe.getProducts(r.Context())
	if err != nil {
		renderHTTPError(log, r, w, errors.Wrap(err, "could not retrieve products"), http.StatusInternalServerError)
		return
	}
	cart, err := fe.getCart(r.Context(), sessionID(r))
	if err != nil {
		renderHTTPError(log, r, w, errors.Wrap(err, "could not retrieve cart"), http.StatusInternalServerError)
		return
	}

	type productView struct {
		Item  *productcatalogpb.Product
		Price *commonpb.Money
	}
	ps := make([]productView, len(products))
	for i, p := range products {
		price, err := fe.convertCurrency(r.Context(), p.GetPriceUsd(), currentCurrency(r))
		if err != nil {
			renderHTTPError(log, r, w, errors.Wrapf(err, "failed to do currency conversion for product %s", p.GetId()), http.StatusInternalServerError)
			return
		}
		ps[i] = productView{p, price}
	}

	// Set ENV_PLATFORM (default to local if not set; use env var if set; otherwise detect GCP, which overrides env)_
	var env = os.Getenv("ENV_PLATFORM")
	// Only override from env variable if set + valid env
	if env == "" || stringinSlice(validEnvs, env) == false {
		fmt.Println("env platform is either empty or invalid")
		env = "local"
	}
	// Autodetect GCP
	addrs, err := net.LookupHost("metadata.google.internal.")
	if err == nil && len(addrs) >= 0 {
		log.Debugf("Detected Google metadata server: %v, setting ENV_PLATFORM to GCP.", addrs)
		env = "gcp"
	}

	log.Debugf("ENV_PLATFORM is: %s", env)
	plat = platformDetails{}
	plat.setPlatformDetails(strings.ToLower(env))

	var postLoginAlerts []*notificationpb.StockAlert
	if _, err := r.Cookie("notif_check"); err == nil {
		postLoginAlerts, _ = fe.openAlertsForRequest(w, r)
	}
	log.Warn("Post-login alerts: %v", postLoginAlerts)

	log.Warn("Home handler called, auth is ", authEnabled)
	if err := templates.ExecuteTemplate(w, "home", injectCommonTemplateData(r, map[string]interface{}{
		"show_currency":     true,
		"currencies":        currencies,
		"products":          ps,
		"cart_size":         cartSize(cart),
		"banner_color":      os.Getenv("BANNER_COLOR"), // illustrates canary deployments
		"ad":                fe.chooseAd(r.Context(), []string{}, log),
		"post_login_alerts": postLoginAlerts,
	})); err != nil {
		log.Error(err)
	}
}

func (plat *platformDetails) setPlatformDetails(env string) {
	if env == "aws" {
		plat.provider = "AWS"
		plat.css = "aws-platform"
	} else if env == "onprem" {
		plat.provider = "On-Premises"
		plat.css = "onprem-platform"
	} else if env == "azure" {
		plat.provider = "Azure"
		plat.css = "azure-platform"
	} else if env == "gcp" {
		plat.provider = "Google Cloud"
		plat.css = "gcp-platform"
	} else if env == "alibaba" {
		plat.provider = "Alibaba Cloud"
		plat.css = "alibaba-platform"
	} else {
		plat.provider = "local"
		plat.css = "local"
	}
}

// productHandler processes HTTP requests to display an individual product detail page.
//
// It extracts the product ID from the URL path, retrieves the catalog details,
// converts prices into the user's preferred currency, and loads supporting data
// such as active cart size and recommendations. It then optionally queries
// external services for ratings, inventory stock levels, and packaging info,
// logs a view analytics event, and renders the product template.
func (fe *frontendServer) productHandler(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(ctxKeyLog{}).(logrus.FieldLogger)
	id := mux.Vars(r)["id"]
	if id == "" {
		renderHTTPError(log, r, w, errors.New("product id not specified"), http.StatusBadRequest)
		return
	}
	log.WithField("id", id).WithField("currency", currentCurrency(r)).
		Debug("serving product page")

	cookie, _ := r.Cookie(cookieAuth)
	p, err := fe.getProduct(r.Context(), id, cookie)
	if err != nil {
		renderHTTPError(log, r, w, errors.Wrap(err, "could not retrieve product"), http.StatusInternalServerError)
		return
	}
	currencies, err := fe.getCurrencies(r.Context())
	if err != nil {
		renderHTTPError(log, r, w, errors.Wrap(err, "could not retrieve currencies"), http.StatusInternalServerError)
		return
	}

	cart, err := fe.getCart(r.Context(), sessionID(r))
	if err != nil {
		renderHTTPError(log, r, w, errors.Wrap(err, "could not retrieve cart"), http.StatusInternalServerError)
		return
	}

	price, err := fe.convertCurrency(r.Context(), p.GetPriceUsd(), currentCurrency(r))
	if err != nil {
		renderHTTPError(log, r, w, errors.Wrap(err, "failed to convert currency"), http.StatusInternalServerError)
		return
	}

	// ignores the error retrieving recommendations since it is not critical
	recommendations, err := fe.getRecommendations(r.Context(), sessionID(r), []string{id})
	if err != nil {
		log.WithField("error", err).Warn("failed to get product recommendations")
	}

	product := struct {
		Item     *productcatalogpb.Product
		Price    *commonpb.Money
		Ratings  *ratings
		Stock    int64
		Severity string
	}{p, price, nil, -1, ""}

	if fe.ratingSvcAddr != "" {
		resp, err := http.Get(fmt.Sprintf("http://%s/ratings/product/%s", fe.ratingSvcAddr, p.GetId()))
		log.Println("Response: %s", resp)
		if err == nil {
			var ratings ratings
			defer resp.Body.Close()
			if err := json.NewDecoder(resp.Body).Decode(&ratings); err == nil {
				log.Println("Ratings: %s", ratings)
				product.Ratings = &ratings
			}
		} else {
			log.WithField("error", err).Warn("failed to connect to ratingservice")
		}
	}

	if fe.inventorySvcAddr != "" {
		stock, err := fe.getStock(r.Context(), p.GetId())
		if err != nil {
			log.WithField("error", err).Warn("failed to get stock from inventory service")
		} else {
			log.WithField("stock", stock).Debug("got stock from inventory service")
			product.Stock = stock
			product.Severity = severityFromStock(stock)
		}
	}

	// Fetch packaging info (weight/dimensions) of the product
	// The packaging service is an optional microservice you can run as part of a Google Cloud demo.
	var packagingInfo *PackagingInfo = nil
	if isPackagingServiceConfigured() {
		packagingInfo, err = httpGetPackagingInfo(id)
		if err != nil {
			fmt.Println("Failed to obtain product's packaging info:", err)
		}
	}

	fe.analyticsPublisher.Publish(analytics.ProductEvent{
		EventType: analytics.EventView,
		SKU:       p.GetId(),
		SessionID: sessionID(r),
	})

	if err := templates.ExecuteTemplate(w, "product", injectCommonTemplateData(r, map[string]interface{}{
		"ad":              fe.chooseAd(r.Context(), p.Categories, log),
		"show_currency":   true,
		"currencies":      currencies,
		"product":         product,
		"recommendations": recommendations,
		"cart_size":       cartSize(cart),
		"packagingInfo":   packagingInfo,
	})); err != nil {
		log.Println(err)
	}
}

// ratingHandler processes HTTP POST requests to submit a new product rating.
//
// It restricts access to POST operations, parses the submitted review data
// from the form, marshals the payload into JSON, and forwards it to the
// downstream rating microservice. Upon successful creation, it redirects
// the user back to the respective product detail page.
func (fe *frontendServer) ratingHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	score, _ := strconv.ParseFloat(r.FormValue("score"), 32)

	rating := rating{
		Score:     float32(score),
		Body:      r.FormValue("body"),
		ProductID: r.FormValue("product_id"),
		UserID:    r.FormValue("user_id")[:4],
	}

	log.WithField("product", rating.ProductID).WithField("rating", rating).Debug("adding rating")

	jsonData, err := json.Marshal(rating)
	if err != nil {
		log.WithError(err).Error("failed to marshal rating JSON")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	microserviceURL := fmt.Sprintf("http://%s/ratings/new", fe.ratingSvcAddr)

	proxyReq, err := http.NewRequest(http.MethodPost, microserviceURL, bytes.NewBuffer(jsonData))
	if err != nil {
		log.WithError(err).Error("failed to create microservice request")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	proxyReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(proxyReq)
	if err != nil {
		log.WithError(err).Error("rating microservice connection failed")
		http.Error(w, "Service unavailable", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {
		http.Redirect(w, r, fmt.Sprintf("/product/%s", rating.ProductID), http.StatusSeeOther)
	} else {
		log.WithField("status", resp.StatusCode).Error("microservice rejected rating submission")
		http.Error(w, "Failed to submit review to downstream service", http.StatusInternalServerError)
	}
}

// addToCartHandler processes HTTP requests to add a specified product quantity to the user's shopping cart.
//
// It parses and validates the form input using the add-to-cart payload validator,
// verifies the product exists via the catalog service, propagates the user session ID
// via gRPC metadata, inserts the item into the cart.
// Then redirects the client to the cart view.
func (fe *frontendServer) addToCartHandler(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(ctxKeyLog{}).(logrus.FieldLogger)
	quantity, _ := strconv.ParseUint(r.FormValue("quantity"), 10, 32)
	productID := r.FormValue("product_id")
	payload := validator.AddToCartPayload{
		Quantity:  quantity,
		ProductID: productID,
	}
	if err := payload.Validate(); err != nil {
		renderHTTPError(log, r, w, validator.ValidationErrorResponse(err), http.StatusUnprocessableEntity)
		return
	}
	log.WithField("product", payload.ProductID).WithField("quantity", payload.Quantity).Debug("adding to cart")

	cookie, _ := r.Cookie(cookieAuth)
	p, err := fe.getProduct(r.Context(), payload.ProductID, cookie)
	if err != nil {
		renderHTTPError(log, r, w, errors.Wrap(err, "could not retrieve product"), http.StatusInternalServerError)
		return
	}

	ctx := r.Context()

	sID := sessionID(r)
	if sID != "" {
		ctx = metadata.AppendToOutgoingContext(r.Context(), "session-id", sID)
	}

	if err := fe.insertCart(ctx, sessionID(r), p.GetId(), int32(payload.Quantity)); err != nil {
		renderHTTPError(log, r, w, errors.Wrap(err, "failed to add to cart"), http.StatusInternalServerError)
		return
	}
	w.Header().Set("location", baseUrl+"/cart")
	w.WriteHeader(http.StatusFound)
}

// emptyCartHandler processes HTTP requests to clear all items from the user's shopping cart.
//
// It retrieves the current session identifier, invokes the backend cart-clearing
// operation, and redirects the client back to the home page upon success.
func (fe *frontendServer) emptyCartHandler(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(ctxKeyLog{}).(logrus.FieldLogger)
	log.Debug("emptying cart")

	if err := fe.emptyCart(r.Context(), sessionID(r)); err != nil {
		renderHTTPError(log, r, w, errors.Wrap(err, "failed to empty cart"), http.StatusInternalServerError)
		return
	}
	w.Header().Set("location", baseUrl+"/")
	w.WriteHeader(http.StatusFound)
}

// viewCartHandler processes HTTP requests to display the user's shopping cart.
//
// It retrieves available currencies, active cart items, and product catalog details,
// converting prices into the user's preferred currency. It computes shipping
// estimates, calculates itemized totals, checks for any transient order error
// cookies to display flash messages, and renders the cart template.
func (fe *frontendServer) viewCartHandler(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(ctxKeyLog{}).(logrus.FieldLogger)
	log.Debug("view user cart")
	currencies, err := fe.getCurrencies(r.Context())
	if err != nil {
		renderHTTPError(log, r, w, errors.Wrap(err, "could not retrieve currencies"), http.StatusInternalServerError)
		return
	}
	cart, err := fe.getCart(r.Context(), sessionID(r))
	if err != nil {
		renderHTTPError(log, r, w, errors.Wrap(err, "could not retrieve cart"), http.StatusInternalServerError)
		return
	}

	// ignores the error retrieving recommendations since it is not critical
	recommendations, err := fe.getRecommendations(r.Context(), sessionID(r), cartIDs(cart))
	if err != nil {
		log.WithField("error", err).Warn("failed to get product recommendations")
	}

	shippingCost, err := fe.getShippingQuote(r.Context(), cart, currentCurrency(r))
	if err != nil {
		renderHTTPError(log, r, w, errors.Wrap(err, "failed to get shipping quote"), http.StatusInternalServerError)
		return
	}

	type cartItemView struct {
		Item     *productcatalogpb.Product
		Quantity int32
		Price    *commonpb.Money
	}
	items := make([]cartItemView, len(cart))
	totalPrice := commonpb.Money{CurrencyCode: currentCurrency(r)}
	for i, item := range cart {
		cookie, _ := r.Cookie(cookieAuth)
		p, err := fe.getProduct(r.Context(), item.GetProductId(), cookie)
		if err != nil {
			renderHTTPError(log, r, w, errors.Wrapf(err, "could not retrieve product #%s", item.GetProductId()), http.StatusInternalServerError)
			return
		}
		price, err := fe.convertCurrency(r.Context(), p.GetPriceUsd(), currentCurrency(r))
		if err != nil {
			renderHTTPError(log, r, w, errors.Wrapf(err, "could not convert currency for product #%s", item.GetProductId()), http.StatusInternalServerError)
			return
		}

		multPrice := money.MultiplySlow(*price, uint32(item.GetQuantity()))
		items[i] = cartItemView{
			Item:     p,
			Quantity: item.GetQuantity(),
			Price:    &multPrice}
		totalPrice = money.Must(money.Sum(totalPrice, multPrice))
	}
	totalPrice = money.Must(money.Sum(totalPrice, *shippingCost))
	year := time.Now().Year()

	var errorMessage string

	if cookie, err := r.Cookie("order_error"); err == nil {
		errorMessage = cookie.Value

		clearCookie := &http.Cookie{
			Name:     "order_error",
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
		}
		http.SetCookie(w, clearCookie)
	}

	if err := templates.ExecuteTemplate(w, "cart", injectCommonTemplateData(r, map[string]interface{}{
		"error":            errorMessage,
		"currencies":       currencies,
		"recommendations":  recommendations,
		"cart_size":        cartSize(cart),
		"shipping_cost":    shippingCost,
		"show_currency":    true,
		"total_cost":       totalPrice,
		"items":            items,
		"expiration_years": []int{year, year + 1, year + 2, year + 3, year + 4},
	})); err != nil {
		log.Println(err)
	}
}

// placeOrderHandler processes HTTP POST requests to finalize a user purchase.
//
// It parses and validates shipping and payment details from the form payload,
// injects session metadata into the outgoing context, and issues a PlaceOrder
// gRPC call to the checkout service. If the checkout fails due to a business
// logic abort (e.g., payment failure or insufficient stock), it sets a transient
// error cookie and redirects back to the cart; otherwise, it calculates the
// total amount paid and renders the order confirmation view.
func (fe *frontendServer) placeOrderHandler(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(ctxKeyLog{}).(logrus.FieldLogger)
	log.Debug("placing order")

	var (
		email         = r.FormValue("email")
		streetAddress = r.FormValue("street_address")
		zipCode, _    = strconv.ParseInt(r.FormValue("zip_code"), 10, 32)
		city          = r.FormValue("city")
		state         = r.FormValue("state")
		country       = r.FormValue("country")
		ccNumber      = r.FormValue("credit_card_number")
		ccMonth, _    = strconv.ParseInt(r.FormValue("credit_card_expiration_month"), 10, 32)
		ccYear, _     = strconv.ParseInt(r.FormValue("credit_card_expiration_year"), 10, 32)
		ccCVV, _      = strconv.ParseInt(r.FormValue("credit_card_cvv"), 10, 32)
	)

	payload := validator.PlaceOrderPayload{
		Email:         email,
		StreetAddress: streetAddress,
		ZipCode:       zipCode,
		City:          city,
		State:         state,
		Country:       country,
		CcNumber:      ccNumber,
		CcMonth:       ccMonth,
		CcYear:        ccYear,
		CcCVV:         ccCVV,
	}
	if err := payload.Validate(); err != nil {
		renderHTTPError(log, r, w, validator.ValidationErrorResponse(err), http.StatusUnprocessableEntity)
		return
	}

	ctx := r.Context()
	sID := sessionID(r)
	if sID != "" {
		ctx = metadata.AppendToOutgoingContext(r.Context(), "session-id", sID)
	}

	order, err := checkoutpb.NewCheckoutServiceClient(fe.checkoutSvcConn).
		PlaceOrder(ctx, &checkoutpb.PlaceOrderRequest{
			Email: payload.Email,
			CreditCard: &paymentpb.CreditCardInfo{
				CreditCardNumber:          payload.CcNumber,
				CreditCardExpirationMonth: int32(payload.CcMonth),
				CreditCardExpirationYear:  int32(payload.CcYear),
				CreditCardCvv:             int32(payload.CcCVV)},
			UserId:       sessionID(r),
			UserCurrency: currentCurrency(r),
			Address: &commonpb.Address{
				StreetAddress: payload.StreetAddress,
				City:          payload.City,
				State:         payload.State,
				ZipCode:       int32(payload.ZipCode),
				Country:       payload.Country},
		})
	if err != nil {
		errStr := err.Error()

		businessFailure := strings.Contains(errStr, "code = Aborted") &&
			strings.Contains(errStr, "insufficient stock or payment failed")

		if businessFailure {
			http.SetCookie(w, &http.Cookie{
				Name:     "order_error",
				Value:    "Checkout failed: Insufficient stock or payment error.",
				Path:     "/",
				MaxAge:   60,
				HttpOnly: true,
			})

			http.Redirect(w, r, "/cart", http.StatusSeeOther)
			return
		}

		renderHTTPError(log, r, w, errors.Wrap(err, "failed to complete the order"), http.StatusInternalServerError)
		return
	}
	log.WithField("order", order.GetOrder().GetOrderId()).Info("order placed")

	order.GetOrder().GetItems()
	recommendations, _ := fe.getRecommendations(ctx, sessionID(r), nil)

	totalPaid := *order.GetOrder().GetShippingCost()
	for _, v := range order.GetOrder().GetItems() {
		multPrice := money.MultiplySlow(*v.GetCost(), uint32(v.GetItem().GetQuantity()))
		totalPaid = money.Must(money.Sum(totalPaid, multPrice))
	}

	currencies, err := fe.getCurrencies(ctx)
	if err != nil {
		renderHTTPError(log, r, w, errors.Wrap(err, "could not retrieve currencies"), http.StatusInternalServerError)
		return
	}

	if err := templates.ExecuteTemplate(w, "order", injectCommonTemplateData(r, map[string]interface{}{
		"show_currency":   false,
		"currencies":      currencies,
		"order":           order.GetOrder(),
		"total_paid":      &totalPaid,
		"recommendations": recommendations,
	})); err != nil {
		log.Println(err)
	}
}

func (fe *frontendServer) assistantHandler(w http.ResponseWriter, r *http.Request) {
	currencies, err := fe.getCurrencies(r.Context())
	if err != nil {
		renderHTTPError(log, r, w, errors.Wrap(err, "could not retrieve currencies"), http.StatusInternalServerError)
		return
	}

	if err := templates.ExecuteTemplate(w, "assistant", injectCommonTemplateData(r, map[string]interface{}{
		"show_currency": false,
		"currencies":    currencies,
	})); err != nil {
		log.Println(err)
	}
}

// logoutHandler processes HTTP requests to terminate the user session.
//
// It iterates through all active cookies present in the request, explicitly
// invalidating and expiring each one, and then redirects the client
// back to the home page.
func (fe *frontendServer) logoutHandler(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(ctxKeyLog{}).(logrus.FieldLogger)
	log.Debug("logging out")
	for _, c := range r.Cookies() {
		c.Expires = time.Now().Add(-time.Hour * 24 * 365)
		c.MaxAge = -1
		http.SetCookie(w, c)
	}
	w.Header().Set("Location", baseUrl+"/")
	w.WriteHeader(http.StatusFound)
}

// getProductByID processes HTTP requests to retrieve and return a single product as JSON.
//
// It extracts the product identifier from the router variables, fetches the product
// details from the catalog service using the authentication cookie, marshals
// the payload into JSON, and writes the resulting data directly to the HTTP response.
func (fe *frontendServer) getProductByID(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["ids"]
	if id == "" {
		return
	}

	cookie, _ := r.Cookie(cookieAuth)
	p, err := fe.getProduct(r.Context(), id, cookie)
	if err != nil {
		return
	}

	jsonData, err := json.Marshal(p)
	if err != nil {
		fmt.Println(err)
		return
	}

	w.Write(jsonData)
	w.WriteHeader(http.StatusOK)
}

func (fe *frontendServer) chatBotHandler(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(ctxKeyLog{}).(logrus.FieldLogger)
	type Response struct {
		Message string `json:"message"`
	}

	type LLMResponse struct {
		Content string         `json:"content"`
		Details map[string]any `json:"details"`
	}

	var response LLMResponse

	url := "http://" + fe.shoppingAssistantSvcAddr
	req, err := http.NewRequest(http.MethodPost, url, r.Body)
	if err != nil {
		renderHTTPError(log, r, w, errors.Wrap(err, "failed to create request"), http.StatusInternalServerError)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		renderHTTPError(log, r, w, errors.Wrap(err, "failed to send request"), http.StatusInternalServerError)
		return
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		renderHTTPError(log, r, w, errors.Wrap(err, "failed to read response"), http.StatusInternalServerError)
		return
	}

	fmt.Printf("%+v\n", body)
	fmt.Printf("%+v\n", res)

	err = json.Unmarshal(body, &response)
	if err != nil {
		renderHTTPError(log, r, w, errors.Wrap(err, "failed to unmarshal body"), http.StatusInternalServerError)
		return
	}

	// respond with the same message
	json.NewEncoder(w).Encode(Response{Message: response.Content})

	w.WriteHeader(http.StatusOK)
}

// setCurrencyHandler processes HTTP requests to update the user's preferred currency.
//
// It parses and validates the submitted currency code against the payload validator,
// persists the new currency choice in an HTTP cookie, and redirects the client
// back to their referring page (or the home page if no referer is provided).
func (fe *frontendServer) setCurrencyHandler(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(ctxKeyLog{}).(logrus.FieldLogger)
	cur := r.FormValue("currency_code")
	payload := validator.SetCurrencyPayload{Currency: cur}
	if err := payload.Validate(); err != nil {
		renderHTTPError(log, r, w, validator.ValidationErrorResponse(err), http.StatusUnprocessableEntity)
		return
	}
	log.WithField("curr.new", payload.Currency).WithField("curr.old", currentCurrency(r)).
		Debug("setting currency")

	if payload.Currency != "" {
		http.SetCookie(w, &http.Cookie{
			Name:   cookieCurrency,
			Value:  payload.Currency,
			MaxAge: cookieMaxAge,
		})
	}
	referer := r.Header.Get("referer")
	if referer == "" {
		referer = baseUrl + "/"
	}
	w.Header().Set("Location", referer)
	w.WriteHeader(http.StatusFound)
}

// chooseAd queries for advertisements available and randomly chooses one, if
// available. It ignores the error retrieving the ad since it is not critical.
func (fe *frontendServer) chooseAd(ctx context.Context, ctxKeys []string, log logrus.FieldLogger) *adpb.Ad {
	ads, err := fe.getAd(ctx, ctxKeys)
	if err != nil {
		log.WithField("error", err).Warn("failed to retrieve ads")
		return nil
	}
	return ads[rand.Intn(len(ads))]
}

// renderHTTPError logs an HTTP request failure, sets the response status code,
// and renders the user-facing error template with the provided error details.
func renderHTTPError(log logrus.FieldLogger, r *http.Request, w http.ResponseWriter, err error, code int) {
	log.WithField("error", err).Error("request error")
	errMsg := fmt.Sprintf("%+v", err)

	w.WriteHeader(code)

	if templateErr := templates.ExecuteTemplate(w, "error", injectCommonTemplateData(r, map[string]interface{}{
		"error":       errMsg,
		"status_code": code,
		"status":      http.StatusText(code),
	})); templateErr != nil {
		log.Println(templateErr)
	}
}

// injectCommonTemplateData populates and returns a map containing common
// global template context variables, merging them with any route-specific payload data.
func injectCommonTemplateData(r *http.Request, payload map[string]any) map[string]any {
	data := map[string]any{
		"session_id":           sessionID(r),
		"request_id":           r.Context().Value(ctxKeyRequestID{}),
		"user_currency":        currentCurrency(r),
		"platform_css":         plat.css,
		"platform_name":        plat.provider,
		"is_cymbal_brand":      isCymbalBrand,
		"assistant_enabled":    assistantEnabled,
		"auth_enabled":         authEnabled,
		"inventory_enabled":    inventoryEnabled,
		"notification_enabled": notificationEnabled,
		"deploymentDetails":    deploymentDetailsMap,
		"frontendMessage":      frontendMessage,
		"currentYear":          time.Now().Year(),
		"baseUrl":              baseUrl,
	}

	for k, v := range payload {
		data[k] = v
	}

	return data
}

// currentCurrency retrieves the user's selected currency from the request cookie,
// falling back to the default system currency if the cookie is missing or unset.
func currentCurrency(r *http.Request) string {
	c, _ := r.Cookie(cookieCurrency)
	if c != nil {
		return c.Value
	}
	return defaultCurrency
}

// sessionID retrieves the current session id from the request cookie,
// falling back to an empty string if the cookie is missing or unset.
func sessionID(r *http.Request) string {
	v := r.Context().Value(ctxKeySessionID{})
	if v != nil {
		return v.(string)
	}
	return ""
}

// cartIDs extracts product IDs from a slice of cart items.
func cartIDs(c []*cartpb.CartItem) []string {
	out := make([]string, len(c))
	for i, v := range c {
		out[i] = v.GetProductId()
	}
	return out
}

// cartSize calculates the total number of items in a shopping cart.
func cartSize(c []*cartpb.CartItem) int {
	cartSize := 0
	for _, item := range c {
		cartSize += int(item.GetQuantity())
	}
	return cartSize
}

// renderMoney formats a monetary amount into a human-readable string representation,
// combining the appropriate currency symbol, units, and fractional nano-precision value.
func renderMoney(money *commonpb.Money) string {
	currencyLogo := renderCurrencyLogo(money.GetCurrencyCode())
	return fmt.Sprintf("%s%d.%02d", currencyLogo, money.GetUnits(), money.GetNanos()/10000000)
}

// renderCurrencyLogo maps a given currency code to its corresponding symbol,
// defaulting to the US dollar sign if the code is unrecognized.
func renderCurrencyLogo(currencyCode string) string {
	logos := map[string]string{
		"USD": "$",
		"CAD": "$",
		"JPY": "¥",
		"EUR": "€",
		"TRY": "₺",
		"GBP": "£",
	}

	logo := "$" //default
	if val, ok := logos[currencyCode]; ok {
		logo = val
	}
	return logo
}

func stringinSlice(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

// calculateOrderTotal computes the aggregate monetary sum of a slice of order items,
// multiplying each item's cost by its quantity, normalizing fractional nanos,
// and returning the total in the matching currency.
func calculateOrderTotal(items []*checkoutpb.OrderItem) *commonpb.Money {
	if len(items) == 0 {
		return &commonpb.Money{}
	}

	var totalUnits int64
	var totalNanos int64
	currency := items[0].GetCost().GetCurrencyCode()

	for _, item := range items {
		cost := item.GetCost()
		quantity := int64(item.GetItem().GetQuantity())
		totalUnits += cost.GetUnits() * quantity
		totalNanos += int64(cost.GetNanos()) * quantity
	}

	totalUnits += int64(totalNanos / 1_000_000_000)
	totalNanos = totalNanos % 1_000_000_000

	if totalNanos < 0 {
		log.WithField("totalNanos", totalNanos).Warn("totalNanos is negative")
		log.WithField("totalUnits", totalUnits).Warn("totalUnits")
		log.WithField("items", items).Warn("items in order")
	}

	return &commonpb.Money{
		CurrencyCode: currency,
		Units:        totalUnits,
		Nanos:        int32(totalNanos),
	}
}

// severityFromStock maps an inventory stock count to a corresponding severity status level.
func severityFromStock(stock int64) string {
	switch {
	case stock == 0:
		return "critical"
	case stock < 5:
		return "critical"
	case stock < 10:
		return "low"
	default:
		return "available"
	}
}
