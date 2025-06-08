package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	config := LoadConfig()
	defer config.CloseDB()

	// Print configuration in development mode
	config.PrintConfig()

	// Set Gin mode
	gin.SetMode(config.GinMode)

	// Create Gin router
	router := gin.New()

	// Load HTML templates from static directory
	router.LoadHTMLGlob("static/*.html")

	// Add middleware
	router.Use(requestLoggingMiddleware())
	router.Use(gin.Recovery())

	if config.EnableCORS {
		router.Use(corsMiddleware())
	}

	// Serve static files
	router.Static("/static", "./static")

	// Home page
	router.GET("/", func(c *gin.Context) {
		c.File("./static/index.html")
	})

	// Main shorten endpoint (this is what your frontend calls)
	router.POST("/shorten", shortenHandler(config))

	// Health check
	router.GET("/health", healthHandler(config))

	// API routes group (optional, for additional API endpoints)
	api := router.Group("/api")
	{
		api.POST("/shorten", shortenHandler(config))
		api.GET("/stats", statsHandler(config))
		api.GET("/urls", listURLsHandler(config))
		api.POST("/cleanup", cleanupHandler(config))
		api.GET("/info/:alias", urlInfoHandler(config))
	}

	// Redirect handler (must be last to catch all remaining routes)
	router.GET("/:alias", redirectHandler(config))

	// 404 handler
	router.NoRoute(notFoundHandler())

	// Start server
	log.Printf("🚀 Server starting on port %s", config.Port)
	log.Printf("🌐 Access the application at: %s", config.BaseURL)

	if err := http.ListenAndServe(":"+config.Port, router); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
