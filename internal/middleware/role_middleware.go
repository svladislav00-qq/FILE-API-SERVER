package middleware

import (
	"net/http"
	"slices"

	"github.com/gin-gonic/gin"
)

func RoleMiddleware(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role := c.GetString("role")

		if slices.Contains(roles, role) {
			c.Next()
			return
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
	}
}
