package main

import (
	"crypto/rsa"
	"net"
	"os"

	"github.com/golang-jwt/jwt/v5"
	authpb "github.com/netd-tud/ds-onlineshop/src/authservice/genproto/auth"
	shared "github.com/netd-tud/ds-onlineshop/src/shared"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

type AuthServer struct {
	authpb.UnimplementedAuthServiceServer
	privateKey *rsa.PrivateKey

	ldapURL   string
	adminDN   string
	adminPass string
	baseDN    string

	port string
}

func main() {
	var privateKeyPath string
	shared.MustMapEnv(&privateKeyPath, "JWT_PRIVATE_KEY_PATH")

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
	shared.MustMapEnv(&svc.ldapURL, "LDAP_SERVICE_ADDR")
	shared.MustMapEnv(&svc.adminDN, "LDAP_ADMIN_DN")
	shared.MustMapEnv(&svc.adminPass, "LDAP_ADMIN_PASS")
	shared.MustMapEnv(&svc.baseDN, "LDAP_BASE_DN")
	shared.MustMapEnv(&svc.port, "PORT")

	log.Infof("Server Config: %s", svc)

	lis, err := net.Listen("tcp", ":"+svc.port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	srv := grpc.NewServer()
	authpb.RegisterAuthServiceServer(srv, svc)
	healthcheck := health.NewServer()
	healthpb.RegisterHealthServer(srv, healthcheck)

	log.Println("Auth Service running on port " + svc.port + "...")
	if err := srv.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
