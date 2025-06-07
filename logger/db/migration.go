// Package db provides database migration functionality.
package db

// Migration defines the interface for database migrations.
type Migration interface {
	Up() error
	Down() error
}
