package main

import (
	"database/sql"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
)

// Config holds all configuration for the application
type Config struct {
	// Server Configuration
	Port    string
	GinMode string

	// Database Configuration
	DBPath           string
	DBBackupInterval time.Duration

	// Application Configuration
	MaxClicks int
	BaseURL   string

	// Development Settings
	LogLevel   string
	EnableCORS bool

	// Health Check Configuration
	HealthCheckInterval time.Duration
	CleanupInterval     time.Duration

	// Database connection (private)
	db *sql.DB
}

// LoadConfig loads configuration from environment variables and .env file
func LoadConfig() *Config {
	// Load .env file if it exists (ignore error in production)
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found or could not be loaded: %v", err)
	}

	config := &Config{
		// Server Configuration with defaults
		Port:    getEnv("PORT", "8080"),
		GinMode: getEnv("GIN_MODE", "debug"),

		// Database Configuration with defaults
		DBPath:           getEnv("DB_PATH", "./data/urls.db"),
		DBBackupInterval: getEnvAsDuration("DB_BACKUP_INTERVAL", 1*time.Hour),

		// Application Configuration with defaults
		MaxClicks: getEnvAsInt("MAX_CLICKS", 5),
		BaseURL:   getEnv("BASE_URL", "http://localhost:8080"),

		// Development Settings with defaults
		LogLevel:   getEnv("LOG_LEVEL", "info"),
		EnableCORS: getEnvAsBool("ENABLE_CORS", true),

		// Health Check Configuration with defaults
		HealthCheckInterval: getEnvAsDuration("HEALTH_CHECK_INTERVAL", 30*time.Second),
		CleanupInterval:     getEnvAsDuration("CLEANUP_INTERVAL", 5*time.Minute),
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		log.Fatalf("Configuration validation failed: %v", err)
	}

	// Initialize database
	if err := config.InitDB(); err != nil {
		log.Fatalf("Database initialization failed: %v", err)
	}

	return config
}

// InitDB initializes the database connection
func (c *Config) InitDB() error {
	// Create directory if it doesn't exist
	if err := os.MkdirAll("./data", 0755); err != nil {
		return err
	}

	// Open database connection
	db, err := sql.Open("sqlite3", c.DBPath+"?_foreign_keys=on&_journal_mode=WAL")
	if err != nil {
		return err
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return err
	}

	c.db = db

	// Create/migrate tables
	if err := c.createOrMigrateTables(); err != nil {
		return err
	}

	log.Printf("✅ Database initialized successfully at %s", c.DBPath)
	return nil
}

// GetDB returns the database connection
func (c *Config) GetDB() *sql.DB {
	return c.db
}

// CloseDB closes the database connection
func (c *Config) CloseDB() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

// createOrMigrateTables creates tables or migrates existing ones
func (c *Config) createOrMigrateTables() error {
	// First, check if the table exists and what columns it has
	tableInfo, err := c.getTableInfo("urls")
	if err != nil {
		return err
	}

	if len(tableInfo) == 0 {
		// Table doesn't exist, create it
		return c.createTables()
	}

	// Table exists, check if we need to migrate
	hasAlias := false
	for _, column := range tableInfo {
		if column == "alias" {
			hasAlias = true
			break
		}
	}

	if !hasAlias {
		// Need to migrate old table
		log.Println("🔄 Migrating database schema...")
		return c.migrateTables()
	}

	// Table is up to date
	return nil
}

// getTableInfo returns column names for a table
func (c *Config) getTableInfo(tableName string) ([]string, error) {
	query := "PRAGMA table_info(" + tableName + ")"
	rows, err := c.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, pk int
		var defaultValue sql.NullString

		err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk)
		if err != nil {
			return nil, err
		}
		columns = append(columns, name)
	}

	return columns, nil
}

