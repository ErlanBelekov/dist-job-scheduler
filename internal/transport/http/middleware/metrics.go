package middleware

import (
	"strconv"
	"time"

	"github.com/ErlanBelekov/dist-job-scheduler/internal/metrics"
	"github.com/gin-gonic/gin"
)

func Metrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		status := strconv.Itoa(c.Writer.Status())
		path := c.FullPath()
		if path == "" {
			path = "unknown"
		}
		method := c.Request.Method
		duration := time.Since(start).Seconds()

		metrics.HTTPRequestDuration.WithLabelValues(method, path, status).Observe(duration)
		metrics.HTTPRequestsTotal.WithLabelValues(method, path, status).Inc()
	}
}
