package main

import (
	"database/sql"
	"fmt"
	"time"
)

// SaveURL saves a URL to the database
func SaveURL(config *Config, urlData URLData) error {
	if db := config.GetDB(); db != nil {
		query := `
		INSERT INTO urls (alias, original_url, short_url, clicks, max_clicks, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
		`

		_, err := db.Exec(query,
			urlData.Alias,
			urlData.URL,
			urlData.ShortURL,
			urlData.Clicks,
			urlData.MaxClicks,
			urlData.CreatedAt,
		)

		if err != nil {
			return fmt.Errorf("failed to save URL: %v", err)
		}

		return nil
	}
	return fmt.Errorf("database connection not available")
}

// GetURLByAlias retrieves a URL by its alias
func GetURLByAlias(config *Config, alias string) (*URLData, error) {
	if db := config.GetDB(); db != nil {
		query := `
		SELECT alias, original_url, short_url, clicks, max_clicks, created_at
		FROM urls
		WHERE alias = ?
		`

		var urlData URLData
		var createdAt string

		err := db.QueryRow(query, alias).Scan(
			&urlData.Alias,
			&urlData.URL,
			&urlData.ShortURL,
			&urlData.Clicks,
			&urlData.MaxClicks,
			&createdAt,
		)

		if err != nil {
			if err == sql.ErrNoRows {
				return nil, nil // URL not found
			}
			return nil, fmt.Errorf("failed to get URL: %v", err)
		}

		// Parse created_at timestamp
		if parsedTime, err := time.Parse("2006-01-02 15:04:05", createdAt); err == nil {
			urlData.CreatedAt = parsedTime
		}

		// Set OriginalURL for compatibility
		urlData.OriginalURL = urlData.URL

		return &urlData, nil
	}
	return nil, fmt.Errorf("database connection not available")
}

// IncrementURLClicks increments the click count for a URL and returns the new count
func IncrementURLClicks(config *Config, alias string) (int, error) {
	if db := config.GetDB(); db == nil {
		return 0, fmt.Errorf("database connection not available")
	}

	db := config.GetDB()

	// Use a transaction to ensure atomicity
	tx, err := db.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// First, get current click count and max clicks
	var currentClicks, maxClicks int
	selectQuery := `SELECT clicks, max_clicks FROM urls WHERE alias = ?`
	err = tx.QueryRow(selectQuery, alias).Scan(&currentClicks, &maxClicks)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("URL with alias %s not found", alias)
		}
		return 0, fmt.Errorf("failed to get current clicks: %v", err)
	}

	// Check if already at max clicks
	if currentClicks >= maxClicks {
		return currentClicks, fmt.Errorf("URL has already reached maximum clicks (%d/%d)", currentClicks, maxClicks)
	}

	// Atomically increment click count only if under the limit
	updateQuery := `UPDATE urls SET clicks = clicks + 1 WHERE alias = ? AND clicks < max_clicks`
	result, err := tx.Exec(updateQuery, alias)
	if err != nil {
		return 0, fmt.Errorf("failed to update clicks: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get affected rows: %v", err)
	}

	if rowsAffected == 0 {
		// Re-check current state
		var recheckClicks, recheckMaxClicks int
		recheckQuery := `SELECT clicks, max_clicks FROM urls WHERE alias = ?`
		err = tx.QueryRow(recheckQuery, alias).Scan(&recheckClicks, &recheckMaxClicks)
		if err != nil {
			if err == sql.ErrNoRows {
				return 0, fmt.Errorf("URL with alias %s not found", alias)
			}
			return 0, fmt.Errorf("failed to recheck clicks: %v", err)
		}

		if recheckClicks >= recheckMaxClicks {
			return recheckClicks, fmt.Errorf("URL has already reached maximum clicks (%d/%d)", recheckClicks, recheckMaxClicks)
		}

		return 0, fmt.Errorf("failed to increment clicks - no rows affected")
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %v", err)
	}

	// Return new click count
	newClickCount := currentClicks + 1
	return newClickCount, nil
}

// GetAllURLs returns all stored URLs
func GetAllURLs(config *Config) ([]URLData, error) {
	if db := config.GetDB(); db != nil {
		query := `
		SELECT alias, original_url, short_url, clicks, max_clicks, created_at
		FROM urls
		ORDER BY created_at DESC
		`

		rows, err := db.Query(query)
		if err != nil {
			return nil, fmt.Errorf("failed to get URLs: %v", err)
		}
		defer rows.Close()

		var urls []URLData
		for rows.Next() {
			var urlData URLData
			var createdAt string

			err := rows.Scan(
				&urlData.Alias,
				&urlData.URL,
				&urlData.ShortURL,
				&urlData.Clicks,
				&urlData.MaxClicks,
				&createdAt,
			)

			if err != nil {
				return nil, fmt.Errorf("failed to scan URL: %v", err)
			}

			// Parse created_at timestamp
			if parsedTime, err := time.Parse("2006-01-02 15:04:05", createdAt); err == nil {
				urlData.CreatedAt = parsedTime
			}

			// Set OriginalURL for compatibility
			urlData.OriginalURL = urlData.URL

			urls = append(urls, urlData)
		}

		return urls, nil
	}
	return nil, fmt.Errorf("database connection not available")
}

// CleanupExpiredURLs removes URLs that have exceeded their click limit
func CleanupExpiredURLs(config *Config) (int, error) {
	if db := config.GetDB(); db != nil {
		query := `
		DELETE FROM urls
		WHERE clicks >= max_clicks
		`

		result, err := db.Exec(query)
		if err != nil {
			return 0, fmt.Errorf("failed to cleanup expired URLs: %v", err)
		}

		affected, err := result.RowsAffected()
		if err != nil {
			return 0, fmt.Errorf("failed to get affected rows: %v", err)
		}

		return int(affected), nil
	}
	return 0, fmt.Errorf("database connection not available")
}

// GetStats retrieves statistics about the URL shortener
func GetStats(config *Config) (*StatsResponse, error) {
	if db := config.GetDB(); db == nil {
		return nil, fmt.Errorf("database connection not available")
	}

	db := config.GetDB()
	stats := &StatsResponse{}

	// Get total number of URLs
	err := db.QueryRow("SELECT COUNT(*) FROM urls").Scan(&stats.TotalURLs)
	if err != nil {
		return nil, fmt.Errorf("failed to get total URLs count: %v", err)
	}

	// Get total clicks across all URLs
	err = db.QueryRow("SELECT COALESCE(SUM(clicks), 0) FROM urls").Scan(&stats.TotalClicks)
	if err != nil {
		return nil, fmt.Errorf("failed to get total clicks: %v", err)
	}

	// Get active URLs (clicks < max_clicks)
	err = db.QueryRow("SELECT COUNT(*) FROM urls WHERE clicks < max_clicks").Scan(&stats.ActiveURLs)
	if err != nil {
		return nil, fmt.Errorf("failed to get active URLs count: %v", err)
	}

	// Get expired URLs (clicks >= max_clicks)
	err = db.QueryRow("SELECT COUNT(*) FROM urls WHERE clicks >= max_clicks").Scan(&stats.ExpiredURLs)
	if err != nil {
		return nil, fmt.Errorf("failed to get expired URLs count: %v", err)
	}

	return stats, nil
}
