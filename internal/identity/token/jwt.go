// Package token 定义 identity 使用的 JWT 签发与验证接口。
package token

import (
	"crypto/rsa"
	"time"

	"github.com/Yukinoshita03/gopherops/internal/identity/domain"
)

// SignerConfig 是构造 JWT 签发器时提供的一次性配置。
// 私钥只保存在 identity 服务中。
type SignerConfig struct {
	PrivateKey *rsa.PrivateKey
	Issuer     string
	Audience   string
	TTL        time.Duration
}

// VerifierConfig 是构造令牌验证器时提供的一次性配置。
// 验证服务只需要公钥、预期的签发方和受众。
type VerifierConfig struct {
	PublicKey *rsa.PublicKey
	Issuer    string
	Audience  string
	ClockSkew time.Duration
}

// Signer 为用户主体签发紧凑格式的 RS256 JWT。
// 签发方、受众、签发时间和过期时间由可信配置及签发器生成；调用者只传用户主体。
type Signer interface {
	Sign(subject string) (rawToken string, expiresAt time.Time, err error)
}

// Verifier 使用配置的公钥验证紧凑格式 JWT 的签名、签发方、受众和时间，验证通过后返回 claims。
type Verifier interface {
	Verify(rawToken string) (domain.Payload, error)
}
