// handlers.go
package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// Global config variable (set in main)
var appConfig *Config

// shortenHandler handles URL shortening requests with enhanced validation and features
func shortenHandler(config *Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()

		var req struct {
			URL       string `json:"url"`
			Alias     string `json:"alias"`
			MaxClicks *int   `json:"max_clicks"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Printf("Invalid request format from %s: %v", c.ClientIP(), err)
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:     "Invalid request format",
				Message:   "Please provide a valid JSON request with 'url' field",
				Code:      "INVALID_REQUEST",
				Details:   map[string]interface{}{"validation_error": err.Error()},
				Timestamp: time.Now(),
			})
			return
		}

		// Enhanced logging
		log.Printf("Shorten request from %s: URL=%s, Alias=%s, UserAgent=%s",
			c.ClientIP(), req.URL, req.Alias, c.GetHeader("User-Agent"))

		// Sanitize and validate URL with enhanced validation
		sanitizedURL, err := sanitizeURL(req.URL)
		if err != nil {
			log.Printf("Invalid URL from %s: %s - %v", c.ClientIP(), req.URL, err)
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:     "Invalid URL",
				Message:   fmt.Sprintf("The provided URL is not valid: %v", err),
				Details:   map[string]interface{}{"original_url": req.URL},
				Timestamp: time.Now(),
			})
			return
		}

		// Check for URL length limits
		if len(sanitizedURL) > 2048 {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:     "URL too long",
				Message:   "URL must be less than 2048 characters",
				Timestamp: time.Now(),
			})
			return
		}

		// Check if URL already exists (optional deduplication)
		if existingURL, _ := GetURLByAlias(config, sanitizedURL); existingURL != nil {
			log.Printf("URL already exists: %s -> %s", sanitizedURL, existingURL.ShortURL)
			c.JSON(http.StatusOK, ShortenResponse{
				ShortURL:    existingURL.ShortURL,
				OriginalURL: existingURL.URL,
				Alias:       existingURL.Alias,
				CreatedAt:   existingURL.CreatedAt,
				MaxClicks:   existingURL.MaxClicks,
				Clicks:      existingURL.Clicks,
			})
			return
		}

		// Determine max clicks (use custom value if provided, otherwise use config default)
		maxClicks := config.MaxClicks
		if req.MaxClicks != nil && *req.MaxClicks > 0 && *req.MaxClicks <= 10000 {
			maxClicks = *req.MaxClicks
		}

		var alias string
		if req.Alias != "" {
			// Enhanced custom alias validation
			validatedAlias, err := generateCustomAlias(req.Alias)
			if err != nil {
				c.JSON(http.StatusBadRequest, ErrorResponse{
					Error:     "Invalid custom alias",
					Message:   err.Error(),
					Details:   map[string]interface{}{"alias": req.Alias},
					Timestamp: time.Now(),
				})
				return
			}

			// Check if alias already exists
			existingURL, err := GetURLByAlias(config, validatedAlias)
			if err != nil {
				log.Printf("Database error checking alias %s: %v", validatedAlias, err)
				c.JSON(http.StatusInternalServerError, ErrorResponse{
					Error:     "Database error",
					Message:   "Failed to check alias availability",
					Timestamp: time.Now(),
				})
				return
			}

			if existingURL != nil {
				c.JSON(http.StatusConflict, ErrorResponse{
					Error:     "Alias already exists",
					Message:   fmt.Sprintf("The alias '%s' is already taken. Please choose a different one.", validatedAlias),
					Details:   map[string]interface{}{"alias": validatedAlias},
					Timestamp: time.Now(),
				})
				return
			}

			alias = validatedAlias
		} else {
			// Enhanced random alias generation with retry handling
			for attempts := 0; attempts < 20; attempts++ {
				randomAlias := generateRandomAlias()
				existingURL, err := GetURLByAlias(config, randomAlias)
				if err != nil {
					log.Printf("Database error generating alias (attempt %d): %v", attempts+1, err)
					if attempts >= 19 {
						c.JSON(http.StatusInternalServerError, ErrorResponse{
							Error:     "Database error",
							Message:   "Failed to generate unique alias",
							Timestamp: time.Now(),
						})
						return
					}
					continue
				}

				if existingURL == nil {
					alias = randomAlias
					break
				}
			}

			if alias == "" {
				c.JSON(http.StatusInternalServerError, ErrorResponse{
					Error:     "Failed to generate unique alias",
					Message:   "Please try again or provide a custom alias",
					Timestamp: time.Now(),
				})
				return
			}
		}

		// Create URL data with enhanced fields
		shortURL := fmt.Sprintf("%s/%s", config.BaseURL, alias)
		urlData := URLData{
			Alias:       alias,
			URL:         sanitizedURL,
			OriginalURL: sanitizedURL,
			ShortURL:    shortURL,
			Clicks:      0,
			MaxClicks:   maxClicks,
			CreatedAt:   time.Now(),
		}

		// Save to database with error handling
		if err := SaveURL(config, urlData); err != nil {
			log.Printf("Error saving URL %s: %v", alias, err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:     "Failed to save URL",
				Message:   "Please try again",
				Details:   map[string]interface{}{"alias": alias},
				Timestamp: time.Now(),
			})
			return
		}

		// Enhanced success logging
		duration := time.Since(startTime)
		log.Printf("Successfully created short URL: %s -> %s (took %v)", shortURL, sanitizedURL, duration)

		// Return enhanced success response
		response := ShortenResponse{
			ShortURL:    shortURL,
			OriginalURL: sanitizedURL,
			Alias:       alias,
			CreatedAt:   urlData.CreatedAt,
			MaxClicks:   urlData.MaxClicks,
			Clicks:      urlData.Clicks,
		}

		c.JSON(http.StatusCreated, response)

	}
}

// redirectHandler handles URL redirection with enhanced tracking and error handling
func redirectHandler(config *Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		alias := c.Param("alias")
		if alias == "" {
			log.Printf("Missing alias in redirect request from %s", c.ClientIP())
			c.HTML(http.StatusNotFound, "404.html", gin.H{
				"error":   "Missing URL Alias",
				"message": "No alias provided in the URL",
			})
			return
		}

		// Enhanced logging with user agent and referrer
		userAgent := c.GetHeader("User-Agent")
		referrer := c.GetHeader("Referer")
		log.Printf("Redirect request: alias=%s, ip=%s, user_agent=%s, referrer=%s",
			alias, c.ClientIP(), userAgent, referrer)

		// Try to increment click count first - this will handle all validation
		newClickCount, err := IncrementURLClicks(config, alias)
		if err != nil {
			log.Printf("Failed to increment clicks for %s: %v", alias, err)

			// Get URL data to provide better error messages
			urlData, dbErr := GetURLByAlias(config, alias)
			if dbErr != nil || urlData == nil {
				log.Printf("URL not found for alias: %s", alias)
				c.HTML(http.StatusNotFound, "404.html", gin.H{
					"error":   "URL Not Found",
					"message": fmt.Sprintf("No URL found for alias: %s", alias),
					"alias":   alias,
				})
				return
			}

			// Check if it's an expiration error
			if urlData.Clicks >= urlData.MaxClicks {
				log.Printf("URL expired: %s (%d/%d clicks)", alias, urlData.Clicks, urlData.MaxClicks)
				c.HTML(http.StatusGone, "404.html", gin.H{
					"error":      "URL Expired",
					"message":    fmt.Sprintf("This URL has reached its maximum click limit of %d", urlData.MaxClicks),
					"alias":      alias,
					"clicks":     urlData.Clicks,
					"max_clicks": urlData.MaxClicks,
					"is_expired": true,
				})
				return
			}

			// Other database errors
			c.HTML(http.StatusInternalServerError, "404.html", gin.H{
				"error":   "Database Error",
				"message": "Failed to process request",
			})
			return
		}

		// If we get here, the click was successfully incremented
		// Get the URL data for redirect
		urlData, err := GetURLByAlias(config, alias)
		if err != nil || urlData == nil {
			log.Printf("Database error after successful increment for alias %s: %v", alias, err)
			c.HTML(http.StatusInternalServerError, "404.html", gin.H{
				"error":   "Database Error",
				"message": "Failed to retrieve URL data",
			})
			return
		}

		// Log successful click tracking
		log.Printf("Click tracked: %s (%d/%d clicks)", alias, newClickCount, urlData.MaxClicks)

		// Check if this was the last allowed click
		if newClickCount >= urlData.MaxClicks {
			log.Printf("Info: URL %s has reached its maximum click limit (%d/%d)", alias, newClickCount, urlData.MaxClicks)
		}

		// Add cache-control headers to prevent browser caching
		c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", "0")

		// Enhanced redirect with proper status code
		log.Printf("Redirecting %s to %s", alias, urlData.URL)
		c.Redirect(http.StatusFound, urlData.URL) // Use 302 instead of 301 to prevent caching
	}
}

// statsHandler provides enhanced statistics
func statsHandler(config *Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		stats, err := GetStats(config)
		if err != nil {
			log.Printf("Error retrieving stats: %v", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:     "Failed to retrieve statistics",
				Message:   err.Error(),
				Code:      "STATS_ERROR",
				Timestamp: time.Now(),
			})
			return
		}

		c.JSON(http.StatusOK, stats)
	}
}

// healthHandler provides comprehensive health check
func healthHandler(config *Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		health := gin.H{
			"status":    "healthy",
			"timestamp": time.Now().UTC(),
			"version":   "1.0.0",
		}

		// Check database connection
		if db := config.GetDB(); db != nil {
			if err := db.Ping(); err != nil {
				health["status"] = "unhealthy"
				health["database"] = "disconnected"
				health["error"] = err.Error()
				c.JSON(http.StatusServiceUnavailable, health)
				return
			}
			health["database"] = "connected"

			// Add database stats
			if stats, err := GetStats(config); err == nil {
				health["stats"] = gin.H{
					"total_urls":   stats.TotalURLs,
					"total_clicks": stats.TotalClicks,
					"active_urls":  stats.ActiveURLs,
					"expired_urls": stats.ExpiredURLs,
				}
			}
		} else {
			health["database"] = "not_configured"
		}

		c.JSON(http.StatusOK, health)
	}
}

// requestLoggingMiddleware logs all incoming requests
func requestLoggingMiddleware() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("%s - [%s] \"%s %s %s %d %s \"%s\" %s\"\n",
			param.ClientIP,
			param.TimeStamp.Format(time.RFC1123),
			param.Method,
			param.Path,
			param.Request.Proto,
			param.StatusCode,
			param.Latency,
			param.Request.UserAgent(),
			param.ErrorMessage,
		)
	})
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		// Set CORS headers
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-Requested-With")
		c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400")

		// Handle preflight OPTIONS request
		if c.Request.Method == "OPTIONS" {
			log.Printf("CORS preflight request from origin: %s", origin)
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func urlInfoHandler(config *Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		alias := c.Param("alias")
		if alias == "" {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:     "Missing alias parameter",
				Code:      "MISSING_ALIAS",
				Timestamp: time.Now(),
			})
			return
		}

		// Get URL from database
		urlData, err := GetURLByAlias(config, alias)
		if err != nil {
			log.Printf("Database error retrieving URL info for %s: %v", alias, err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:     "Database error",
				Message:   "Failed to retrieve URL",
				Code:      "DATABASE_ERROR",
				Timestamp: time.Now(),
			})
			return
		}

		if urlData == nil {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:     "URL not found",
				Message:   fmt.Sprintf("No URL found for alias: %s", alias),
				Code:      "URL_NOT_FOUND",
				Timestamp: time.Now(),
			})
			return
		}

		// Calculate remaining clicks
		remainingClicks := urlData.MaxClicks - urlData.Clicks
		if remainingClicks < 0 {
			remainingClicks = 0
		}

		// Return comprehensive URL information
		c.JSON(http.StatusOK, gin.H{
			"alias":            urlData.Alias,
			"original_url":     urlData.URL,
			"short_url":        urlData.ShortURL,
			"clicks":           urlData.Clicks,
			"max_clicks":       urlData.MaxClicks,
			"remaining_clicks": remainingClicks,
			"created_at":       urlData.CreatedAt,
			"is_expired":       urlData.Clicks >= urlData.MaxClicks,
			"usage_percentage": float64(urlData.Clicks) / float64(urlData.MaxClicks) * 100,
		})
	}
}

func cleanupHandler(config *Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Printf("Cleanup request from %s", c.ClientIP())

		deletedCount, err := CleanupExpiredURLs(config)
		if err != nil {
			log.Printf("Cleanup failed: %v", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:     "Failed to cleanup expired URLs",
				Message:   err.Error(),
				Code:      "CLEANUP_ERROR",
				Timestamp: time.Now(),
			})
			return
		}

		log.Printf("Cleanup completed: %d URLs deleted", deletedCount)
		c.JSON(http.StatusOK, gin.H{
			"message":       "Cleanup completed successfully",
			"deleted_count": deletedCount,
			"timestamp":     time.Now(),
		})
	}
}

func listURLsHandler(config *Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Parse pagination parameters
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
		if page < 1 {
			page = 1
		}
		if limit < 1 || limit > 100 {
			limit = 50
		}

		// Parse filter parameters
		status := c.Query("status") // "active", "expired", or "all"

		urls, err := GetAllURLs(config)
		if err != nil {
			log.Printf("Error retrieving URLs: %v", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:     "Failed to retrieve URLs",
				Message:   err.Error(),
				Code:      "RETRIEVAL_ERROR",
				Timestamp: time.Now(),
			})
			return
		}

		// Filter URLs based on status
		var filteredURLs []URLData
		for _, url := range urls {
			switch status {
			case "active":
				if url.Clicks < url.MaxClicks {
					filteredURLs = append(filteredURLs, url)
				}
			case "expired":
				if url.Clicks >= url.MaxClicks {
					filteredURLs = append(filteredURLs, url)
				}
			default:
				filteredURLs = append(filteredURLs, url)
			}
		}

		// Calculate pagination
		totalURLs := len(filteredURLs)
		totalPages := (totalURLs + limit - 1) / limit
		start := (page - 1) * limit
		end := start + limit
		if end > totalURLs {
			end = totalURLs
		}

		var paginatedURLs []URLData
		if start < totalURLs {
			paginatedURLs = filteredURLs[start:end]
		}

		response := ListURLsResponse{
			URLs:       paginatedURLs,
			Count:      len(paginatedURLs),
			Page:       page,
			Limit:      limit,
			TotalPages: totalPages,
			HasNext:    page < totalPages,
			HasPrev:    page > 1,
		}

		c.JSON(http.StatusOK, response)
	}
}

func notFoundHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Printf("404 - Endpoint not found: %s %s from %s",
			c.Request.Method, c.Request.URL.Path, c.ClientIP())

		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:     "Endpoint not found",
			Message:   fmt.Sprintf("The requested endpoint %s %s was not found", c.Request.Method, c.Request.URL.Path),
			Code:      "ENDPOINT_NOT_FOUND",
			Timestamp: time.Now(),
		})
	}
}
