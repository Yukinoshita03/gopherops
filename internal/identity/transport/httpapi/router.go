package httpapi

import (
	"net/http"

	"github.com/Yukinoshita03/gopherops/internal/identity/application/usecase"
	"github.com/gin-gonic/gin"
)

func NewRouter(register usecase.RegisterUseCase, login usecase.LoginUseCase) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())

	registerHandler := NewRegisterHandler(register)
	loginHandler := NewLoginHandler(login)

	router.GET("/healthz", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	router.POST("/v1/auth/register", registerHandler.Register)
	router.POST("/v1/auth/login", loginHandler.Login)

	return router
}
