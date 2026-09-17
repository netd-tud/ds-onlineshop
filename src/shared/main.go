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

// contextKey is a custom string type used for defining context keys to prevent collisions across packages.
type contextKey string

// UserContextKey is the context key used to store and retrieve parsed UserClaims within the gRPC request context.
const UserContextKey contextKey = "user_info"

// CtxKeyLoadTest is an empty struct type used as a context key to store load-testing flags in the context.
type CtxKeyLoadTest struct{}

// LoadTestHeaderName is the HTTP/gRPC metadata header key used to propagate load-testing flags across microservices.
const LoadTestHeaderName = "x-load-test"

// UserClaims defines the JWT claims structure for end-user authorization.
//
// It encapsulates identity attributes, assigned user roles, and standard registered JWT claims.
type UserClaims struct {
	UserID   string   `json:"user_id"`
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
	Title    string   `json:"title"`
	Name     string   `json:"name"`
	jwt.RegisteredClaims
}

var systemJWTSecret = []byte(os.Getenv("SYSTEM_JWT_SECRET"))

// SystemClaims defines the JWT claims structure used for inter-service HMAC authentication.
//
// It captures the identifying service name, elevated service roles, and standard registered JWT claims.
type SystemClaims struct {
	ServiceName string   `json:"service_name"`
	Roles       []string `json:"roles"`
	jwt.RegisteredClaims
}

// Category represents a string-based classification for inventory and catalog product grouping.
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

// Permission specifies the access level (Read or Write) granted for a specific domain resource.
type Permission int

const (
	PermissionRead Permission = iota
	PermissionWrite
)

// CategoryAccess pairs an inventory Category with its associated access Permission level.
type CategoryAccess struct {
	Category   Category
	Permission Permission
}

// CategoriesForClaims maps RBAC role strings to their corresponding CategoryAccess permission rules.
//
// It defines which product categories a role can access and whether read or write privileges are granted.
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

// CurrenciesForClaims maps RBAC role strings to their authorized ISO currency codes.
//
// It restricts multi-currency operational permissions based on assigned user roles.
var CurrenciesForClaims = map[string][]string{
	"admins":       {"USD", "EUR", "GBP", "JPY", "CAD", "TRY"},
	"currency-usd": {"USD"},
	"currency-eur": {"EUR"},
	"currency-gbp": {"GBP"},
	"currency-jpy": {"JPY"},
	"currency-cad": {"CAD"},
	"currency-try": {"TRY"},
}

// defaultPublicMethods defines a set of standard gRPC full method names that bypass authentication.
//
// By default, health check and health watch endpoints are marked public to permit unauthenticated health probes.
var defaultPublicMethods = map[string]struct{}{
	healthpb.Health_Check_FullMethodName: {},
	healthpb.Health_Watch_FullMethodName: {},
	healthpb.Health_List_FullMethodName:  {},
}

// NewAuthInterceptor constructs a gRPC UnaryServerInterceptor that enforces JWT bearer token authentication.
//
// It validates incoming RSA public-key user tokens or HMAC system tokens against exempt methods,
// populating the request context with parsed claims upon success.
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

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
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
		rawToken := tokenParts[1]

		claims := &UserClaims{}
		token, err := jwt.ParseWithClaims(rawToken, claims, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return pubKey, nil
		}, jwt.WithLeeway(5*time.Second))

		if err == nil && token.Valid {
			return handler(context.WithValue(ctx, UserContextKey, claims), req)
		}

		if len(systemJWTSecret) > 0 {
			sysClaims := &SystemClaims{}
			sysToken, err := jwt.ParseWithClaims(rawToken, sysClaims, func(t *jwt.Token) (any, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
				}
				return systemJWTSecret, nil
			}, jwt.WithLeeway(5*time.Second))

			if err == nil && sysToken.Valid {
				userClaims := &UserClaims{
					UserID:           sysClaims.ServiceName,
					Username:         sysClaims.ServiceName,
					Roles:            sysClaims.Roles,
					Title:            "SYSTEM",
					Name:             sysClaims.ServiceName,
					RegisteredClaims: sysClaims.RegisteredClaims,
				}
				return handler(context.WithValue(ctx, UserContextKey, userClaims), req)
			}
		}

		return nil, status.Error(codes.Unauthenticated, "invalid or expired token")
	}
}

