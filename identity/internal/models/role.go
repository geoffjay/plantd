package models

import (
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
