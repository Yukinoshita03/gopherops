package httpapi

import (
	"errors"
	"net/http"

	"github.com/Yukinoshita03/gopherops/internal/identity/application/usecase"
	"github.com/gin-gonic/gin"
)

type LoginHandler struct {
	login usecase.LoginUseCase
}

func NewLoginHandler(login usecase.LoginUseCase) *LoginHandler {
	return &LoginHandler{login: login}
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	UserID int64 `json:"user_id"`
}

func (h *LoginHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{
			Code:    "invalid_json",
			Message: "request body must be valid JSON",
		})
		return
	}

	output, err := h.login.Execute(c.Request.Context(), usecase.LoginInput{
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrInvalidLoginInput):
			c.JSON(http.StatusBadRequest, errorResponse{
				Code:    "invalid_login_input",
				Message: "username and password are required",
			})
		case errors.Is(err, usecase.ErrInvalidCredentials):
			c.JSON(http.StatusUnauthorized, errorResponse{
				Code:    "invalid_credentials",
				Message: "username or password is incorrect",
			})
		default:
			c.JSON(http.StatusInternalServerError, errorResponse{
				Code:    "internal_error",
				Message: "login failed",
			})
		}
		return
	}

	c.JSON(http.StatusOK, loginResponse{UserID: output.UserID})
}
