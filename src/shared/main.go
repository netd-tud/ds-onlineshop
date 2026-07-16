package auth

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pkg/errors"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type contextKey string

const UserContextKey contextKey = "user_info"

type UserClaims struct {
	UserID   string   `json:"user_id"`
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
	Title    string   `json:"title"`
	Name     string   `json:"name"`
	jwt.RegisteredClaims
}

type Category string

const (
	CategoryAll         Category = "all"
	CategoryAccessories Category = "accessories"
	CategoryClothing    Category = "clothing"
	CategoryTops        Category = "tops"
	CategoryFootwear    Category = "footwear"
	CategoryBeauty      Category = "beauty"
	CategoryHair        Category = "hair"
	CategoryDecor       Category = "decor"
	CategoryHome        Category = "home"
	CategoryKitchen     Category = "kitchen"
)

type Permission int

const (
	PermissionRead Permission = iota
	PermissionWrite
)

type CategoryAccess struct {
	Category   Category
	Permission Permission
}

var CategoriesForClaims = map[string]CategoryAccess{
	"admins":                       {CategoryAll, PermissionWrite},
	"inventory-accessories-view":   {CategoryAccessories, PermissionRead},
	"inventory-accessories-manage": {CategoryAccessories, PermissionWrite},
	"inventory-clothing-view":      {CategoryClothing, PermissionRead},
	"inventory-clothing-manage":    {CategoryClothing, PermissionWrite},
	"inventory-tops-view":          {CategoryTops, PermissionRead},
	"inventory-tops-manage":        {CategoryTops, PermissionWrite},
	"inventory-footwear-view":      {CategoryFootwear, PermissionRead},
	"inventory-footwear-manage":    {CategoryFootwear, PermissionWrite},
	"inventory-beauty-view":        {CategoryBeauty, PermissionRead},
	"inventory-beauty-manage":      {CategoryBeauty, PermissionWrite},
	"inventory-hair-view":          {CategoryHair, PermissionRead},
	"inventory-hair-manage":        {CategoryHair, PermissionWrite},
	"inventory-kitchen-view":       {CategoryKitchen, PermissionRead},
	"inventory-kitchen-manage":     {CategoryKitchen, PermissionWrite},
	"inventory-decor-view":         {CategoryDecor, PermissionRead},
	"inventory-decor-manage":       {CategoryDecor, PermissionWrite},
	"inventory-home-view":          {CategoryHome, PermissionRead},
	"inventory-home-manage":        {CategoryHome, PermissionWrite},
}

var defaultPublicMethods = map[string]struct{}{
	healthpb.Health_Check_FullMethodName: {},
	healthpb.Health_Watch_FullMethodName: {},
	healthpb.Health_List_FullMethodName:  {},
}

func NewAuthInterceptor(publicKeyPEM []byte, publicMethods ...string) grpc.UnaryServerInterceptor {
	pubKey, err := jwt.ParseRSAPublicKeyFromPEM(publicKeyPEM)
	if err != nil {
		panic(fmt.Sprintf("failed to parse public key: %v", err))
	}

	exempt := make(map[string]struct{}, len(defaultPublicMethods)+len(publicMethods))
	for k := range defaultPublicMethods {
		exempt[k] = struct{}{}
	}
	for _, m := range publicMethods {
		exempt[m] = struct{}{}
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if _, ok := exempt[info.FullMethod]; ok {
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "metadata is missing")
		}

		authHeader := md.Get("authorization")
		if len(authHeader) == 0 {
			return nil, status.Error(codes.Unauthenticated, "authorization token is missing")
		}

		tokenParts := strings.SplitN(authHeader[0], " ", 2)
		if len(tokenParts) != 2 || strings.ToLower(tokenParts[0]) != "bearer" {
			return nil, status.Error(codes.Unauthenticated, "authorization header format must be Bearer <token>")
		}

		claims := &UserClaims{}
		token, err := jwt.ParseWithClaims(tokenParts[1], claims, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return pubKey, nil
		}, jwt.WithLeeway(5*time.Second))

		if err != nil || !token.Valid {
			return nil, status.Error(codes.Unauthenticated, "invalid or expired token")
		}

		return handler(context.WithValue(ctx, UserContextKey, claims), req)
	}
}

func GetClaims(ctx context.Context) (*UserClaims, bool) {
	claims, ok := ctx.Value(UserContextKey).(*UserClaims)
	return claims, ok
}

func ClaimsToCategories(claims *UserClaims) []CategoryAccess {
	var categories []CategoryAccess
	for _, role := range claims.Roles {
		if access, ok := CategoriesForClaims[role]; ok {
			categories = append(categories, access)
		}
	}
	return categories
}

func MustMapEnv(target *string, envKey string) {
	v := os.Getenv(envKey)
	if v == "" {
		panic(fmt.Sprintf("environment variable %q not set", envKey))
	}
	*target = v
}

func MustConnGRPC(ctx context.Context, conn **grpc.ClientConn, addr string) {
	var err error
	_, cancel := context.WithTimeout(ctx, time.Second*3)
	defer cancel()
	*conn, err = grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()))
	if err != nil {
		panic(errors.Wrapf(err, "grpc: failed to connect %s", addr))
	}
}
