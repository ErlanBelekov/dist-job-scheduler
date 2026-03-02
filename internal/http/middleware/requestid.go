package middleware

import (
	"github.com/ErlanBelekov/dist-job-scheduler/internal/requestid"
	"github.com/gin-gonic/gin"
)

// RequestID injects a request ID into the context and response header.
// If the incoming request already carries X-Request-ID, it is preserved;
// otherwise a new UUID v4 is generated.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if id == "" {
			id = requestid.New()
		}

		ctx := requestid.WithRequestID(c.Request.Context(), id)
		c.Request = c.Request.WithContext(ctx)
		c.Header("X-Request-ID", id)
		c.Next()
	}
}
