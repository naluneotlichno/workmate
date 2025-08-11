package api

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

const (
	statusWarnThreshold  = 400
	statusErrorThreshold = 500
)

// ZerologLogger is a Gin middleware that logs requests using zerolog.
func ZerologLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery
		method := c.Request.Method
		clientIP := c.ClientIP()
		ua := c.Request.UserAgent()

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		size := c.Writer.Size()

		evt := log.Info()
		switch {
		case status >= statusErrorThreshold:
			evt = log.Error()
		case status >= statusWarnThreshold:
			evt = log.Warn()
		}

		if raw != "" {
			path = path + "?" + raw
		}

		evt.
			Int("status", status).
			Str("method", method).
			Str("path", path).
			Dur("latency", latency).
			Str("client_ip", clientIP).
			Int("bytes", size).
			Str("user_agent", ua).
			Msg("http request completed")
	}
}
