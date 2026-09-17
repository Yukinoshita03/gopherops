// Package rs256 提供基于 RSA-SHA256 的 JWT 签发器和验证器。
package rs256

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/Yukinoshita03/gopherops/internal/identity/domain"
	identitytoken "github.com/Yukinoshita03/gopherops/internal/identity/token"
)

var (
	// ErrInvalidSignerConfig 表示签发器配置缺少必要信息或包含无效值。
	ErrInvalidSignerConfig = errors.New("invalid RS256 signer config")
	// ErrInvalidSubject 表示令牌主体为空。
	ErrInvalidSubject = errors.New("JWT subject is required")
)

// RS256Signer 持有签发令牌所需的私钥和可信声明配置。
type RS256Signer struct {
	privateKey *rsa.PrivateKey
	issuer     string
	audience   string
	ttl        time.Duration
}

// NewRS256Signer 根据服务端配置创建 RS256 签发器。
func NewRS256Signer(cfg identitytoken.SignerConfig) (*RS256Signer, error) {
	if cfg.PrivateKey == nil ||
		cfg.PrivateKey.N == nil ||
		cfg.PrivateKey.N.Sign() <= 0 ||
		cfg.PrivateKey.N.BitLen() < 2048 ||
		cfg.PrivateKey.D == nil ||
		cfg.PrivateKey.E < 3 ||
		cfg.PrivateKey.E%2 == 0 ||
		len(cfg.PrivateKey.Primes) < 2 ||
		cfg.Issuer == "" ||
		cfg.Audience == "" ||
		cfg.TTL < time.Second {
		return nil, ErrInvalidSignerConfig
	}
	for _, prime := range cfg.PrivateKey.Primes {
		if prime == nil || prime.Sign() <= 0 {
			return nil, ErrInvalidSignerConfig
		}
	}
	if err := cfg.PrivateKey.Validate(); err != nil {
		return nil, ErrInvalidSignerConfig
	}

	return &RS256Signer{
		privateKey: cfg.PrivateKey,
		issuer:     cfg.Issuer,
		audience:   cfg.Audience,
		ttl:        cfg.TTL,
	}, nil
}

// Sign 为主体创建紧凑格式的 RS256 JWT。
// iat 和 exp 使用 Unix 秒；过期时间由服务端配置的 TTL 计算。
func (s *RS256Signer) Sign(subject string) (string, time.Time, error) {
	if s == nil || s.privateKey == nil {
		return "", time.Time{}, ErrInvalidSignerConfig
	}
	if strings.TrimSpace(subject) == "" {
		return "", time.Time{}, ErrInvalidSubject
	}

	issuedAt := time.Now().UTC().Truncate(time.Second)
	expiresAt := issuedAt.Add(s.ttl).Truncate(time.Second)
	if !expiresAt.After(issuedAt) {
		return "", time.Time{}, ErrInvalidSignerConfig
	}

	headerJSON, err := json.Marshal(domain.Header{Alg: "RS256", Typ: "JWT"})
	if err != nil {
		return "", time.Time{}, err
	}
	payloadJSON, err := json.Marshal(domain.Payload{
		Subject:   subject,
		Issuer:    s.issuer,
		Audience:  s.audience,
		IssuedAt:  issuedAt.Unix(),
		ExpiresAt: expiresAt.Unix(),
	})
	if err != nil {
		return "", time.Time{}, err
	}

	encodedHeader := base64.RawURLEncoding.EncodeToString(headerJSON)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadJSON)
	signingInput := encodedHeader + "." + encodedPayload
	digest := sha256.Sum256([]byte(signingInput))
	signature, err := rsa.SignPKCS1v15(rand.Reader, s.privateKey, crypto.SHA256, digest[:])
	if err != nil {
		return "", time.Time{}, err
	}

	rawToken := signingInput + "." + base64.RawURLEncoding.EncodeToString(signature)
	return rawToken, expiresAt, nil
}

// 编译期检查 RS256Signer 实现了 identity token 层定义的接口。
var _ identitytoken.Signer = (*RS256Signer)(nil)
