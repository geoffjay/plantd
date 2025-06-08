// Package testhelpers provides utilities for testing the identity service.
package testhelpers

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/geoffjay/plantd/identity/internal/models"
)

// SetupTestDB creates an in-memory SQLite database for testing.
func SetupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	// Run auto-migration
	err = models.AutoMigrate(db)
	require.NoError(t, err)

	return db
}

// CleanupTestDB cleans up the test database.
func CleanupTestDB(t *testing.T, db *gorm.DB) {
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())
}

// CreateTestUser creates a test user for testing purposes.
func CreateTestUser(t *testing.T, db *gorm.DB, overrides ...func(*models.User)) *models.User {
	user := &models.User{
		Email:          "test@example.com",
		Username:       "testuser",
		HashedPassword: "hashedpassword123",
		FirstName:      "Test",
		LastName:       "User",
		IsActive:       true,
		EmailVerified:  false,
	}

	for _, override := range overrides {
		override(user)
	}

	err := db.Create(user).Error
	require.NoError(t, err)

	return user
}

// CreateTestOrganization creates a test organization for testing purposes.
func CreateTestOrganization(t *testing.T, db *gorm.DB, overrides ...func(*models.Organization)) *models.Organization {
	org := &models.Organization{
		Name:        "Test Organization",
		Slug:        "test-org",
		Description: "A test organization",
		IsActive:    true,
	}

	for _, override := range overrides {
		override(org)
	}

	err := db.Create(org).Error
	require.NoError(t, err)

	return org
}

// CreateTestRole creates a test role for testing purposes.
func CreateTestRole(t *testing.T, db *gorm.DB, overrides ...func(*models.Role)) *models.Role {
	role := &models.Role{
		Name:        "Test Role",
		Description: "A test role",
		Permissions: `["read", "write"]`,
		Scope:       models.RoleScopeOrganization,
	}

	for _, override := range overrides {
		override(role)
	}

	err := db.Create(role).Error
	require.NoError(t, err)

	return role
}

// WithEmail returns a function that sets the email for a user.
func WithEmail(email string) func(*models.User) {
	return func(u *models.User) {
		u.Email = email
	}
}

// WithUsername returns a function that sets the username for a user.
func WithUsername(username string) func(*models.User) {
	return func(u *models.User) {
		u.Username = username
	}
}

// WithOrgSlug returns a function that sets the slug for an organization.
func WithOrgSlug(slug string) func(*models.Organization) {
	return func(o *models.Organization) {
		o.Slug = slug
	}
}

// WithRoleName returns a function that sets the name for a role.
func WithRoleName(name string) func(*models.Role) {
	return func(r *models.Role) {
		r.Name = name
	}
}

// WithRoleScope returns a function that sets the scope for a role.
func WithRoleScope(scope models.RoleScope) func(*models.Role) {
	return func(r *models.Role) {
		r.Scope = scope
	}
}
