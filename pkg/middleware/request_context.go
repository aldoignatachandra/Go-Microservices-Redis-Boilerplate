package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/ignata/go-microservices-boilerplate/pkg/utils"
)

const correlationIDHeader = "X-Correlation-ID"

// RequestContextMetadata enriches request context with client/request metadata
// that can be consumed by downstream use cases and event producers.
func RequestContextMetadata() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := GetRequestID(c)
		correlationID := c.GetHeader(correlationIDHeader)
		if correlationID == "" {
			correlationID = requestID
		}

		enrichedCtx := utils.WithRequestContextMetadata(
			c.Request.Context(),
			c.ClientIP(),
			c.Request.UserAgent(),
			requestID,
			correlationID,
		)

		c.Request = c.Request.WithContext(enrichedCtx)
		c.Next()
	}
}
