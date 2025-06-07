// Package types provides type definitions for the application.
package types

// LoginRequest represents a user login request.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse represents a user login response.
type LoginResponse struct {
	Token string `json:"token"`
}