// LoadTestInterceptor constructs a gRPC UnaryClientInterceptor that propagates load-testing markers via outgoing metadata.
//
// It inspects the context for a CtxKeyLoadTest value and attaches the x-load-test header to downstream gRPC calls.
func LoadTestInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if loadTest, ok := ctx.Value(CtxKeyLoadTest{}).(string); ok {
			ctx = metadata.AppendToOutgoingContext(ctx, LoadTestHeaderName, loadTest)
		}

		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// GetClaims retrieves the parsed UserClaims from the provided context.
//
// It returns the user claims struct pointer and a boolean indicating whether valid claims were present.
func GetClaims(ctx context.Context) (*UserClaims, bool) {
	claims, ok := ctx.Value(UserContextKey).(*UserClaims)
	return claims, ok
}

// ClaimsToCategories maps assigned user roles to their corresponding slice of allowed CategoryAccess permissions.
func ClaimsToCategories(claims *UserClaims) []CategoryAccess {
	var categoryAccesses []CategoryAccess
	for _, role := range claims.Roles {
		if access, ok := CategoriesForClaims[role]; ok {
			categoryAccesses = append(categoryAccesses, access)
		}
	}
	return categoryAccesses
}

// ClaimsToCurrencies maps assigned user roles to their corresponding slice of supported ISO currency codes.
func ClaimsToCurrencies(claims *UserClaims) []string {
	var currencies []string
	for _, role := range claims.Roles {
		if access, ok := CurrenciesForClaims[role]; ok {
			currencies = append(currencies, access...)
		}
	}
	return currencies
}

// MustMapEnv populates the target string pointer with the value of the specified environment variable.
//
// It panics if the environment variable is unset or contains an empty string.
func MustMapEnv(target *string, envKey string) {
	v := os.Getenv(envKey)
	if v == "" {
		panic(fmt.Sprintf("environment variable %q not set", envKey))
	}
	*target = v
}

// MustConnGRPC initializes an insecure gRPC client connection with OpenTelemetry stats handlers and load-test interceptors.
//
// It assigns the created connection to the provided pointer and panics if the connection setup fails.
func MustConnGRPC(ctx context.Context, conn **grpc.ClientConn, addr string) {
	var err error
	*conn, err = grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
		grpc.WithUnaryInterceptor(LoadTestInterceptor()))
	if err != nil {
		panic(errors.Wrapf(err, "grpc: failed to connect %s", addr))
	}
}

// ContextWithLoadTest wraps the provided context with a load-testing flag value.
func ContextWithLoadTest(ctx context.Context, loadTest string) context.Context {
	return context.WithValue(ctx, CtxKeyLoadTest{}, loadTest)
}

// IsLoadTest checks whether the current execution context or incoming metadata denotes a load-test invocation.
//
// It returns true if either the context key or incoming 'x-load-test' metadata header equals "true".
func IsLoadTest(ctx context.Context) bool {
	if loadTest, ok := ctx.Value(CtxKeyLoadTest{}).(string); ok && loadTest == "true" {
		return true
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if ok && len(md[LoadTestHeaderName]) > 0 {
		if md[LoadTestHeaderName][0] == "true" {
			return true
		}
	}

	return false
}

// GenerateSystemToken creates a short-lived, HMAC-signed system JWT for inter-service authentication.
//
// It returns the signed JWT string or an error if the SYSTEM_JWT_SECRET environment variable is missing.
func GenerateSystemToken(serviceName string) (string, error) {
	if len(systemJWTSecret) == 0 {
		return "", fmt.Errorf("SYSTEM_JWT_SECRET not set")
	}

	claims := SystemClaims{
		ServiceName: serviceName,
		Roles:       []string{"SYSTEM_SERVICE"},
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(5 * time.Minute)),
			Issuer:    "system",
			Subject:   serviceName,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(systemJWTSecret)
}

// ValidateSystemToken verifies and parses an HMAC-signed system JWT string.
//
// It checks the signing method and signature validity, returning the parsed SystemClaims or an error.
func ValidateSystemToken(tokenString string) (*SystemClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &SystemClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return systemJWTSecret, nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*SystemClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, fmt.Errorf("invalid system token")
}
