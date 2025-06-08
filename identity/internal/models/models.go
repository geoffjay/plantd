// Package models provides the database models for the identity service.
package models

import (
	"gorm.io/gorm"

	log "github.com/sirupsen/logrus"
)

// AllModels returns a slice of all model types for auto-migration.
func AllModels() []interface{} {
	return []interface{}{
		&User{},
		&Organization{},
		&Role{},
	}
}

// AutoMigrate runs auto-migration for all models.
func AutoMigrate(db *gorm.DB) error {
	log.Info("running database auto-migration")

	models := AllModels()
	for _, model := range models {
		if err := db.AutoMigrate(model); err != nil {
			return err
		}
	}

	log.Info("database auto-migration completed successfully")
	return nil
}

// CreateIndexes creates additional indexes that aren't handled by GORM tags.
func CreateIndexes(_ *gorm.DB) error {
	log.Info("creating additional database indexes")

	// TODO: add `db` back as arg when this is implemented

	// Add any custom indexes here if needed
	// Example:
	// if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_users_email_verified ON users(email_verified)").Error; err != nil {
	//     return err
	// }

	log.Info("additional database indexes created successfully")
	return nil
}
