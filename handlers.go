// handlers.go
package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
)

const shortIDLength = 6

func generateShortID() (string, error) {
	b := make([]byte, shortIDLength)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	id := base64.URLEncoding.EncodeToString(b)
	return id[:shortIDLength], nil
}

func validateURL(input string) error {
	u, err := url.ParseRequestURI(input)
	if err != nil {
		return errors.New("invalid URL format")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return errors.New("URL must start with http or https")
	}
	return nil
}

func shortenURLHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		URL   string `json:"url"`
		Alias string `json:"alias"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	if err := validateURL(req.URL); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	shortID := strings.TrimSpace(req.Alias)
	var err error

	if shortID != "" {
		if len(shortID) > 30 {
			http.Error(w, "Alias too long", http.StatusBadRequest)
			return
		}
		var exists int
		err = db.QueryRow("SELECT COUNT(*) FROM urls WHERE id = ?", shortID).Scan(&exists)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		if exists > 0 {
			http.Error(w, "Alias already taken", http.StatusConflict)
			return
		}
	} else {
		for i := 0; i < 5; i++ {
			shortID, err = generateShortID()
			if err != nil {
				http.Error(w, "Failed to generate short ID", http.StatusInternalServerError)
				return
			}
			var exists int
			err = db.QueryRow("SELECT COUNT(*) FROM urls WHERE id = ?", shortID).Scan(&exists)
			if err != nil {
				http.Error(w, "Database error", http.StatusInternalServerError)
				return
			}
			if exists == 0 {
				break
			}
		}
	}

	_, err = db.Exec("INSERT INTO urls (id, long_url) VALUES (?, ?)", shortID, req.URL)
	if err != nil {
		http.Error(w, "Failed to store URL", http.StatusInternalServerError)
		return
	}

	response := map[string]string{
		"short_url": "http://localhost:8081/" + shortID,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func redirectHandler(w http.ResponseWriter, r *http.Request) {
	shortID := strings.TrimPrefix(r.URL.Path, "/")
	if shortID == "" {
		http.NotFound(w, r)
		return
	}

	var longURL string
	err := db.QueryRow("SELECT long_url FROM urls WHERE id = ?", shortID).Scan(&longURL)
	if err == sql.ErrNoRows {
		http.NotFound(w, r)
		return
	} else if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, longURL, http.StatusFound)
}
