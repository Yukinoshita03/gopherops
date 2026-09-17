package httpapi_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Yukinoshita03/gopherops/internal/identity/application/port"
	"github.com/Yukinoshita03/gopherops/internal/identity/application/service"
	"github.com/Yukinoshita03/gopherops/internal/identity/domain"
	identitytoken "github.com/Yukinoshita03/gopherops/internal/identity/token"
	rs256 "github.com/Yukinoshita03/gopherops/internal/identity/token/rsa_256"
	"github.com/Yukinoshita03/gopherops/internal/identity/transport/httpapi"
	"golang.org/x/crypto/bcrypt"
)

type authIntegrationUserRepository struct {
	user *domain.User
}

func (r *authIntegrationUserRepository) Create(context.Context, *domain.User) error {
	return errors.New("Create should not be called by the login integration test")
}

var _ port.UserRepository = (*authIntegrationUserRepository)(nil)

func (r *authIntegrationUserRepository) GetByUsername(_ context.Context, username string) (*domain.User, error) {
	if r.user.Username != username {
		return nil, domain.ErrUserNotFound
	}
	return r.user, nil
}

func (r *authIntegrationUserRepository) GetByID(_ context.Context, id int64) (*domain.User, error) {
	if r.user.ID != id {
		return nil, domain.ErrUserNotFound
	}
	return r.user, nil
}

func TestLoginTokenAuthenticatesMeRoute(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate test RSA key: %v", err)
	}
	const issuer = "gopherops-identity-test"
	const audience = "gopherops-api-test"
	signer, err := rs256.NewRS256Signer(identitytoken.SignerConfig{
		PrivateKey: privateKey,
		Issuer:     issuer,
		Audience:   audience,
		TTL:        time.Minute,
	})
	if err != nil {
		t.Fatalf("create test signer: %v", err)
	}
	verifier, err := rs256.NewRS256Verifier(identitytoken.VerifierConfig{
		PublicKey: &privateKey.PublicKey,
		Issuer:    issuer,
		Audience:  audience,
	})
	if err != nil {
		t.Fatalf("create test verifier: %v", err)
	}

	const userID int64 = 42
	const username = "alice"
	const password = "integration-password"
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash test password: %v", err)
	}
	userRepo := &authIntegrationUserRepository{user: &domain.User{
		ID:           userID,
		Username:     username,
		PasswordHash: string(passwordHash),
	}}
	router := httpapi.NewRouter(
		service.NewRegisterService(userRepo),
		service.NewLoginService(userRepo, signer),
		verifier,
	)

	loginRequest := httptest.NewRequest(
		http.MethodPost,
		"/v1/auth/login",
		bytes.NewBufferString(`{"username":"alice","password":"integration-password"}`),
	)
	loginRequest.Header.Set("Content-Type", "application/json")
	loginRecorder := httptest.NewRecorder()
	router.ServeHTTP(loginRecorder, loginRequest)
	if loginRecorder.Code != http.StatusOK {
		t.Fatalf("login status = %d, want %d; body: %s", loginRecorder.Code, http.StatusOK, loginRecorder.Body.String())
	}
	var loginBody struct {
		UserID      int64     `json:"user_id"`
		AccessToken string    `json:"access_token"`
		ExpiresAt   time.Time `json:"expires_at"`
	}
	if err := json.Unmarshal(loginRecorder.Body.Bytes(), &loginBody); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if loginBody.UserID != userID || loginBody.AccessToken == "" || loginBody.ExpiresAt.IsZero() {
		t.Fatalf("login response = %+v, missing expected identity token fields", loginBody)
	}

	meRequest := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	meRequest.Header.Set("Authorization", "Bearer "+loginBody.AccessToken)
	meRecorder := httptest.NewRecorder()
	router.ServeHTTP(meRecorder, meRequest)
	if meRecorder.Code != http.StatusOK {
		t.Fatalf("/v1/me status = %d, want %d; body: %s", meRecorder.Code, http.StatusOK, meRecorder.Body.String())
	}
	var meBody struct {
		UserID int64 `json:"user_id"`
	}
	if err := json.Unmarshal(meRecorder.Body.Bytes(), &meBody); err != nil {
		t.Fatalf("decode /v1/me response: %v", err)
	}
	if meBody.UserID != userID {
		t.Fatalf("/v1/me user_id = %d, want %d", meBody.UserID, userID)
	}

	parts := bytes.Split([]byte(loginBody.AccessToken), []byte("."))
	if len(parts) != 3 {
		t.Fatalf("signed token has %d segments, want 3", len(parts))
	}
	parts[2][0] = 'A'
	tamperedToken := string(bytes.Join(parts, []byte(".")))
	if tamperedToken == loginBody.AccessToken {
		parts[2][0] = 'B'
		tamperedToken = string(bytes.Join(parts, []byte(".")))
	}
	tamperedRequest := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	tamperedRequest.Header.Set("Authorization", "Bearer "+tamperedToken)
	tamperedRecorder := httptest.NewRecorder()
	router.ServeHTTP(tamperedRecorder, tamperedRequest)
	if tamperedRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("tampered token status = %d, want %d; body: %s", tamperedRecorder.Code, http.StatusUnauthorized, tamperedRecorder.Body.String())
	}
}
