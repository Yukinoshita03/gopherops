package httpapi

import (
	"net/http"
	"strconv"
	"strings"

	identitytoken "github.com/Yukinoshita03/gopherops/internal/identity/token"
	"github.com/gin-gonic/gin"
)

const authenticatedUserIDKey = "identity.authenticated_user_id"

// AuthMiddleware 只接受 Authorization 头中的 Bearer token。
// 验证通过后，将令牌主体中的正整数用户 ID 放入 Gin 请求上下文。
func AuthMiddleware(verifier identitytoken.Verifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		authorizationHeaders := c.Request.Header.Values("Authorization")
		if len(authorizationHeaders) != 1 || verifier == nil {
			abortUnauthorized(c)
			return
		}
		parts := strings.Fields(authorizationHeaders[0])
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			abortUnauthorized(c)
			return
		}

		claims, err := verifier.Verify(parts[1])
		if err != nil {
			abortUnauthorized(c)
			return
		}

		userID, err := strconv.ParseInt(claims.Subject, 10, 64)
		if err != nil || userID <= 0 {
			abortUnauthorized(c)
			return
		}

		c.Set(authenticatedUserIDKey, userID)
		c.Next()
	}
}

// AuthenticatedUserID 返回由认证中间件写入的用户 ID。
func AuthenticatedUserID(c *gin.Context) (int64, bool) {
	value, exists := c.Get(authenticatedUserIDKey)
	if !exists {
		return 0, false
	}
	userID, ok := value.(int64)
	return userID, ok
}

func abortUnauthorized(c *gin.Context) {
	c.Header("WWW-Authenticate", "Bearer")
	c.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse{
		Code:    "unauthorized",
		Message: "authentication required",
	})
}
