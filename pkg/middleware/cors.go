// Package middleware provides common HTTP middleware for Go microservices.
package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// CORSConfig holds CORS middleware configuration.
type CORSConfig struct {
	// AllowedOrigins is a list of origins allowed to make requests.
	// Use "*" to allow any origin.
	AllowedOrigins []string

	// AllowedMethods is a list of HTTP methods allowed.
	// Default: ["GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"]
	AllowedMethods []string

	// AllowedHeaders is a list of headers allowed.
	// Default: ["Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID"]
	AllowedHeaders []string

	// ExposedHeaders is a list of headers exposed to the browser.
	ExposedHeaders []string

	// AllowCredentials indicates if credentials can be included in requests.
	AllowCredentials bool

	// MaxAge indicates how long the results of a preflight request can be cached.
	MaxAge int
}

// DefaultCORSConfig returns a default CORS configuration.
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID"},
		AllowCredentials: false,
		MaxAge:           86400, // 24 hours
	}
}

// CORS returns a CORS middleware with the given configuration.
func CORS(config CORSConfig) gin.HandlerFunc {
	// Set defaults
	if len(config.AllowedMethods) == 0 {
		config.AllowedMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	}
	if len(config.AllowedHeaders) == 0 {
		config.AllowedHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID"}
	}
	if config.MaxAge == 0 {
		config.MaxAge = 86400
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Check if origin is allowed
		allowedOrigin := ""
		for _, allowed := range config.AllowedOrigins {
			if allowed == "*" || allowed == origin {
				allowedOrigin = allowed
				break
			}
		}

		// Set CORS headers
		if allowedOrigin != "" {
			c.Header("Access-Control-Allow-Origin", allowedOrigin)
		}

		c.Header("Access-Control-Allow-Methods", strings.Join(config.AllowedMethods, ", "))
		c.Header("Access-Control-Allow-Headers", strings.Join(config.AllowedHeaders, ", "))

		if len(config.ExposedHeaders) > 0 {
			c.Header("Access-Control-Expose-Headers", strings.Join(config.ExposedHeaders, ", "))
		}

		if config.AllowCredentials {
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		c.Header("Access-Control-Max-Age", string(rune(config.MaxAge)))

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
