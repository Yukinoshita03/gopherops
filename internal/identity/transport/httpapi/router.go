package httpapi

import (
	"net/http"

	"github.com/Yukinoshita03/gopherops/internal/identity/application/usecase"
	identitytoken "github.com/Yukinoshita03/gopherops/internal/identity/token"
	"github.com/gin-gonic/gin"
)

func NewRouter(
	register usecase.RegisterUseCase,
	login usecase.LoginUseCase,
	verifier identitytoken.Verifier,
	authorizeProject ...usecase.AuthorizeProjectUseCase,
) *gin.Engine {
	if len(authorizeProject) > 1 {
		panic("NewRouter accepts at most one project authorization use case")
	}
	var projectAccessHandler *ProjectAccessHandler
	if len(authorizeProject) == 1 {
		projectAccessHandler = NewProjectAccessHandler(authorizeProject[0])
	}

	router := gin.New()
	router.Use(gin.Recovery())

	registerHandler := NewRegisterHandler(register)
	loginHandler := NewLoginHandler(login)

	router.GET("/healthz", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	router.POST("/v1/auth/register", registerHandler.Register)
	router.POST("/v1/auth/login", loginHandler.Login)
	router.GET("/v1/me", AuthMiddleware(verifier), Me)
	if projectAccessHandler != nil {
		router.GET("/v1/projects/:project_id/access", AuthMiddleware(verifier), projectAccessHandler.Check)
	}

	return router
}
