package httpapi

import (
	"errors"
	"net/http"

	"github.com/Yukinoshita03/gopherops/internal/identity/application/usecase"
	"github.com/Yukinoshita03/gopherops/internal/identity/domain"
	"github.com/gin-gonic/gin"
)

type RegisterHandler struct {
	register usecase.RegisterUseCase
}

func NewRegisterHandler(register usecase.RegisterUseCase) *RegisterHandler {
	return &RegisterHandler{register: register}
}

type registerRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
type registerResponse struct {
	UserID int64 `json:"user_id"`
}
type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (h *RegisterHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{
			Code:    "invalid_json",
			Message: "request body must be valid JSON",
		})
		return
	}

	output, err := h.register.Execute(
		c.Request.Context(),
		usecase.RegisterInput{
			Username: req.Username,
			Password: req.Password,
		},
	)
	if err != nil {
		if errors.Is(err, usecase.ErrInvalidRegisterInput) {
			c.JSON(http.StatusBadRequest, errorResponse{
				Code:    "invalid_register_input",
				Message: "username and password are required",
			})
			return
		}
		if errors.Is(err, domain.ErrUserAlreadyExists) {
			c.JSON(http.StatusConflict, errorResponse{
				Code:    "username_already_exists",
				Message: "username is already registered",
			})
			return
		}

		// 不把数据库或其他内部错误原样返回给客户端。
		c.JSON(http.StatusInternalServerError, errorResponse{
			Code:    "internal_error",
			Message: "registration failed",
		})
		return
	}

	c.JSON(http.StatusCreated, registerResponse{UserID: output.UserID})
}
