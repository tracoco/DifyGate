package gateapi

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// AuthMiddleware creates a middleware that checks for a valid API key in the Authorization header
func AuthMiddleware(log *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := os.Getenv("DIFYGATE_API_KEY")
		if apiKey == "" {
			log.Error("API key not configured in environment variables")
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "API authentication not properly configured"})
			return
		}

		// Get the Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			log.Warn("Attempted access without Authorization header")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			return
		}

		// Check if the Authorization header has the correct format
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			log.Warn("Invalid Authorization header format")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization format, expected 'Bearer API_KEY'"})
			return
		}

		// Check if the API key is correct
		if parts[1] != apiKey {
			log.Warn("Invalid API key provided")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key"})
			return
		}

		// API key is valid, proceed
		c.Next()
	}
}
