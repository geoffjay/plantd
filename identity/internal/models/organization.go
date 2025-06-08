package models

import (
	"time"

	"gorm.io/gorm"
)

// Organization represents an organization in the identity system.
type Organization struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"not null;size:255" json:"name"`
	Slug        string         `gorm:"uniqueIndex;not null;size:100" json:"slug"`
	Description string         `gorm:"size:1000" json:"description"`
	IsActive    bool           `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	// Many-to-many relationships
	Users []User `gorm:"many2many:user_organizations;" json:"users,omitempty"`
	Roles []Role `gorm:"many2many:organization_roles;" json:"roles,omitempty"`
}

// TableName returns the table name for the Organization model.
func (Organization) TableName() string {
	return "organizations"
}
