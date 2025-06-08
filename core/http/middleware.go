// Package http provides HTTP middleware functionality.
package http

import (
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// LoggerMiddleware returns a gin middleware for logging HTTP requests.
func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()
		c.Next()
		endTime := time.Now()
		latency := endTime.Sub(startTime)
		reqMethod := c.Request.Method
		reqURI := c.Request.RequestURI
		status := c.Writer.Status()
		clientIP := c.ClientIP()
		log.WithFields(log.Fields{
			"status":     status,
			"latency":    latency,
			"client_ip":  clientIP,
			"req_method": reqMethod,
			"req_uri":    reqURI,
		}).Info()
	}
}
