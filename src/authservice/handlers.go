package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-ldap/ldap/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
	pb "github.com/turt1z/microservices-demo/src/authservice/genproto"
	"google.golang.org/grpc/codes"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

var log *logrus.Logger

func init() {
	log = logrus.New()
	log.Level = logrus.DebugLevel
	log.Formatter = &logrus.JSONFormatter{
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "severity",
			logrus.FieldKeyMsg:   "message",
		},
		TimestampFormat: time.RFC3339Nano,
	}
	log.Out = os.Stdout
}

func (as *AuthServer) Check(ctx context.Context, req *healthpb.HealthCheckRequest) (*healthpb.HealthCheckResponse, error) {
	return &healthpb.HealthCheckResponse{Status: healthpb.HealthCheckResponse_SERVING}, nil
}

func (as *AuthServer) Watch(req *healthpb.HealthCheckRequest, ws healthpb.Health_WatchServer) error {
	return status.Errorf(codes.Unimplemented, "health check via Watch not implemented")
}

func (as *AuthServer) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	if req.Username == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "username and password required")
	}
	log.Infof("Received login request for user: %s", req.Username)

	userDetails, err := as.lookupUser(req.Username)
	if err != nil {
		log.Errorf("User lookup failed: %v", err)
		return nil, status.Error(codes.Unauthenticated, "invalid username or password")
	}

	err = as.verifyPassword(userDetails.DN, req.Password)
	if err != nil {
		log.Errorf("User authentication failed: %v", err)
		return nil, status.Error(codes.Unauthenticated, "invalid username or password")
	}

	claims := &UserClaims{
		UserID:   req.Username,
		Username: req.Username,
		Roles:    userDetails.Roles,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt: jwt.NewNumericDate(time.Now()),
			Issuer:   "auth-service.theonlineshop.com",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(as.privateKey)
	if err != nil {
		log.Infof("Failed to generate access token for user %s: %v", req.Username, err)
		return nil, status.Error(codes.Internal, "failed to generate access token")
	}

	return &pb.LoginResponse{
		Token: tokenString,
	}, nil
}

func (s *AuthServer) lookupUser(username string) (*UserDetails, error) {
	l, err := ldap.DialURL(s.ldapURL)
	if err != nil {
		return nil, err
	}
	defer l.Close()

	err = l.Bind(s.adminDN, s.adminPass)
	if err != nil {
		log.Errorf("Admin bind failed: %v", err)
		return nil, fmt.Errorf("admin bind failed: %w", err)
	}

	searchRequest := ldap.NewSearchRequest(
		s.baseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0,
		0,
		false,
		fmt.Sprintf("(uid=%s)", ldap.EscapeFilter(username)),
		[]string{"dn", "memberOf"},
		nil,
	)
	log.Infof("Executing search request: %s", searchRequest)

	sr, err := l.Search(searchRequest)
	if err != nil {
		log.Errorf("User lookup failed: %v", err)
		return nil, err
	}

	log.Infof("User lookup completed: %s", sr.Entries)

	if len(sr.Entries) != 1 {
		log.Errorf("User not found or duplicate entries matched")
		return nil, fmt.Errorf("user not found or duplicate entries matched")
	}

	userEntry := sr.Entries[0]

	rawGroups := userEntry.GetAttributeValues("memberOf")
	log.Infof("Found raw groups for user %s: %v", userEntry.DN, rawGroups)
	var roles []string
	for _, groupDN := range rawGroups {
		parts := strings.Split(groupDN, ",")
		if len(parts) > 0 && strings.HasPrefix(strings.ToLower(parts[0]), "cn=") {
			roles = append(roles, strings.TrimPrefix(parts[0], "cn="))
		}
	}

	return &UserDetails{
		DN:    userEntry.DN,
		Roles: roles,
	}, nil
}

func (s *AuthServer) verifyPassword(userDN string, password string) error {
	l, err := ldap.DialURL(s.ldapURL)
	if err != nil {
		return err
	}
	defer l.Close()

	log.Infof("Attempting to bind user: %s", userDN)
	err = l.Bind(userDN, password)
	if err != nil {
		return fmt.Errorf("user authentication failed: %w", err)
	}

	log.Info("User authentication successful")
	return nil
}
