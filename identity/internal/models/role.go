package models

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// RoleScope defines the scope of a role.
type RoleScope string

const (
	// RoleScopeGlobal indicates a role that applies globally.
	RoleScopeGlobal RoleScope = "global"
	// RoleScopeOrganization indicates a role that applies within an organization.
	RoleScopeOrganization RoleScope = "organization"
)

// Role represents a role in the identity system.
type Role struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"not null;size:100" json:"name"`
	Description string         `gorm:"size:500" json:"description"`
	Permissions string         `gorm:"type:text" json:"permissions"` // JSON array of permissions
	Scope       RoleScope      `gorm:"not null;default:'organization'" json:"scope"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	// Many-to-many relationships
	Users         []User         `gorm:"many2many:user_roles;" json:"users,omitempty"`
	Organizations []Organization `gorm:"many2many:organization_roles;" json:"organizations,omitempty"`
}

// TableName returns the table name for the Role model.
func (Role) TableName() string {
	return "roles"
}

// IsGlobal returns true if the role has global scope.
func (r *Role) IsGlobal() bool {
	return r.Scope == RoleScopeGlobal
}

// IsOrganizationScoped returns true if the role is organization-scoped.
func (r *Role) IsOrganizationScoped() bool {
	return r.Scope == RoleScopeOrganization
}

// HasPermission checks if the role has a specific permission.
func (r *Role) HasPermission(permission string) bool {
	if permission == "" {
		return false
	}

	permissions, err := r.GetPermissions()
	if err != nil {
		return false
	}

	for _, perm := range permissions {
		if perm == permission {
			return true
		}
	}

	return false
}

// GetPermissions returns the list of permissions for the role.
func (r *Role) GetPermissions() ([]string, error) {
	if r.Permissions == "" || r.Permissions == "null" {
		return []string{}, nil
	}

	var permissions []string
	err := json.Unmarshal([]byte(r.Permissions), &permissions)
	if err != nil {
		return nil, err
	}

	if permissions == nil {
		return []string{}, nil
	}

	return permissions, nil
}

// AddPermission adds a permission to the role if it doesn't already exist.
func (r *Role) AddPermission(permission string) error {
	if permission == "" {
		return nil
	}

	permissions, err := r.GetPermissions()
	if err != nil {
		return err
	}

	// Check if permission already exists
	for _, perm := range permissions {
		if perm == permission {
			return nil // Permission already exists
		}
	}

	// Add the new permission
	permissions = append(permissions, permission)

	// Marshal back to JSON
	permissionsJSON, err := json.Marshal(permissions)
	if err != nil {
		return err
	}

	r.Permissions = string(permissionsJSON)
	return nil
}

// RemovePermission removes a permission from the role.
func (r *Role) RemovePermission(permission string) error {
	permissions, err := r.GetPermissions()
	if err != nil {
		return err
	}

	// Find and remove the permission
	var newPermissions []string
	for _, perm := range permissions {
		if perm != permission {
			newPermissions = append(newPermissions, perm)
		}
	}

	// Marshal back to JSON
	permissionsJSON, err := json.Marshal(newPermissions)
	if err != nil {
		return err
	}

	r.Permissions = string(permissionsJSON)
	return nil
}
