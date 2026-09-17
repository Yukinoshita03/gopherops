package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type meResponse struct {
	UserID int64 `json:"user_id"`
}

// Me 返回中间件从已验证令牌中提取的用户 ID。
func Me(c *gin.Context) {
	userID, ok := AuthenticatedUserID(c)
	if !ok {
		abortUnauthorized(c)
		return
	}

	c.JSON(http.StatusOK, meResponse{UserID: userID})
}
