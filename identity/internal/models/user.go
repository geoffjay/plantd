package models

import (
	"strings"
	"time"

	"gorm.io/gorm"
)

// User represents a user in the identity system.
type User struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	Email           string         `gorm:"uniqueIndex;not null;size:255" json:"email"`
	Username        string         `gorm:"uniqueIndex;not null;size:100" json:"username"`
	HashedPassword  string         `gorm:"not null;size:255" json:"-"`
	FirstName       string         `gorm:"size:100" json:"first_name"`
	LastName        string         `gorm:"size:100" json:"last_name"`
	IsActive        bool           `gorm:"default:true" json:"is_active"`
	EmailVerified   bool           `gorm:"default:false" json:"email_verified"`
	EmailVerifiedAt *time.Time     `json:"email_verified_at"`
	LastLoginAt     *time.Time     `json:"last_login_at"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`

	// Many-to-many relationships
	Roles         []Role         `gorm:"many2many:user_roles;" json:"roles,omitempty"`
	Organizations []Organization `gorm:"many2many:user_organizations;" json:"organizations,omitempty"`
}

// TableName returns the table name for the User model.
func (User) TableName() string {
	return "users"
}

// BeforeCreate is a GORM hook that runs before creating a user.
func (u *User) BeforeCreate(_ *gorm.DB) error {
	// Ensure email is lowercase
	u.Email = strings.ToLower(u.Email)
	return nil
}

// BeforeUpdate is a GORM hook that runs before updating a user.
func (u *User) BeforeUpdate(_ *gorm.DB) error {
	// Ensure email is lowercase
	u.Email = strings.ToLower(u.Email)
	return nil
}

// GetFullName returns the user's full name.
func (u *User) GetFullName() string {
	if u.FirstName == "" && u.LastName == "" {
		return u.Username
	}
	return strings.TrimSpace(u.FirstName + " " + u.LastName)
}

// IsEmailVerified returns true if the user's email is verified.
func (u *User) IsEmailVerified() bool {
	return u.EmailVerified && u.EmailVerifiedAt != nil
}

// MarkEmailAsVerified marks the user's email as verified.
func (u *User) MarkEmailAsVerified() {
	u.EmailVerified = true
	now := time.Now()
	u.EmailVerifiedAt = &now
}

// UpdateLastLogin updates the user's last login timestamp.
func (u *User) UpdateLastLogin() {
	now := time.Now()
	u.LastLoginAt = &now
}
