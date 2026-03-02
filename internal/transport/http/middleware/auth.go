package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

const errUnauthorized = "Unauthorized"

// Auth validates a Bearer JWT and sets "userID" in the gin context.
//
// When jwksURL is non-empty the token is verified against the JWKS endpoint
// (RS256 — Clerk). The key set is auto-cached and refreshed every 15 minutes.
//
// When jwksURL is empty, hmacKey is used for HS256 verification (legacy local dev).
func Auth(jwksURL string, hmacKey []byte) gin.HandlerFunc {
	var cache *jwk.Cache

	if jwksURL != "" {
		c := jwk.NewCache(context.Background())
		if err := c.Register(jwksURL, jwk.WithMinRefreshInterval(15*time.Minute)); err != nil {
			panic("jwk cache register: " + err.Error())
		}
		cache = c
	}

	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": errUnauthorized})
			return
		}

		rawToken := strings.TrimPrefix(header, "Bearer ")

		var (
			tok jwt.Token
			err error
		)

		if cache != nil {
			keySet, fetchErr := cache.Get(c.Request.Context(), jwksURL)
			if fetchErr != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": errUnauthorized})
				return
			}
			tok, err = jwt.Parse([]byte(rawToken), jwt.WithKeySet(keySet), jwt.WithValidate(true))
		} else {
			tok, err = jwt.Parse([]byte(rawToken), jwt.WithKey(jwa.HS256, hmacKey), jwt.WithValidate(true))
		}

		if err != nil || tok == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": errUnauthorized})
			return
		}

		userID := tok.Subject()
		if userID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": errUnauthorized})
			return
		}

		c.Set("userID", userID)
		c.Next()
	}
}
