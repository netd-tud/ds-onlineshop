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
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	pb "github.com/turt1z/microservices-demo/src/frontend/genproto"
	"github.com/turt1z/microservices-demo/src/frontend/money"
	"github.com/turt1z/microservices-demo/src/frontend/validator"
	shared "github.com/turt1z/microservices-demo/src/shared"
)

type platformDetails struct {
	css      string
	provider string
}

type Ratings struct {
	Ratings []Rating `json:"ratings"`
	Average float32  `json:"average"`
}
type Rating struct {
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
	templates        = template.Must(template.New("").
				Funcs(template.FuncMap{
			"renderMoney":        renderMoney,
			"renderCurrencyLogo": renderCurrencyLogo,
		}).ParseGlob("templates/*.html"))
	plat platformDetails
)

var validEnvs = []string{"local", "gcp", "azure", "aws", "onprem", "alibaba"}

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

func computeHeavyLoad(iterations int) string {
	data := []byte("monitoring-load-test-payload-data")
	hash := sha256.Sum256(data)

	for i := 0; i < iterations; i++ {
		// Chain the hashes together to force serial CPU computation
		hash = sha256.Sum256(hash[:])
	}

	return fmt.Sprintf("%x", hash)
}

type Product struct {
	Item        *pb.Product
	Stock       int64
	Reorderable bool
}

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
		return
	}
	categoryAccess := shared.ClaimsToCategories(claims)

	products, _ := fe.getProducts(r.Context())
	inventoryProducts, _ := fe.listInventory(r.Context())

	combinedMap := make(map[string]*Product, len(products)+len(inventoryProducts))
	for _, product := range products {
		combinedMap[product.GetId()] = &Product{Item: product, Stock: 0, Reorderable: false}
	}
	for _, inventoryProduct := range inventoryProducts {
		if cp, ok := combinedMap[inventoryProduct.GetId()]; ok {
			cp.Stock = inventoryProduct.GetStock()
		} else {
			log.Warn("Could not find catalog Product corresponding to inventory Product with ID: %s", inventoryProduct.GetId())
		}
	}

	combinedList := make([]*Product, 0, len(combinedMap))
	for _, cp := range combinedMap {
		combinedList = append(combinedList, cp)
	}

	log.Infof("Claims: %s", claims)

	//categories := claimsToCategories(claims)

	log.Infof("User %s has access to categories: %v, combined inventory list has the following content: %v", claims.Username, categoryAccess, combinedList)

	filtered := combinedList
	if categoryAccess != nil {
		tmp := make([]*Product, 0, len(combinedList))
		for _, cp := range combinedList {
			if cp == nil || cp.Item == nil {
				continue
			}
			if slices.Contains(categoryAccess, shared.CategoryAccess{shared.CategoryAll, shared.PermissionWrite}) {
				tmp = append(tmp, cp)
				continue
			}
			for _, cat := range cp.Item.Categories {
				targetW := shared.CategoryAccess{shared.Category(cat), shared.PermissionWrite}
				targetRO := shared.CategoryAccess{shared.Category(cat), shared.PermissionRead}
				if slices.Contains(categoryAccess, targetW) {
					cp = &Product{Item: cp.Item, Stock: cp.Stock, Reorderable: true}
				} else if !slices.Contains(categoryAccess, targetRO) {
					break
				}
				tmp = append(tmp, cp)
			}
		}
		filtered = tmp
	}

	log.Infof("User %s has access to categories: %v, filtered inventory list has the following content: %v", claims.Username, categoryAccess, filtered)

	if err := templates.ExecuteTemplate(w, "reorder", injectCommonTemplateData(r, map[string]interface{}{
		"show_currency": false,
		"products":      filtered,
		"banner_color":  os.Getenv("BANNER_COLOR"),
	})); err != nil {
		log.Error(err)
	}
}

func (fe *frontendServer) claimsFromCookie(cookie *http.Cookie) (*shared.UserClaims, *jwt.Token, error) {
	tokenString := cookie.Value
	claims := &shared.UserClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		return fe.publicKey, nil
	})
	return claims, token, err
}