// migrateTables migrates old table structure to new one
func (c *Config) migrateTables() error {
	// Start transaction
	tx, err := c.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Create new table with correct schema
	createNewTable := `
	CREATE TABLE urls_new (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		alias TEXT UNIQUE NOT NULL,
		original_url TEXT NOT NULL,
		short_url TEXT NOT NULL,
		clicks INTEGER DEFAULT 0,
		max_clicks INTEGER DEFAULT 5,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`

	if _, err := tx.Exec(createNewTable); err != nil {
		return err
	}

	// Copy data from old table (adjust based on your old schema)
	// This assumes the old table had: id, url, short_code, clicks, created_at
	copyData := `
	INSERT INTO urls_new (alias, original_url, short_url, clicks, max_clicks, created_at)
	SELECT
		COALESCE(short_code, SUBSTR(short_url, LENGTH(short_url) - 5)) as alias,
		COALESCE(url, original_url) as original_url,
		COALESCE(short_url, 'http://localhost:8080/' || short_code) as short_url,
		COALESCE(clicks, 0) as clicks,
		5 as max_clicks,
		COALESCE(created_at, CURRENT_TIMESTAMP) as created_at
	FROM urls;
	`

	if _, err := tx.Exec(copyData); err != nil {
		// If copy fails, try a simpler approach
		log.Println("⚠️  Complex migration failed, trying simple migration...")

		// Drop the new table and create a fresh one
		if _, err := tx.Exec("DROP TABLE urls_new"); err != nil {
			return err
		}

		// Drop old table and create new one
		if _, err := tx.Exec("DROP TABLE urls"); err != nil {
			return err
		}

		if _, err := tx.Exec(createNewTable); err != nil {
			return err
		}
	} else {
		// Drop old table and rename new one
		if _, err := tx.Exec("DROP TABLE urls"); err != nil {
			return err
		}

		if _, err := tx.Exec("ALTER TABLE urls_new RENAME TO urls"); err != nil {
			return err
		}
	}

	// Create indexes
	createIndexes := `
	CREATE INDEX IF NOT EXISTS idx_alias ON urls(alias);
	CREATE INDEX IF NOT EXISTS idx_created_at ON urls(created_at);
	`

	if _, err := tx.Exec(createIndexes); err != nil {
		return err
	}

	// Create trigger
	createTrigger := `
	CREATE TRIGGER IF NOT EXISTS update_urls_timestamp
	AFTER UPDATE ON urls
	FOR EACH ROW
	BEGIN
		UPDATE urls SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
	END;
	`

	if _, err := tx.Exec(createTrigger); err != nil {
		return err
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return err
	}

	log.Println("✅ Database migration completed successfully")
	return nil
}

// createTables creates the necessary database tables
func (c *Config) createTables() error {
	query := `
	CREATE TABLE IF NOT EXISTS urls (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		alias TEXT UNIQUE NOT NULL,
		original_url TEXT NOT NULL,
		short_url TEXT NOT NULL,
		clicks INTEGER DEFAULT 0,
		max_clicks INTEGER DEFAULT 5,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_alias ON urls(alias);
	CREATE INDEX IF NOT EXISTS idx_created_at ON urls(created_at);

	-- Trigger to update updated_at timestamp
	CREATE TRIGGER IF NOT EXISTS update_urls_timestamp
	AFTER UPDATE ON urls
	FOR EACH ROW
	BEGIN
		UPDATE urls SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
	END;
	`

	if _, err := c.db.Exec(query); err != nil {
		return err
	}

	log.Println("✅ Database tables created successfully")
	return nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Validate GIN_MODE
	if c.GinMode != "debug" && c.GinMode != "release" && c.GinMode != "test" {
		log.Printf("Warning: Invalid GIN_MODE '%s', using 'debug'. Valid values: debug, release, test", c.GinMode)
		c.GinMode = "debug"
	}

	// Validate MaxClicks
	if c.MaxClicks <= 0 {
		log.Printf("Warning: Invalid MAX_CLICKS '%d', using default value 5", c.MaxClicks)
		c.MaxClicks = 5
	}

	// Validate Port
	if c.Port == "" {
		log.Printf("Warning: Empty PORT, using default value 8080")
		c.Port = "8080"
	}

	return nil
}

// IsDevelopment returns true if running in development mode
func (c *Config) IsDevelopment() bool {
	return c.GinMode == "debug"
}

// IsProduction returns true if running in production mode
func (c *Config) IsProduction() bool {
	return c.GinMode == "release"
}

// IsTest returns true if running in test mode
func (c *Config) IsTest() bool {
	return c.GinMode == "test"
}

// Helper functions to read environment variables

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt gets an environment variable as integer or returns a default value
func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
		log.Printf("Warning: Invalid integer value for %s: %s, using default: %d", key, value, defaultValue)
	}
	return defaultValue
}

// getEnvAsBool gets an environment variable as boolean or returns a default value
func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
		log.Printf("Warning: Invalid boolean value for %s: %s, using default: %t", key, value, defaultValue)
	}
	return defaultValue
}

// getEnvAsDuration gets an environment variable as duration or returns a default value
func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
		log.Printf("Warning: Invalid duration value for %s: %s, using default: %v", key, value, defaultValue)
	}
	return defaultValue
}

// PrintConfig prints the current configuration (for debugging)
func (c *Config) PrintConfig() {
	if c.IsDevelopment() {
		log.Println("=== Application Configuration ===")
		log.Printf("Port: %s", c.Port)
		log.Printf("GIN Mode: %s", c.GinMode)
		log.Printf("Database Path: %s", c.DBPath)
		log.Printf("Max Clicks: %d", c.MaxClicks)
		log.Printf("Base URL: %s", c.BaseURL)
		log.Printf("Log Level: %s", c.LogLevel)
		log.Printf("Enable CORS: %t", c.EnableCORS)
		log.Printf("Health Check Interval: %v", c.HealthCheckInterval)
		log.Printf("Cleanup Interval: %v", c.CleanupInterval)
		log.Println("=================================")
	}
}
