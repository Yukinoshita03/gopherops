package rs256

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/Yukinoshita03/gopherops/internal/identity/domain"
	identitytoken "github.com/Yukinoshita03/gopherops/internal/identity/token"
)

func TestRS256SignerSignProducesVerifiableToken(t *testing.T) {
	privateKey := mustGenerateTestKey(t)
	signer, err := NewRS256Signer(identitytoken.SignerConfig{
		PrivateKey: privateKey,
		Issuer:     "gopherops-identity",
		Audience:   "gopherops-api",
		TTL:        5 * time.Minute,
	})
	if err != nil {
		t.Fatalf("NewRS256Signer() error = %v", err)
	}

	startedAt := time.Now().Unix()
	rawToken, expiresAt, err := signer.Sign("user-123")
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}
	if expiresAt.Location() != time.UTC || expiresAt.Nanosecond() != 0 {
		t.Fatalf("Sign() expiresAt = %v, want UTC with whole-second precision", expiresAt)
	}

	parts := strings.Split(rawToken, ".")
	if len(parts) != 3 {
		t.Fatalf("Sign() token has %d segments, want 3", len(parts))
	}
	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		t.Fatalf("decode JWT header: %v", err)
	}
	var header domain.Header
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		t.Fatalf("unmarshal JWT header: %v", err)
	}
	if header.Alg != "RS256" || header.Typ != "JWT" {
		t.Fatalf("JWT header = %+v, want RS256 JWT", header)
	}

	verifier := mustNewTestVerifier(t, &privateKey.PublicKey, 0)
	claims, err := verifier.Verify(rawToken)
	if err != nil {
		t.Fatalf("Verifier.Verify() error = %v", err)
	}
	if claims.Subject != "user-123" ||
		claims.Issuer != "gopherops-identity" ||
		claims.Audience != "gopherops-api" ||
		claims.IssuedAt < startedAt ||
		claims.IssuedAt > time.Now().Unix() ||
		claims.ExpiresAt != expiresAt.Unix() {
		t.Fatalf("verified claims = %+v, unexpected values", claims)
	}
}

func TestRS256SignerRejectsInvalidConfig(t *testing.T) {
	privateKey := mustGenerateTestKey(t)
	valid := identitytoken.SignerConfig{
		PrivateKey: privateKey,
		Issuer:     "gopherops-identity",
		Audience:   "gopherops-api",
		TTL:        5 * time.Minute,
	}

	cases := []struct {
		name string
		edit func(*identitytoken.SignerConfig)
	}{
		{name: "私钥缺失", edit: func(cfg *identitytoken.SignerConfig) { cfg.PrivateKey = nil }},
		{name: "私钥模数缺失", edit: func(cfg *identitytoken.SignerConfig) {
			cfg.PrivateKey = &rsa.PrivateKey{PublicKey: rsa.PublicKey{E: 65537}}
		}},
		{name: "RSA 密钥过短", edit: func(cfg *identitytoken.SignerConfig) {
			cfg.PrivateKey = &rsa.PrivateKey{
				PublicKey: rsa.PublicKey{N: big.NewInt(7), E: 3},
				D:         big.NewInt(3),
				Primes:    []*big.Int{big.NewInt(2), big.NewInt(3)},
			}
		}},
		{name: "签发方缺失", edit: func(cfg *identitytoken.SignerConfig) { cfg.Issuer = "" }},
		{name: "受众缺失", edit: func(cfg *identitytoken.SignerConfig) { cfg.Audience = "" }},
		{name: "有效期小于一秒", edit: func(cfg *identitytoken.SignerConfig) { cfg.TTL = time.Millisecond }},
		{name: "有效期为零", edit: func(cfg *identitytoken.SignerConfig) { cfg.TTL = 0 }},
		{name: "有效期为负数", edit: func(cfg *identitytoken.SignerConfig) { cfg.TTL = -time.Second }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := valid
			tc.edit(&cfg)
			if _, err := NewRS256Signer(cfg); !errors.Is(err, ErrInvalidSignerConfig) {
				t.Fatalf("NewRS256Signer() error = %v, want %v", err, ErrInvalidSignerConfig)
			}
		})
	}
}

func TestRS256SignerRejectsEmptySubject(t *testing.T) {
	privateKey := mustGenerateTestKey(t)
	signer, err := NewRS256Signer(identitytoken.SignerConfig{
		PrivateKey: privateKey,
		Issuer:     "gopherops-identity",
		Audience:   "gopherops-api",
		TTL:        time.Minute,
	})
	if err != nil {
		t.Fatalf("NewRS256Signer() error = %v", err)
	}

	for _, subject := range []string{"", " \t\n"} {
		if token, expiresAt, err := signer.Sign(subject); !errors.Is(err, ErrInvalidSubject) || token != "" || !expiresAt.IsZero() {
			t.Fatalf("Sign(%q) = (%q, %v, %v), want empty values and %v", subject, token, expiresAt, err, ErrInvalidSubject)
		}
	}
}