func claimsToCategories(claims *shared.UserClaims) []string {
	log.Infof("User %s has the following roles: %v", claims.Username, claims.Roles)
	var categories []string

	for _, role := range claims.Roles {
		switch role {
		case "admin":
			categories = append(categories, "all")
		case "inventory-accessories-manage":
			categories = append(categories, "accessories")
		case "inventory-clothing-manage":
			categories = append(categories, "clothing")
		}
	}

	return categories
}

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

		http.SetCookie(w, &http.Cookie{
			Name:     cookieAuth,
			Value:    "",
			MaxAge:   -1,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			// Secure: true,
		})
		http.Redirect(w, r, baseUrl+"/login", http.StatusFound)
		return
	}

	log.WithField("username", claims.Username).Info("valid token confirmed, directing to account")
	http.Redirect(w, r, baseUrl+"/account", http.StatusFound)
}

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

	nextTarget := r.FormValue("next")
	// Handle POST
	username := r.FormValue("uid")
	password := r.FormValue("password")

	log.WithField("username", username).Info("login attempt")
	resp, err := pb.NewAuthServiceClient(fe.authSvcConn).Login(r.Context(), &pb.LoginRequest{
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

	if nextTarget == "" || !strings.HasPrefix(nextTarget, "/") || strings.HasPrefix(nextTarget, "//") {
		nextTarget = "/"
	}

	http.Redirect(w, r, baseUrl+nextTarget, http.StatusFound)
}

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
	}

	if err := templates.ExecuteTemplate(w, "account", injectCommonTemplateData(r, templateData)); err != nil {
		log.Error(err)
	}
}

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
		Item  *pb.Product
		Price *pb.Money
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

	if err := templates.ExecuteTemplate(w, "home", injectCommonTemplateData(r, map[string]interface{}{
		"show_currency": true,
		"currencies":    currencies,
		"products":      ps,
		"cart_size":     cartSize(cart),
		"banner_color":  os.Getenv("BANNER_COLOR"), // illustrates canary deployments
		"ad":            fe.chooseAd(r.Context(), []string{}, log),
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
		Item    *pb.Product
		Price   *pb.Money
		Ratings *Ratings
		Stock   int64
	}{p, price, nil, -1}

	if fe.ratingSvcAddr != "" {
		resp, err := http.Get(fmt.Sprintf("http://%s/ratings/product/%s", fe.ratingSvcAddr, p.GetId()))
		log.Println("Response: %s", resp)
		if err == nil {
			var ratings Ratings
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

func (fe *frontendServer) ratingHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	score, _ := strconv.ParseFloat(r.FormValue("score"), 32)

	rating := Rating{
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

	if err := fe.insertCart(r.Context(), sessionID(r), p.GetId(), int32(payload.Quantity)); err != nil {
		renderHTTPError(log, r, w, errors.Wrap(err, "failed to add to cart"), http.StatusInternalServerError)
		return
	}
	w.Header().Set("location", baseUrl+"/cart")
	w.WriteHeader(http.StatusFound)
}

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
		Item     *pb.Product
		Quantity int32
		Price    *pb.Money
	}
	items := make([]cartItemView, len(cart))
	totalPrice := pb.Money{CurrencyCode: currentCurrency(r)}
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

	if err := templates.ExecuteTemplate(w, "cart", injectCommonTemplateData(r, map[string]interface{}{
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

	order, err := pb.NewCheckoutServiceClient(fe.checkoutSvcConn).
		PlaceOrder(r.Context(), &pb.PlaceOrderRequest{
			Email: payload.Email,
			CreditCard: &pb.CreditCardInfo{
				CreditCardNumber:          payload.CcNumber,
				CreditCardExpirationMonth: int32(payload.CcMonth),
				CreditCardExpirationYear:  int32(payload.CcYear),
				CreditCardCvv:             int32(payload.CcCVV)},
			UserId:       sessionID(r),
			UserCurrency: currentCurrency(r),
			Address: &pb.Address{
				StreetAddress: payload.StreetAddress,
				City:          payload.City,
				State:         payload.State,
				ZipCode:       int32(payload.ZipCode),
				Country:       payload.Country},
		})
	if err != nil {
		renderHTTPError(log, r, w, errors.Wrap(err, "failed to complete the order"), http.StatusInternalServerError)
		return
	}
	log.WithField("order", order.GetOrder().GetOrderId()).Info("order placed")

	order.GetOrder().GetItems()
	recommendations, _ := fe.getRecommendations(r.Context(), sessionID(r), nil)

	totalPaid := *order.GetOrder().GetShippingCost()
	for _, v := range order.GetOrder().GetItems() {
		multPrice := money.MultiplySlow(*v.GetCost(), uint32(v.GetItem().GetQuantity()))
		totalPaid = money.Must(money.Sum(totalPaid, multPrice))
	}

	currencies, err := fe.getCurrencies(r.Context())
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
func (fe *frontendServer) chooseAd(ctx context.Context, ctxKeys []string, log logrus.FieldLogger) *pb.Ad {
	ads, err := fe.getAd(ctx, ctxKeys)
	if err != nil {
		log.WithField("error", err).Warn("failed to retrieve ads")
		return nil
	}
	return ads[rand.Intn(len(ads))]
}

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

func injectCommonTemplateData(r *http.Request, payload map[string]interface{}) map[string]interface{} {
	data := map[string]interface{}{
		"session_id":        sessionID(r),
		"request_id":        r.Context().Value(ctxKeyRequestID{}),
		"user_currency":     currentCurrency(r),
		"platform_css":      plat.css,
		"platform_name":     plat.provider,
		"is_cymbal_brand":   isCymbalBrand,
		"assistant_enabled": assistantEnabled,
		"deploymentDetails": deploymentDetailsMap,
		"frontendMessage":   frontendMessage,
		"currentYear":       time.Now().Year(),
		"baseUrl":           baseUrl,
	}

	for k, v := range payload {
		data[k] = v
	}

	return data
}

func currentCurrency(r *http.Request) string {
	c, _ := r.Cookie(cookieCurrency)
	if c != nil {
		return c.Value
	}
	return defaultCurrency
}

func sessionID(r *http.Request) string {
	v := r.Context().Value(ctxKeySessionID{})
	if v != nil {
		return v.(string)
	}
	return ""
}

func cartIDs(c []*pb.CartItem) []string {
	out := make([]string, len(c))
	for i, v := range c {
		out[i] = v.GetProductId()
	}
	return out
}

// get total # of items in cart
func cartSize(c []*pb.CartItem) int {
	cartSize := 0
	for _, item := range c {
		cartSize += int(item.GetQuantity())
	}
	return cartSize
}

func renderMoney(money pb.Money) string {
	currencyLogo := renderCurrencyLogo(money.GetCurrencyCode())
	return fmt.Sprintf("%s%d.%02d", currencyLogo, money.GetUnits(), money.GetNanos()/10000000)
}

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
