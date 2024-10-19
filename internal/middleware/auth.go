package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

func AuthRequired(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		splittedHeader := strings.Split(authHeader, " ")

		if len(splittedHeader) != 2 || splittedHeader[0] != "Bearer" {
			c.JSON(401, gin.H{
				"error": "Unable to parse 'Authorization' header",
			})
			c.Abort()
			return
		}

		token := splittedHeader[1]
		if token == "" {
			c.JSON(401, gin.H{"error": "Authorization token required"})
			c.Abort()
			return
		} else if token != secret {
			c.JSON(403, gin.H{"error": "Invalid token provided"})
			c.Abort()
			return
		}

		c.Next()
	}
}
