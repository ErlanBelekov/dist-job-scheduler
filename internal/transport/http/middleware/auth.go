package middleware

import (
	"net/http"
	"strings"

	"github.com/ErlanBelekov/dist-job-scheduler/internal/domain"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Auth validates a Bearer JWT and sets "userID" in the gin context.
func Auth(jwtKey []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": domain.ErrUnauthorized.Error()})
			return
		}

		rawToken := strings.TrimPrefix(header, "Bearer ")

		token, err := jwt.Parse(rawToken, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, domain.ErrUnauthorized
			}
			return jwtKey, nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": domain.ErrUnauthorized.Error()})
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": domain.ErrUnauthorized.Error()})
			return
		}

		userID, ok := claims["sub"].(string)
		if !ok || userID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": domain.ErrUnauthorized.Error()})
			return
		}

		c.Set("userID", userID)
		c.Next()
	}
}
