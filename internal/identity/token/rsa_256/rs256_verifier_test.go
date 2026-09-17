package rs256

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
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

func TestRS256VerifierVerify(t *testing.T) {
	privateKey := mustGenerateTestKey(t)
	verifier := mustNewTestVerifier(t, &privateKey.PublicKey, 0)
	now := time.Now().Unix()

	validClaims := testClaims(now-60, now+60)
	validToken := signTestToken(t, privateKey, `{"alg":"RS256","typ":"JWT"}`, validClaims)

	t.Run("有效令牌返回声明", func(t *testing.T) {
		got, err := verifier.Verify(validToken)
		if err != nil {
			t.Fatalf("Verify() error = %v", err)
		}
		want := domain.Payload{
			Subject:   "user-123",
			Issuer:    "gopherops-identity",
			Audience:  "gopherops-api",
			IssuedAt:  now - 60,
			ExpiresAt: now + 60,
		}
		if got != want {
			t.Fatalf("Verify() payload = %+v, want %+v", got, want)
		}
	})

	wrongIssuer := testClaims(now-60, now+60)
	wrongIssuer["iss"] = "another-issuer"
	wrongAudience := testClaims(now-60, now+60)
	wrongAudience["aud"] = "another-api"
	audienceArray := testClaims(now-60, now+60)
	audienceArray["aud"] = []string{"gopherops-api"}
	invalidIssuedAt := testClaims(now-60, now+60)
	invalidIssuedAt["iat"] = "not-a-number"
	missingSubject := testClaims(now-60, now+60)
	delete(missingSubject, "sub")
	missingIssuedAt := testClaims(now-60, now+60)
	delete(missingIssuedAt, "iat")
	missingExpiration := testClaims(now-60, now+60)
	delete(missingExpiration, "exp")
	expired := testClaims(now-120, now-60)
	futureIssued := testClaims(now+60, now+120)
	futureNotBefore := testClaims(now-60, now+120)
	futureNotBefore["nbf"] = now + 60
	noLifetime := testClaims(now-60, now-60)

	cases := []struct {
		name    string
		token   string
		payload map[string]any
		header  string
	}{
		{name: "格式错误", token: "not-a-jwt"},
		{name: "存在空段", token: "a..c"},
		{name: "Base64URL 错误", token: "!.e30.AA"},
		{name: "头部 JSON 错误", header: `{"alg":`},
		{name: "拒绝 none 算法", header: `{"alg":"none"}`, payload: validClaims},
		{name: "拒绝其他算法", header: `{"alg":"HS256"}`, payload: validClaims},
		{name: "签名被篡改", token: tamperTestTokenSignature(t, validToken)},
		{name: "签发方不匹配", payload: wrongIssuer},
		{name: "受众不匹配", payload: wrongAudience},
		{name: "不接受数组受众", payload: audienceArray},
		{name: "主体缺失", payload: missingSubject},
		{name: "iat 缺失", payload: missingIssuedAt},
		{name: "iat 类型错误", payload: invalidIssuedAt},
		{name: "exp 缺失", payload: missingExpiration},
		{name: "令牌过期", payload: expired},
		{name: "iat 在未来", payload: futureIssued},
		{name: "nbf 在未来", payload: futureNotBefore},
		{name: "有效期为空", payload: noLifetime},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rawToken := tc.token
			if rawToken == "" {
				header := tc.header
				if header == "" {
					header = `{"alg":"RS256","typ":"JWT"}`
				}
				payload := tc.payload
				if payload == nil {
					payload = validClaims
				}
				rawToken = signTestToken(t, privateKey, header, payload)
			}

			got, err := verifier.Verify(rawToken)
			if !errors.Is(err, ErrInvalidToken) {
				t.Fatalf("Verify() error = %v, want %v", err, ErrInvalidToken)
			}
			if got != (domain.Payload{}) {
				t.Fatalf("Verify() payload on failure = %+v, want zero value", got)
			}
		})
	}
}

