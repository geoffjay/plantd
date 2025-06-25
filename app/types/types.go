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

// PageData represents common data passed to page templates.
type PageData struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Keywords    string `json:"keywords"`
	Error       string `json:"error,omitempty"`
	Success     string `json:"success,omitempty"`
	CSRFToken   string `json:"csrf_token,omitempty"`
}
