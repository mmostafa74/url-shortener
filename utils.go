package main

import (
	"crypto/rand"
	"fmt"
	"math/big"
	mathrand "math/rand"
	"net/url"
	"strings"
	"time"
)

// isValidURL checks if a string is a valid URL
func isValidURL(str string) bool {
	if str == "" {
		return false
	}

	// Add http:// if no scheme is present
	if !strings.HasPrefix(str, "http://") && !strings.HasPrefix(str, "https://") {
		str = "http://" + str
	}

	u, err := url.Parse(str)
	return err == nil && u.Scheme != "" && u.Host != ""
}

// generateRandomAlias generates a cryptographically secure random alias for URLs
func generateRandomAlias() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const length = 6

	b := make([]byte, length)
	charsetLen := big.NewInt(int64(len(charset)))

	for i := range b {
		// Generate cryptographically secure random number
		n, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			// Fallback to a simpler method if crypto/rand fails
			// This should rarely happen
			return generateFallbackAlias()
		}
		b[i] = charset[n.Int64()]
	}

	return string(b)
}

// generateFallbackAlias generates a fallback alias using time-based approach
// This is used only if crypto/rand fails (which should be very rare)
func generateFallbackAlias() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const length = 8 // Slightly longer for fallback

	// Create a new local random generator with time-based seed
	source := mathrand.NewSource(time.Now().UnixNano())
	rng := mathrand.New(source)

	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rng.Intn(len(charset))]
	}

	return string(b)
}

// generateCustomAlias validates and generates a custom alias
func generateCustomAlias(customAlias string) (string, error) {
	// Remove any whitespace
	customAlias = strings.TrimSpace(customAlias)

	// Check if empty
	if customAlias == "" {
		return "", fmt.Errorf("custom alias cannot be empty")
	}

	// Check length (between 3 and 50 characters)
	if len(customAlias) < 3 || len(customAlias) > 50 {
		return "", fmt.Errorf("custom alias must be between 3 and 50 characters")
	}

	// Check for valid characters (alphanumeric, hyphens, underscores)
	for _, char := range customAlias {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '-' || char == '_') {
			return "", fmt.Errorf("custom alias can only contain letters, numbers, hyphens, and underscores")
		}
	}

	// Check for reserved words
	reservedWords := []string{
		"api", "admin", "www", "mail", "ftp", "localhost", "health", "stats",
		"static", "assets", "public", "private", "secure", "login", "logout",
		"register", "signup", "signin", "dashboard", "profile", "settings",
	}

	lowerAlias := strings.ToLower(customAlias)
	for _, reserved := range reservedWords {
		if lowerAlias == reserved {
			return "", fmt.Errorf("'%s' is a reserved word and cannot be used as alias", customAlias)
		}
	}

	return customAlias, nil
}

// sanitizeURL cleans and validates a URL
func sanitizeURL(rawURL string) (string, error) {
	// Trim whitespace
	rawURL = strings.TrimSpace(rawURL)

	if rawURL == "" {
		return "", fmt.Errorf("URL cannot be empty")
	}

	// Add scheme if missing
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "http://" + rawURL
	}

	// Parse and validate URL
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL format: %v", err)
	}

	// Check if host is present
	if parsedURL.Host == "" {
		return "", fmt.Errorf("URL must have a valid host")
	}

	// Check for localhost in production (optional security measure)
	if strings.Contains(strings.ToLower(parsedURL.Host), "localhost") ||
		strings.Contains(parsedURL.Host, "127.0.0.1") ||
		strings.Contains(parsedURL.Host, "::1") {
		// You might want to allow this in development mode
		if appConfig != nil && appConfig.IsProduction() {
			return "", fmt.Errorf("localhost URLs are not allowed in production")
		}
	}

	return parsedURL.String(), nil
}