func TestRS256VerifierHonorsClockSkew(t *testing.T) {
	privateKey := mustGenerateTestKey(t)
	verifier := mustNewTestVerifier(t, &privateKey.PublicKey, time.Minute)
	now := time.Now().Unix()

	cases := []struct {
		name    string
		payload map[string]any
		wantErr bool
	}{
		{
			name:    "過期時間在容差内",
			payload: testClaims(now-120, now-10),
		},
		{
			name:    "過期時間超出容差",
			payload: testClaims(now-180, now-120),
			wantErr: true,
		},
		{
			name:    "簽發時間在容差内",
			payload: testClaims(now+10, now+120),
		},
		{
			name:    "簽發時間超出容差",
			payload: testClaims(now+120, now+180),
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			token := signTestToken(t, privateKey, `{"alg":"RS256","typ":"JWT"}`, tc.payload)
			_, err := verifier.Verify(token)
			if tc.wantErr && !errors.Is(err, ErrInvalidToken) {
				t.Fatalf("Verify() error = %v, want %v", err, ErrInvalidToken)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("Verify() error = %v, want nil", err)
			}
		})
	}
}

func TestNewRS256VerifierRejectsInvalidConfig(t *testing.T) {
	privateKey := mustGenerateTestKey(t)
	valid := identitytoken.VerifierConfig{
		PublicKey: &privateKey.PublicKey,
		Issuer:    "gopherops-identity",
		Audience:  "gopherops-api",
	}

	cases := []struct {
		name string
		edit func(*identitytoken.VerifierConfig)
	}{
		{name: "公钥缺失", edit: func(cfg *identitytoken.VerifierConfig) { cfg.PublicKey = nil }},
		{name: "公钥模数缺失", edit: func(cfg *identitytoken.VerifierConfig) { cfg.PublicKey = &rsa.PublicKey{E: 65537} }},
		{name: "RSA 密钥过短", edit: func(cfg *identitytoken.VerifierConfig) { cfg.PublicKey = &rsa.PublicKey{N: big.NewInt(7), E: 3} }},
		{name: "签发方缺失", edit: func(cfg *identitytoken.VerifierConfig) { cfg.Issuer = "" }},
		{name: "受众缺失", edit: func(cfg *identitytoken.VerifierConfig) { cfg.Audience = "" }},
		{name: "时钟偏差为负数", edit: func(cfg *identitytoken.VerifierConfig) { cfg.ClockSkew = -time.Second }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := valid
			tc.edit(&cfg)
			if _, err := NewRS256Verifier(cfg); !errors.Is(err, ErrInvalidVerifierConfig) {
				t.Fatalf("NewRS256Verifier() error = %v, want %v", err, ErrInvalidVerifierConfig)
			}
		})
	}
}

func mustGenerateTestKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate test RSA key: %v", err)
	}
	return key
}

func mustNewTestVerifier(t *testing.T, publicKey *rsa.PublicKey, clockSkew time.Duration) *RS256Verifier {
	t.Helper()
	verifier, err := NewRS256Verifier(identitytoken.VerifierConfig{
		PublicKey: publicKey,
		Issuer:    "gopherops-identity",
		Audience:  "gopherops-api",
		ClockSkew: clockSkew,
	})
	if err != nil {
		t.Fatalf("create test verifier: %v", err)
	}
	return verifier
}

func testClaims(issuedAt, expiresAt int64) map[string]any {
	return map[string]any{
		"sub": "user-123",
		"iss": "gopherops-identity",
		"aud": "gopherops-api",
		"iat": issuedAt,
		"exp": expiresAt,
	}
}

func signTestToken(t *testing.T, privateKey *rsa.PrivateKey, header string, claims map[string]any) string {
	t.Helper()
	payloadJSON, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("marshal test claims: %v", err)
	}
	message := base64.RawURLEncoding.EncodeToString([]byte(header)) + "." + base64.RawURLEncoding.EncodeToString(payloadJSON)
	digest := sha256.Sum256([]byte(message))
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatalf("sign test token: %v", err)
	}
	return message + "." + base64.RawURLEncoding.EncodeToString(signature)
}

func tamperTestTokenSignature(t *testing.T, token string) string {
	t.Helper()
	parts := strings.Split(token, ".")
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		t.Fatalf("decode test signature: %v", err)
	}
	signature[0] ^= 0xff
	parts[2] = base64.RawURLEncoding.EncodeToString(signature)
	return strings.Join(parts, ".")
}
