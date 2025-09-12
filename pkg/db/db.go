package db

import (
	"fmt"
	"log"
	"os"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB is the global database instance
var DB *gorm.DB

// Config holds database configuration
type Config struct {
	DatabasePath string
	LogLevel     logger.LogLevel
}

// DefaultConfig returns default database configuration
func DefaultConfig() *Config {
	return &Config{
		DatabasePath: "passengerprincess.db",
		LogLevel:     logger.Info,
	}
}

// Initialize sets up the database connection and runs migrations
func Initialize(config *Config) error {
	if config == nil {
		config = DefaultConfig()
	}

	var err error

	// Configure GORM logger
	gormConfig := &gorm.Config{
		Logger: logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			logger.Config{
				LogLevel: config.LogLevel,
			},
		),
	}

	// Open database connection
	DB, err = gorm.Open(sqlite.Open(config.DatabasePath), gormConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure SQLite settings
	if err := configureSQLite(config); err != nil {
		return fmt.Errorf("failed to configure SQLite: %w", err)
	}

	// Auto-migrate the schema
	if err := autoMigrate(); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	log.Println("Database initialized and migrated successfully")

	return nil
}

// configureSQLite applies SQLite-specific settings
func configureSQLite(config *Config) error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}

	pragmas := []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA journal_mode = WAL",
		"PRAGMA synchronous = NORMAL",
		"PRAGMA cache_size = 1000000",
		"PRAGMA temp_store = memory",
	}

	// Set connection pool settings for concurrent access
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)

	for _, pragma := range pragmas {
		if _, err := sqlDB.Exec(pragma); err != nil {
			return fmt.Errorf("failed to execute pragma %s: %w", pragma, err)
		}
	}

	return nil
}

// autoMigrate runs automatic migrations for all models
func autoMigrate() error {
	return DB.AutoMigrate(
		&Restaurant{},
		&Supercharger{},
		&RestaurantSuperchargerMapping{},
		&MapsCallLog{},
		&CacheHit{},
		&RouteCallLog{},
	)
}

// Close closes the database connection
func Close() error {
	if DB == nil {
		return nil
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}

	return sqlDB.Close()
}

// GetDB returns the global database instance
func GetDB() *gorm.DB {
	return DB
}

// Health checks database connectivity
func Health() error {
	if DB == nil {
		return fmt.Errorf("database not initialized")
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}

	return sqlDB.Ping()
}
