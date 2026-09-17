package httpapi

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Yukinoshita03/gopherops/internal/identity/domain"
	identitytoken "github.com/Yukinoshita03/gopherops/internal/identity/token"
	"github.com/gin-gonic/gin"
)

type verifierFunc func(string) (domain.Payload, error)

func (f verifierFunc) Verify(rawToken string) (domain.Payload, error) {
	return f(rawToken)
}

func TestAuthMiddlewareRejectsMissingOrMalformedAuthorization(t *testing.T) {
	tests := []struct {
		name           string
		authorizations []string
	}{
		{name: "missing"},
		{name: "non-bearer", authorizations: []string{"Basic token"}},
		{name: "missing token", authorizations: []string{"Bearer"}},
		{name: "extra fields", authorizations: []string{"Bearer token extra"}},
		{name: "duplicate headers", authorizations: []string{"Bearer first", "Bearer second"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			verifierCalls := 0
			nextCalled := false
			router := gin.New()
			router.GET("/protected", AuthMiddleware(verifierFunc(func(string) (domain.Payload, error) {
				verifierCalls++
				return domain.Payload{}, errors.New("unexpected verification call")
			})), func(c *gin.Context) {
				nextCalled = true
				c.Status(http.StatusNoContent)
			})

			request := httptest.NewRequest(http.MethodGet, "/protected", nil)
			for _, authorization := range test.authorizations {
				request.Header.Add("Authorization", authorization)
			}
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)

			assertUnauthorized(t, recorder)
			if verifierCalls != 0 {
				t.Fatalf("verifier calls = %d, want 0 for malformed authorization", verifierCalls)
			}
			if nextCalled {
				t.Fatal("protected handler was called for malformed authorization")
			}
		})
	}
}

func TestAuthMiddlewareRejectsInvalidTokenAndInvalidSubject(t *testing.T) {
	for _, test := range []struct {
		name    string
		payload domain.Payload
		err     error
	}{
		{name: "verification error", err: errors.New("signature mismatch")},
		{name: "non-numeric subject", payload: domain.Payload{Subject: "alice"}},
		{name: "zero subject", payload: domain.Payload{Subject: "0"}},
		{name: "negative subject", payload: domain.Payload{Subject: "-1"}},
		{name: "overflow subject", payload: domain.Payload{Subject: "999999999999999999999999"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			nextCalled := false
			router := gin.New()
			router.GET("/protected", AuthMiddleware(verifierFunc(func(string) (domain.Payload, error) {
				return test.payload, test.err
			})), func(c *gin.Context) {
				nextCalled = true
				c.Status(http.StatusNoContent)
			})

			request := httptest.NewRequest(http.MethodGet, "/protected", nil)
			request.Header.Set("Authorization", "Bearer signed-token")
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)

			assertUnauthorized(t, recorder)
			if nextCalled {
				t.Fatal("protected handler was called for an invalid token subject")
			}
		})
	}
}

func TestAuthMiddlewarePassesVerifiedUserIDToNextHandler(t *testing.T) {
	const rawToken = "signed.token.value"
	nextCalled := false
	router := gin.New()
	router.GET("/protected", AuthMiddleware(verifierFunc(func(got string) (domain.Payload, error) {
		if got != rawToken {
			t.Errorf("verifier token = %q, want %q", got, rawToken)
		}
		return domain.Payload{Subject: "42"}, nil
	})), func(c *gin.Context) {
		nextCalled = true
		userID, ok := AuthenticatedUserID(c)
		if !ok || userID != 42 {
			t.Errorf("AuthenticatedUserID() = %d, %t; want 42, true", userID, ok)
		}
		c.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("Authorization", "bEaReR "+rawToken)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d; body: %s", recorder.Code, http.StatusNoContent, recorder.Body.String())
	}
	if !nextCalled {
		t.Fatal("protected handler was not called for a valid token")
	}
}

func TestAuthMiddlewareRejectsMissingVerifier(t *testing.T) {
	router := gin.New()
	router.GET("/protected", AuthMiddleware(nil), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("Authorization", "Bearer signed-token")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	assertUnauthorized(t, recorder)
}

func assertUnauthorized(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body: %s", recorder.Code, http.StatusUnauthorized, recorder.Body.String())
	}
	if got, want := recorder.Header().Get("WWW-Authenticate"), "Bearer"; got != want {
		t.Fatalf("WWW-Authenticate = %q, want %q", got, want)
	}
	if got, want := recorder.Body.String(), `{"code":"unauthorized","message":"authentication required"}`; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

var _ identitytoken.Verifier = verifierFunc(nil)
