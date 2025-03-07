package gateapi

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/tracoco/DifyGate/gate"
)

// RegisterRoutes sets up all API routes
func RegisterRoutes(r *gin.Engine, mailService *gate.Service, log *logrus.Logger) {
	// Add request logging middleware
	r.Use(LoggingMiddleware(log))

	// API versioning
	v1 := r.Group("/api/v1")

	handler := NewWhatsAppHandler(log)
	// WhatsApp webhook endpoints - NOT protected by auth (needed for Meta verification)
	whatsapp := v1.Group("/whatsapp")
	{
		// Handler for WhatsApp webhook verification (GET) and messages (POST)
		whatsapp.GET("/webhook", handler.HandleWhatsAppWebhookGet)
		whatsapp.POST("/webhook", handler.HandleWhatsAppWebhookPost)
	}

	// Protected routes - require API key
	protected := v1.Group("")
	protected.Use(AuthMiddleware(log))

	// Health check endpoint
	protected.GET("/health", HealthCheck)

	// Email endpoints
	emails := protected.Group("/emails")
	{
		handler := NewEmailHandler(mailService, log)
		emails.POST("/send", handler.SendEmail)
	}
}

// LoggingMiddleware adds request logging
func LoggingMiddleware(log *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		start := time.Now()

		// Process request
		c.Next()

		// Log request details
		latency := time.Since(start)
		log.WithFields(logrus.Fields{
			"status":     c.Writer.Status(),
			"method":     c.Request.Method,
			"path":       c.Request.URL.Path,
			"latency":    latency,
			"client_ip":  c.ClientIP(),
			"user_agent": c.Request.UserAgent(),
		}).Info("API request")
	}
}

// HealthCheck provides a simple health check endpoint
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"service":   "DifyGate",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}
