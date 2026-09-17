// Package rs256 提供基于 RSA-SHA256 的 JWT 验证器。
package rs256

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"strings"
	"time"

	"github.com/Yukinoshita03/gopherops/internal/identity/domain"
	identitytoken "github.com/Yukinoshita03/gopherops/internal/identity/token"
)

var (
	// ErrInvalidVerifierConfig 表示验证器配置缺少必要信息或包含无效值。
	ErrInvalidVerifierConfig = errors.New("invalid RS256 verifier config")
	// ErrInvalidToken 表示令牌格式、签名或声明校验失败。
	ErrInvalidToken = errors.New("invalid JWT")
)

// RS256Verifier 保存验证 JWT 所需的可信配置。
// 公钥、签发方和受众来自服务端配置，不能由令牌内容决定。
type RS256Verifier struct {
	publicKey *rsa.PublicKey
	issuer    string
	audience  string
	clockSkew time.Duration
}

// NewRS256Verifier 根据服务端配置创建 RS256 验证器。
func NewRS256Verifier(cfg identitytoken.VerifierConfig) (*RS256Verifier, error) {
	if cfg.PublicKey == nil ||
		cfg.PublicKey.N == nil ||
		cfg.PublicKey.N.Sign() <= 0 ||
		cfg.PublicKey.N.BitLen() < 2048 ||
		cfg.PublicKey.E < 3 ||
		cfg.PublicKey.E%2 == 0 ||
		cfg.Issuer == "" ||
		cfg.Audience == "" ||
		cfg.ClockSkew < 0 {
		return nil, ErrInvalidVerifierConfig
	}

	return &RS256Verifier{
		publicKey: &rsa.PublicKey{
			N: new(big.Int).Set(cfg.PublicKey.N),
			E: cfg.PublicKey.E,
		},
		issuer:    cfg.Issuer,
		audience:  cfg.Audience,
		clockSkew: cfg.ClockSkew,
	}, nil
}

// Verify 解析紧凑格式 JWT，固定只接受 RS256，验证签名和标准声明，
// 全部通过后才返回 Payload。
func (v *RS256Verifier) Verify(rawToken string) (domain.Payload, error) {
	if v == nil || v.publicKey == nil {
		return domain.Payload{}, ErrInvalidToken
	}

	parts := strings.Split(rawToken, ".")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return domain.Payload{}, ErrInvalidToken
	}

	decodeSegment := base64.RawURLEncoding.Strict().DecodeString
	headerJSON, err := decodeSegment(parts[0])
	if err != nil {
		return domain.Payload{}, ErrInvalidToken
	}

	var header struct {
		Algorithm string `json:"alg"`
	}
	if err := json.Unmarshal(headerJSON, &header); err != nil || header.Algorithm != "RS256" {
		return domain.Payload{}, ErrInvalidToken
	}

	signature, err := decodeSegment(parts[2])
	if err != nil {
		return domain.Payload{}, ErrInvalidToken
	}
	message := parts[0] + "." + parts[1]
	digest := sha256.Sum256([]byte(message))
	if err := rsa.VerifyPKCS1v15(v.publicKey, crypto.SHA256, digest[:], signature); err != nil {
		return domain.Payload{}, ErrInvalidToken
	}

	payloadJSON, err := decodeSegment(parts[1])
	if err != nil {
		return domain.Payload{}, ErrInvalidToken
	}
	var claims struct {
		Subject string `json:"sub"`
		Issuer  string `json:"iss"`
		// 当前 Payload 约定单个字符串受众，不接受 aud 数组格式。
		Audience  string `json:"aud"`
		IssuedAt  *int64 `json:"iat"`
		ExpiresAt *int64 `json:"exp"`
		NotBefore *int64 `json:"nbf"`
	}
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return domain.Payload{}, ErrInvalidToken
	}
	if claims.Subject == "" ||
		claims.Issuer != v.issuer ||
		claims.Audience != v.audience ||
		claims.IssuedAt == nil ||
		claims.ExpiresAt == nil ||
		*claims.IssuedAt >= *claims.ExpiresAt {
		return domain.Payload{}, ErrInvalidToken
	}
	if claims.NotBefore != nil && *claims.NotBefore >= *claims.ExpiresAt {
		return domain.Payload{}, ErrInvalidToken
	}

	now := time.Now()
	if time.Unix(*claims.IssuedAt, 0).After(now.Add(v.clockSkew)) ||
		!now.Before(time.Unix(*claims.ExpiresAt, 0).Add(v.clockSkew)) ||
		(claims.NotBefore != nil && now.Add(v.clockSkew).Before(time.Unix(*claims.NotBefore, 0))) {
		return domain.Payload{}, ErrInvalidToken
	}

	return domain.Payload{
		Subject:   claims.Subject,
		Issuer:    claims.Issuer,
		Audience:  claims.Audience,
		IssuedAt:  *claims.IssuedAt,
		ExpiresAt: *claims.ExpiresAt,
	}, nil
}

// 编译期检查 RS256Verifier 实现了 identity token 层定义的接口。
var _ identitytoken.Verifier = (*RS256Verifier)(nil)
