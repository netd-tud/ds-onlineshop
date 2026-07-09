package auth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
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
	jwt.RegisteredClaims
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
