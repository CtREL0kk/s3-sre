package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"myS3/internal/auth"
	"myS3/internal/dto"
)

const ContextUserID = "user_id"
const ContextUserAdmin = "user_admin"

func Auth(manager *auth.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractBearer(c)
		if token == "" {
			abortUnauthorized(c, "missing token")
			return
		}

		userID, isAdmin, err := manager.ParseAccessToken(token)
		if err != nil {
			abortUnauthorized(c, "invalid token")
			return
		}

		c.Set(ContextUserID, userID)
		c.Set(ContextUserAdmin, isAdmin)
		c.Next()
	}
}

func Admin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if admin, _ := c.Get(ContextUserAdmin); admin != true {
			c.AbortWithStatusJSON(http.StatusForbidden, dto.Error{
				Error: dto.ErrorDetail{Code: "forbidden", Message: "требуются права администратора"},
			})
			return
		}
		c.Next()
	}
}

func extractBearer(c *gin.Context) string {
	h := c.GetHeader("Authorization")
	const prefix = "Bearer "
	if len(h) > len(prefix) && strings.EqualFold(h[:len(prefix)], prefix) {
		return h[len(prefix):]
	}
	return ""
}

func abortUnauthorized(c *gin.Context, message string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, dto.Error{
		Error: dto.ErrorDetail{Code: "unauthorized", Message: message},
	})
}
