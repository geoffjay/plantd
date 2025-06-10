// Package repositories provides the repository implementations for the identity service.
package repositories

import (
	"gorm.io/gorm"
)

// Container holds all repository instances for dependency injection.
type Container struct {
	User         UserRepository
	Organization OrganizationRepository
	Role         RoleRepository
}

// NewContainer creates a new repository container with all repository implementations.
func NewContainer(db *gorm.DB) *Container {
	return &Container{
		User:         NewUserRepository(db),
		Organization: NewOrganizationRepository(db),
		Role:         NewRoleRepository(db),
	}
}
