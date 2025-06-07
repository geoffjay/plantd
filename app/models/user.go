// Package models provides data model definitions.
package models

// User represents a user in the system.
type User struct {
	ID       uint   `json:"id"`
	Email    string `json:"email" form:"email"`
	Password string `json:"password" form:"password"`
}
