package domain

type Header struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}
type Payload struct {
	Subject  string `json:"sub"`
	Issuer   string `json:"iss"`
	Audience string `json:"aud"`
	// IssuedAt 和 ExpiresAt 是以 Unix 秒表示的 NumericDate。
	IssuedAt  int64 `json:"iat"`
	ExpiresAt int64 `json:"exp"`
}

type JWT struct {
	Header  Header  `json:"header"`
	Payload Payload `json:"payload"`
}
