package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/ignata/go-microservices-boilerplate/pkg/utils"
)

func TestRequestContextMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(RequestIDKey, "req-123")
		c.Next()
	})
	router.Use(RequestContextMetadata())
	router.GET("/test", func(c *gin.Context) {
		ctx := c.Request.Context()
		assert.Equal(t, "req-123", utils.GetRequestIDFromContext(ctx))
		assert.Equal(t, "corr-123", utils.GetCorrelationIDFromContext(ctx))
		assert.Equal(t, "198.51.100.24", utils.GetIPAddressFromContext(ctx))
		assert.Equal(t, "PostmanRuntime/7.43.0", utils.GetUserAgentFromContext(ctx))
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "198.51.100.24:54321"
	req.Header.Set("User-Agent", "PostmanRuntime/7.43.0")
	req.Header.Set("X-Correlation-ID", "corr-123")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestRequestContextMetadata_FallbackCorrelationID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(RequestIDKey, "req-456")
		c.Next()
	})
	router.Use(RequestContextMetadata())
	router.GET("/test", func(c *gin.Context) {
		ctx := c.Request.Context()
		assert.Equal(t, "req-456", utils.GetRequestIDFromContext(ctx))
		assert.Equal(t, "req-456", utils.GetCorrelationIDFromContext(ctx))
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "203.0.113.5:12345"
	req.Header.Set("User-Agent", "curl/8.0")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}
