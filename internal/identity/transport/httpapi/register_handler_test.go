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
	"github.com/Yukinoshita03/gopherops/internal/identity/domain"
	"github.com/Yukinoshita03/gopherops/internal/identity/transport/httpapi"
)

type registerUseCaseFunc func(context.Context, usecase.RegisterInput) (usecase.RegisterOutput, error)

func (f registerUseCaseFunc) Execute(
	ctx context.Context,
	input usecase.RegisterInput,
) (usecase.RegisterOutput, error) {
	return f(ctx, input)
}

type loginUseCaseFunc func(context.Context, usecase.LoginInput) (usecase.LoginOutput, error)

func (f loginUseCaseFunc) Execute(
	ctx context.Context,
	input usecase.LoginInput,
) (usecase.LoginOutput, error) {
	return f(ctx, input)
}

func TestRegisterHandlerReturnsConflictForExistingUsername(t *testing.T) {
	router := httpapi.NewRouter(registerUseCaseFunc(func(
		context.Context,
		usecase.RegisterInput,
	) (usecase.RegisterOutput, error) {
		return usecase.RegisterOutput{}, domain.ErrUserAlreadyExists
	}), loginUseCaseFunc(func(
		context.Context,
		usecase.LoginInput,
	) (usecase.LoginOutput, error) {
		return usecase.LoginOutput{}, nil
	}))
	request := httptest.NewRequest(
		http.MethodPost,
		"/v1/auth/register",
		bytes.NewBufferString(`{"username":"alice","password":"secret"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d; body: %s", recorder.Code, http.StatusConflict, recorder.Body.String())
	}
	var body struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if body.Code != "username_already_exists" {
		t.Fatalf("error code = %q, want %q", body.Code, "username_already_exists")
	}
}

func TestRegisterHandlerDoesNotExposeUnexpectedErrors(t *testing.T) {
	router := httpapi.NewRouter(registerUseCaseFunc(func(
		context.Context,
		usecase.RegisterInput,
	) (usecase.RegisterOutput, error) {
		return usecase.RegisterOutput{}, errors.New("database password leaked")
	}), loginUseCaseFunc(func(
		context.Context,
		usecase.LoginInput,
	) (usecase.LoginOutput, error) {
		return usecase.LoginOutput{}, nil
	}))
	request := httptest.NewRequest(
		http.MethodPost,
		"/v1/auth/register",
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
