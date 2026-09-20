// Command identity starts the identity HTTP service.
package main

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Yukinoshita03/gopherops/internal/identity/application/service"
	"github.com/Yukinoshita03/gopherops/internal/identity/repository"
	identitytoken "github.com/Yukinoshita03/gopherops/internal/identity/token"
	rs256 "github.com/Yukinoshita03/gopherops/internal/identity/token/rsa_256"
	"github.com/Yukinoshita03/gopherops/internal/identity/transport/httpapi"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	if err := run(); err != nil {
		log.Printf("identity service stopped: %v", err)
		os.Exit(1)
	}
}

func run() error {
	signer, verifier, err := loadJWTComponentsFromEnv()
	if err != nil {
		return err
	}

	dsn := os.Getenv("IDENTITY_MYSQL_DSN")
	if dsn == "" {
		return errors.New("IDENTITY_MYSQL_DSN is required")
	}

	addr := os.Getenv("IDENTITY_HTTP_ADDR")
	if addr == "" {
		addr = ":8081"
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("open identity database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get identity database connection: %w", err)
	}
	defer sqlDB.Close()

	pingCtx, cancelPing := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelPing()
	if err := sqlDB.PingContext(pingCtx); err != nil {
		return fmt.Errorf("ping identity database: %w", err)
	}

	userRepo := repository.NewMySQLUserRepository(db)
	projectMembershipRepo := repository.NewMySQLProjectMembershipRepository(db)
	registerService := service.NewRegisterService(userRepo)
	loginService := service.NewLoginService(userRepo, signer)
	projectAuthorizationService := service.NewProjectAuthorizationService(projectMembershipRepo)
	router := httpapi.NewRouter(registerService, loginService, verifier, projectAuthorizationService)

	log.Printf("identity service listening on %s", addr)
	if err := router.Run(addr); err != nil {
		return fmt.Errorf("run identity HTTP: %w", err)
	}
	return nil
}

func loadJWTComponentsFromEnv() (*rs256.RS256Signer, *rs256.RS256Verifier, error) {
	privateKeyPath := os.Getenv("IDENTITY_JWT_PRIVATE_KEY_FILE")
	if privateKeyPath == "" {
		return nil, nil, errors.New("IDENTITY_JWT_PRIVATE_KEY_FILE is required")
	}
	issuer := os.Getenv("IDENTITY_JWT_ISSUER")
	if issuer == "" {
		return nil, nil, errors.New("IDENTITY_JWT_ISSUER is required")
	}
	audience := os.Getenv("IDENTITY_JWT_AUDIENCE")
	if audience == "" {
		return nil, nil, errors.New("IDENTITY_JWT_AUDIENCE is required")
	}
	ttlValue := os.Getenv("IDENTITY_JWT_TTL")
	if ttlValue == "" {
		return nil, nil, errors.New("IDENTITY_JWT_TTL is required")
	}
	ttl, err := time.ParseDuration(ttlValue)
	if err != nil {
		return nil, nil, fmt.Errorf("parse IDENTITY_JWT_TTL: %w", err)
	}

	privateKey, err := loadRSAPrivateKey(privateKeyPath)
	if err != nil {
		return nil, nil, err
	}
	signer, err := rs256.NewRS256Signer(identitytoken.SignerConfig{
		PrivateKey: privateKey,
		Issuer:     issuer,
		Audience:   audience,
		TTL:        ttl,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("configure RS256 signer: %w", err)
	}
	verifier, err := rs256.NewRS256Verifier(identitytoken.VerifierConfig{
		PublicKey: &privateKey.PublicKey,
		Issuer:    issuer,
		Audience:  audience,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("configure RS256 verifier: %w", err)
	}
	return signer, verifier, nil
}

func loadRSAPrivateKey(path string) (*rsa.PrivateKey, error) {
	encodedKey, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read JWT private key file: %w", err)
	}
	defer clear(encodedKey)
	block, _ := pem.Decode(encodedKey)
	if block == nil {
		return nil, errors.New("JWT private key file is not valid PEM")
	}

	switch block.Type {
	case "RSA PRIVATE KEY":
		privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse RSA private key: %w", err)
		}
		return privateKey, nil
	case "PRIVATE KEY":
		parsedKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse PKCS#8 private key: %w", err)
		}
		privateKey, ok := parsedKey.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("JWT private key must be RSA")
		}
		return privateKey, nil
	default:
		return nil, errors.New("JWT private key PEM must contain an RSA private key")
	}
}
