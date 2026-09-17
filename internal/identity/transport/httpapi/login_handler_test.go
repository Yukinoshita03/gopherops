package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Yukinoshita03/gopherops/internal/identity/application/usecase"
	"github.com/Yukinoshita03/gopherops/internal/identity/transport/httpapi"
)

func TestLoginHandlerSuccess(t *testing.T) {
	router := httpapi.NewRouter(
		registerUseCaseFunc(func(context.Context, usecase.RegisterInput) (usecase.RegisterOutput, error) {
			return usecase.RegisterOutput{}, nil
		}),
		loginUseCaseFunc(func(_ context.Context, input usecase.LoginInput) (usecase.LoginOutput, error) {
			if input.Username != "alice" || input.Password != "secret" {
				t.Fatalf("login input = %+v, want alice/secret", input)
			}
			return usecase.LoginOutput{UserID: 42}, nil
		}),
	)
	request := httptest.NewRequest(
		http.MethodPost,
		"/v1/auth/login",
		bytes.NewBufferString(`{"username":"alice","password":"secret"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	var body struct {
		UserID int64 `json:"user_id"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if body.UserID != 42 {
		t.Fatalf("user_id = %d, want 42", body.UserID)
	}
}

func TestLoginHandlerReturnsGenericUnauthorizedForInvalidCredentials(t *testing.T) {
	for _, failure := range []string{"unknown user", "wrong password"} {
		t.Run(failure, func(t *testing.T) {
			router := loginTestRouter(func(context.Context, usecase.LoginInput) (usecase.LoginOutput, error) {
				return usecase.LoginOutput{}, usecase.ErrInvalidCredentials
			})
			request := httptest.NewRequest(
				http.MethodPost,
				"/v1/auth/login",
				bytes.NewBufferString(`{"username":"alice","password":"secret"}`),
			)
			request.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, request)

			if recorder.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
			}
			if got, want := recorder.Body.String(), `{"code":"invalid_credentials","message":"username or password is incorrect"}`; got != want {
				t.Fatalf("response = %q, want %q", got, want)
			}
		})
	}
}

func TestLoginHandlerRejectsMalformedJSON(t *testing.T) {
	router := loginTestRouter(func(context.Context, usecase.LoginInput) (usecase.LoginOutput, error) {
		t.Fatal("login use case called for malformed JSON")
		return usecase.LoginOutput{}, nil
	})
	request := httptest.NewRequest(
		http.MethodPost,
		"/v1/auth/login",
		bytes.NewBufferString(`{"username":`),
	)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestLoginHandlerRejectsMissingCredentials(t *testing.T) {
	router := loginTestRouter(func(context.Context, usecase.LoginInput) (usecase.LoginOutput, error) {
		return usecase.LoginOutput{}, usecase.ErrInvalidLoginInput
	})
	request := httptest.NewRequest(
		http.MethodPost,
		"/v1/auth/login",
		bytes.NewBufferString(`{"username":"alice"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	var body struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if body.Code != "invalid_login_input" {
		t.Fatalf("error code = %q, want %q", body.Code, "invalid_login_input")
	}
}

func TestLoginHandlerDoesNotExposeUnexpectedErrors(t *testing.T) {
	router := loginTestRouter(func(context.Context, usecase.LoginInput) (usecase.LoginOutput, error) {
		return usecase.LoginOutput{}, errors.New("database password leaked")
	})
	request := httptest.NewRequest(
		http.MethodPost,
		"/v1/auth/login",
		bytes.NewBufferString(`{"username":"alice","password":"secret"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
	if bytes.Contains(recorder.Body.Bytes(), []byte("database password leaked")) {
		t.Fatalf("response exposed internal error: %s", recorder.Body.String())
	}
}

func loginTestRouter(login func(context.Context, usecase.LoginInput) (usecase.LoginOutput, error)) http.Handler {
	return httpapi.NewRouter(
		registerUseCaseFunc(func(context.Context, usecase.RegisterInput) (usecase.RegisterOutput, error) {
			return usecase.RegisterOutput{}, nil
		}),
		loginUseCaseFunc(login),
	)
}
