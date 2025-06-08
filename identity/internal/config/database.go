package config

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	log "github.com/sirupsen/logrus"
)

const (
	driverPostgres      = "postgres"
	driverPostgresAlias = "postgresql"
	driverSqlite        = "sqlite"
)

// NewDatabase creates a new database connection based on the configuration.
func NewDatabase(config *DatabaseConfig) (*gorm.DB, error) {
	var dialector gorm.Dialector

	switch config.Driver {
	case driverSqlite:
		dialector = sqlite.Open(config.DSN)
	case driverPostgres, driverPostgresAlias:
		dsn := buildPostgresDSN(config)
		dialector = postgres.Open(dsn)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", config.Driver)
	}

	// Configure GORM logger
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	}

	// Enable SQL logging in development
	if log.GetLevel() == log.DebugLevel || log.GetLevel() == log.TraceLevel {
		gormConfig.Logger = logger.Default.LogMode(logger.Info)
	}

	db, err := gorm.Open(dialector, gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool for PostgreSQL
	if config.Driver == driverPostgres || config.Driver == driverPostgresAlias {
		sqlDB, err := db.DB()
		if err != nil {
			return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
		}

		// Set connection pool settings
		sqlDB.SetMaxIdleConns(10)
		sqlDB.SetMaxOpenConns(100)
	}

	log.WithFields(log.Fields{
		"driver": config.Driver,
		"dsn":    maskPassword(config),
	}).Info("database connection established")

	return db, nil
}

// buildPostgresDSN builds a PostgreSQL DSN from the configuration.
func buildPostgresDSN(config *DatabaseConfig) string {
	if config.DSN != "" {
		return config.DSN
	}

	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host,
		config.Port,
		config.Username,
		config.Password,
		config.Database,
		config.SSLMode,
	)
}

// maskPassword returns a masked version of the database config for logging.
func maskPassword(config *DatabaseConfig) string {
	switch config.Driver {
	case driverSqlite:
		return config.DSN
	case driverPostgres, driverPostgresAlias:
		if config.DSN != "" {
			// Simple masking for DSN - replace password with ***
			return "***masked***"
		}
		return fmt.Sprintf("host=%s port=%d user=%s password=*** dbname=%s sslmode=%s",
			config.Host,
			config.Port,
			config.Username,
			config.Database,
			config.SSLMode,
		)
	default:
		return "***unknown driver***"
	}
}
