package handler

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/tracoco/DifyGate/config"
	"github.com/tracoco/DifyGate/gate"
	"github.com/tracoco/DifyGate/gateapi"
)

var (
	log         *logrus.Logger
	router      *gin.Engine
	mailService *gate.Service
)

func init() {
	// Initialize logger
	log = logrus.New()
	log.SetFormatter(&logrus.JSONFormatter{})

	// Check for API key
	apiKey := os.Getenv("DIFYGATE_API_KEY")
	if apiKey == "" {
		log.Warn("DIFYGATE_API_KEY environment variable not set - API endpoints will not be securely protected")
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.WithError(err).Fatal("Failed to load configuration")
	}

	// Initialize email service
	mailService = gate.NewService(cfg.DIFYGATE, log)

	// Initialize Gin router in release mode for production
	gin.SetMode(gin.ReleaseMode)
	router = gin.New()
	router.Use(gin.Recovery())

	// Register API routes
	gateapi.RegisterRoutes(router, mailService, log)
}

// Handler - Vercel serverless function entrypoint
func Handler(w http.ResponseWriter, r *http.Request) {
	router.ServeHTTP(w, r)
}
