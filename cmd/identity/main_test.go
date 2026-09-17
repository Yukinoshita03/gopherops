package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadJWTComponentsFromEnv(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate test RSA key: %v", err)
	}
	keyFile := filepath.Join(t.TempDir(), "identity-private.pem")
	encodedKey := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: mustMarshalPKCS8Key(t, privateKey),
	})
	if err := os.WriteFile(keyFile, encodedKey, 0o600); err != nil {
		t.Fatalf("write test key file: %v", err)
	}

	t.Setenv("IDENTITY_JWT_PRIVATE_KEY_FILE", keyFile)
	t.Setenv("IDENTITY_JWT_ISSUER", "gopherops-identity")
	t.Setenv("IDENTITY_JWT_AUDIENCE", "gopherops-api")
	t.Setenv("IDENTITY_JWT_TTL", "5m")

	signer, verifier, err := loadJWTComponentsFromEnv()
	if err != nil {
		t.Fatalf("loadJWTComponentsFromEnv() error = %v", err)
	}
	rawToken, expiresAt, err := signer.Sign("user-123")
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}
	claims, err := verifier.Verify(rawToken)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if claims.Subject != "user-123" || claims.ExpiresAt != expiresAt.Unix() {
		t.Fatalf("verified claims = %+v, expiresAt = %v", claims, expiresAt)
	}
}

func TestLoadRSAPrivateKeySupportsPKCS1(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate test RSA key: %v", err)
	}
	keyFile := filepath.Join(t.TempDir(), "identity-private-pkcs1.pem")
	encodedKey := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})
	if err := os.WriteFile(keyFile, encodedKey, 0o600); err != nil {
		t.Fatalf("write test key file: %v", err)
	}

	loaded, err := loadRSAPrivateKey(keyFile)
	if err != nil {
		t.Fatalf("loadRSAPrivateKey() error = %v", err)
	}
	if !loaded.Equal(privateKey) {
		t.Fatal("loaded private key differs from the key written to the PEM file")
	}
}

func TestLoadRSAPrivateKeyRejectsInvalidPEM(t *testing.T) {
	keyFile := filepath.Join(t.TempDir(), "invalid.pem")
	if err := os.WriteFile(keyFile, []byte("not a PEM key"), 0o600); err != nil {
		t.Fatalf("write invalid key file: %v", err)
	}
	if _, err := loadRSAPrivateKey(keyFile); err == nil {
		t.Fatal("loadRSAPrivateKey() error = nil, want PEM error")
	}
}

func mustMarshalPKCS8Key(t *testing.T, privateKey *rsa.PrivateKey) []byte {
	t.Helper()
	encodedKey, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatalf("marshal test private key: %v", err)
	}
	return encodedKey
}
