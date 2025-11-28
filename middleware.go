package main

import (
	"time"

	"github.com/gin-gonic/gin"
)

func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		method := c.Request.Method
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		httpRequestTotal.WithLabelValues(method, path).Inc()

		duration := time.Since(start).Seconds()
		httpRequestDuration.WithLabelValues(method, path).Observe(duration)
	}
}
