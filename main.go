package main

import (
	"os"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/tracoco/DifyGate/config"
	"github.com/tracoco/DifyGate/gate"
	"github.com/tracoco/DifyGate/gateapi"
)

func main() {
	// Initialize logger
	log := logrus.New()
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

	// Initialize gate service
	gateService := gate.NewService(cfg.DIFYGATE, log)

	// Initialize Gin router
	router := gin.Default()

	// Register API routes
	gateapi.RegisterRoutes(router, gateService, log)

	// Start the server
	log.WithField("port", 6001).Info("Starting server")
	if err := router.Run(":6001"); err != nil {
		log.WithError(err).Fatal("Server failed to start")
	}
}
