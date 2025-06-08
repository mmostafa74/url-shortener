package main

import "time"

// ShortenRequest represents the request payload for shortening URLs
type ShortenRequest struct {
	URL   string `json:"url" binding:"required"`
	Alias string `json:"alias,omitempty"`
}

// ShortenResponse represents the response for shortened URLs
type ShortenResponse struct {
	ShortURL    string    `json:"short_url"`
	OriginalURL string    `json:"original_url"`
	Alias       string    `json:"alias"`
	CreatedAt   time.Time `json:"created_at"`
	MaxClicks   int       `json:"max_clicks"`
	Clicks      int       `json:"clicks"`
}

// StatsResponse represents statistics about the URL shortener
type StatsResponse struct {
	TotalURLs   int `json:"total_urls"`
	TotalClicks int `json:"total_clicks"`
	ActiveURLs  int `json:"active_urls"`
	ExpiredURLs int `json:"expired_urls"`
}

// URLData represents the stored URL data
type URLData struct {
	Alias       string    `json:"alias"`
	URL         string    `json:"url"`
	OriginalURL string    `json:"original_url"` // Same as URL, for compatibility
	ShortURL    string    `json:"short_url"`
	Clicks      int       `json:"clicks"`
	MaxClicks   int       `json:"max_clicks"`
	CreatedAt   time.Time `json:"created_at"`
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string      `json:"status"`
	Timestamp time.Time   `json:"timestamp"`
	Version   string      `json:"version"`
	Mode      string      `json:"mode"`
	Config    interface{} `json:"config,omitempty"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error     string      `json:"error"`
	Message   string      `json:"message,omitempty"`
	Code      string      `json:"code,omitempty"`
	Details   interface{} `json:"details,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// ListURLsResponse represents the response for listing URLs with pagination
type ListURLsResponse struct {
	URLs       []URLData `json:"urls"`
	Count      int       `json:"count"`
	Page       int       `json:"page"`
	Limit      int       `json:"limit"`
	TotalPages int       `json:"total_pages"`
	HasNext    bool      `json:"has_next"`
	HasPrev    bool      `json:"has_prev"`
}
