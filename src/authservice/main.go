package main

import (
	"crypto/rsa"
	"fmt"
	"net"
	"os"

	"github.com/golang-jwt/jwt/v5"
	pb "github.com/turt1z/microservices-demo/src/authservice/genproto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

type AuthServer struct {
	pb.UnimplementedAuthServiceServer
	privateKey *rsa.PrivateKey

	ldapURL   string
	adminDN   string
	adminPass string
	baseDN    string

	port string
}

type UserDetails struct {
	DN    string
	Roles []string
}

type UserClaims struct {
	UserID   string   `json:"user_id"`
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
	jwt.RegisteredClaims
}

func main() {
	var privateKeyPath string
	mustMapEnv(&privateKeyPath, "JWT_PRIVATE_KEY_PATH")

	privKeyBytes, err := os.ReadFile(privateKeyPath)
	if err != nil {
		log.Fatalf("failed to read private key: %v", err)
	}
	privKey, err := jwt.ParseRSAPrivateKeyFromPEM(privKeyBytes)
	if err != nil {
		log.Fatalf("failed to parse private key: %v", err)
	}

	svc := &AuthServer{
		privateKey: privKey,
	}
	mustMapEnv(&svc.ldapURL, "LDAP_SERVICE_ADDR")
	mustMapEnv(&svc.adminDN, "LDAP_ADMIN_DN")
	mustMapEnv(&svc.adminPass, "LDAP_ADMIN_PASS")
	mustMapEnv(&svc.baseDN, "LDAP_BASE_DN")
	mustMapEnv(&svc.port, "PORT")

	log.Infof("Server Config: %s", svc)

	lis, err := net.Listen("tcp", ":"+svc.port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	srv := grpc.NewServer()
	pb.RegisterAuthServiceServer(srv, svc)
	healthcheck := health.NewServer()
	healthpb.RegisterHealthServer(srv, healthcheck)

	log.Println("Auth Service running on port " + svc.port + "...")
	if err := srv.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func mustMapEnv(target *string, envKey string) {
	v := os.Getenv(envKey)
	if v == "" {
		panic(fmt.Sprintf("environment variable %q not set", envKey))
	}
	*target = v
}
