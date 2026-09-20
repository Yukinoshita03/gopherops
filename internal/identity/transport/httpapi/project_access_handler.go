package httpapi

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/Yukinoshita03/gopherops/internal/identity/application/usecase"
	"github.com/Yukinoshita03/gopherops/internal/identity/domain"
	"github.com/gin-gonic/gin"
)

// ProjectAccessHandler checks whether the authenticated user can access a project.
type ProjectAccessHandler struct {
	authorizeProject usecase.AuthorizeProjectUseCase
}

func NewProjectAccessHandler(authorizeProject usecase.AuthorizeProjectUseCase) *ProjectAccessHandler {
	return &ProjectAccessHandler{authorizeProject: authorizeProject}
}

// Check returns 204 when the authenticated user belongs to the requested project.
func (h *ProjectAccessHandler) Check(c *gin.Context) {
	userID, ok := AuthenticatedUserID(c)
	if !ok {
		abortUnauthorized(c)
		return
	}

	projectID, err := strconv.ParseInt(c.Param("project_id"), 10, 64)
	if err != nil || projectID <= 0 {
		c.JSON(http.StatusBadRequest, errorResponse{
			Code:    "invalid_project_id",
			Message: "project ID must be a positive integer",
		})
		return
	}
	if h.authorizeProject == nil {
		c.JSON(http.StatusInternalServerError, errorResponse{
			Code:    "internal_error",
			Message: "project access check failed",
		})
		return
	}

	err = h.authorizeProject.Execute(c.Request.Context(), usecase.AuthorizeProjectInput{
		UserID:    userID,
		ProjectID: projectID,
	})
	switch {
	case err == nil:
		c.Status(http.StatusNoContent)
	case errors.Is(err, domain.ErrProjectAccessDenied):
		c.JSON(http.StatusForbidden, errorResponse{
			Code:    "project_access_denied",
			Message: "project access denied",
		})
	case errors.Is(err, usecase.ErrInvalidProjectAuthorizationInput):
		c.JSON(http.StatusBadRequest, errorResponse{
			Code:    "invalid_project_id",
			Message: "project ID must be a positive integer",
		})
	default:
		c.JSON(http.StatusInternalServerError, errorResponse{
			Code:    "internal_error",
			Message: "project access check failed",
		})
	}
}
