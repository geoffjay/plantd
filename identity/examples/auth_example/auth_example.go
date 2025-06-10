package main

import (
	"context"
	"fmt"
	"log"

	"github.com/geoffjay/plantd/identity/internal/auth"
	"github.com/geoffjay/plantd/identity/internal/config"
	"github.com/geoffjay/plantd/identity/internal/models"
	"github.com/geoffjay/plantd/identity/internal/repositories"
	"github.com/geoffjay/plantd/identity/internal/services"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	// Initialize logger
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	// Load configuration
	cfg := config.GetConfig()

	// Initialize database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Auto-migrate the schema
	if err := db.AutoMigrate(&models.User{}, &models.Organization{}, &models.Role{}); err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	// Initialize repositories
	userRepo := repositories.NewUserRepository(db)
	orgRepo := repositories.NewOrganizationRepository(db)
	roleRepo := repositories.NewRoleRepository(db)

	// Initialize services
	userService := services.NewUserService(userRepo, roleRepo, orgRepo)

	// Initialize authentication services
	authService := auth.NewAuthService(
		cfg.ToAuthConfig(),
		userRepo,
		userService,
		logger,
	)

	registrationService := auth.NewRegistrationService(
		cfg.ToRegistrationConfig(),
		userRepo,
		userService,
		auth.NewPasswordValidator(cfg.ToPasswordConfig()),
		auth.NewJWTManager(cfg.ToJWTConfig(), auth.NewInMemoryBlacklist()),
		auth.NewRateLimiter(cfg.ToRateLimiterConfig()),
		logger,
	)

	ctx := context.Background()

	// Example 1: User Registration
	fmt.Println("=== User Registration Example ===")
	registrationReq := &auth.RegistrationRequest{
		Email:     "john.doe@example.com",
		Username:  "johndoe",
		Password:  "MySecure173!Auth",
		FirstName: "John",
		LastName:  "Doe",
		IPAddress: "192.168.1.100",
		UserAgent: "Example-Client/1.0",
	}

	registrationResp, err := registrationService.Register(ctx, registrationReq)
	if err != nil {
		log.Printf("Registration failed: %v", err)
		return
	}

	fmt.Printf("Registration successful: %s\n", registrationResp.Message)
	fmt.Printf("User ID: %d\n", registrationResp.User.ID)
	fmt.Printf("Requires verification: %t\n", registrationResp.RequiresVerification)

	// Example 2: Email Verification (if required)
	if registrationResp.RequiresVerification && registrationResp.EmailVerification != "" {
		fmt.Println("\n=== Email Verification Example ===")
		verificationReq := &auth.EmailVerificationRequest{
			Token:     registrationResp.EmailVerification,
			IPAddress: "192.168.1.100",
		}

		if err := registrationService.VerifyEmail(ctx, verificationReq); err != nil {
			log.Printf("Email verification failed: %v", err)
		} else {
			fmt.Println("Email verification successful")
		}
	}

	// Example 3: User Login
	fmt.Println("\n=== User Login Example ===")
	loginReq := &auth.AuthRequest{
		Identifier: "john.doe@example.com", // Can be email or username
		Password:   "MySecure173!Auth",
		IPAddress:  "192.168.1.100",
		UserAgent:  "Example-Client/1.0",
	}

	loginResp, err := authService.Login(ctx, loginReq)
	if err != nil {
		log.Printf("Login failed: %v", err)
		return
	}

	fmt.Printf("Login successful for user: %s\n", loginResp.User.Email)
	fmt.Printf("Access token expires at: %v\n", loginResp.ExpiresAt)
	fmt.Printf("Token type: %s\n", loginResp.TokenPair.TokenType)

	// Example 4: Token Validation
	fmt.Println("\n=== Token Validation Example ===")
	claims, err := authService.ValidateToken(ctx, loginResp.TokenPair.AccessToken)
	if err != nil {
		log.Printf("Token validation failed: %v", err)
	} else {
		fmt.Printf("Token is valid for user: %s (ID: %d)\n", claims.Email, claims.UserID)
		fmt.Printf("User roles: %v\n", claims.Roles)
		fmt.Printf("User permissions: %v\n", claims.Permissions)
	}

	// Example 5: Token Refresh
	fmt.Println("\n=== Token Refresh Example ===")
	refreshReq := &auth.RefreshRequest{
		RefreshToken: loginResp.TokenPair.RefreshToken,
		IPAddress:    "192.168.1.100",
	}

	newTokenPair, err := authService.RefreshToken(ctx, refreshReq)
	if err != nil {
		log.Printf("Token refresh failed: %v", err)
	} else {
		fmt.Printf("Token refresh successful\n")
		fmt.Printf("New access token expires at: %v\n", newTokenPair.AccessTokenExpiresAt)
	}

	// Example 6: Password Change
	fmt.Println("\n=== Password Change Example ===")
	err = authService.ChangePassword(ctx, registrationResp.User.ID, "MySecure173!Auth", "NewStr0ngP@ss!")
	if err != nil {
		log.Printf("Password change failed: %v", err)
	} else {
		fmt.Println("Password change successful")
	}

	// Example 7: Profile Update
	fmt.Println("\n=== Profile Update Example ===")
	profileReq := &auth.ProfileUpdateRequest{
		FirstName: "Jonathan",
		LastName:  "Doe",
		Username:  "jonathan.doe",
		IPAddress: "192.168.1.100",
	}

	updatedUser, err := registrationService.UpdateProfile(ctx, registrationResp.User.ID, profileReq)
	if err != nil {
		log.Printf("Profile update failed: %v", err)
	} else {
		fmt.Printf("Profile updated successfully\n")
		fmt.Printf("New username: %s\n", updatedUser.Username)
		fmt.Printf("Full name: %s\n", updatedUser.GetFullName())
	}

	// Example 8: Password Reset Flow
	fmt.Println("\n=== Password Reset Example ===")
	resetReq := &auth.PasswordResetRequest{
		Email:     "john.doe@example.com",
		IPAddress: "192.168.1.100",
	}

	err = registrationService.InitiatePasswordReset(ctx, resetReq)
	if err != nil {
		log.Printf("Password reset initiation failed: %v", err)
	} else {
		fmt.Println("Password reset initiated successfully")
		fmt.Println("In a real application, a reset email would be sent")
	}

	// Example 9: Password Strength Check
	fmt.Println("\n=== Password Strength Example ===")
	passwords := []string{
		"weak",
		"StrongerPassword123",
		"VeryStrongPassword123!@#",
		"password123", // Common pattern
	}

	for _, pwd := range passwords {
		strength := authService.GetPasswordStrength(pwd)
		fmt.Printf("Password '%s' strength: %d/100\n", pwd, strength)
	}

	// Example 10: Security Statistics
	fmt.Println("\n=== Security Statistics Example ===")
	stats := authService.GetSecurityStats()
	fmt.Printf("Security stats: %+v\n", stats)

	// Example 11: Logout
	fmt.Println("\n=== Logout Example ===")
	if newTokenPair != nil {
		err = authService.Logout(ctx, newTokenPair.AccessToken)
		if err != nil {
			log.Printf("Logout failed: %v", err)
		} else {
			fmt.Println("Logout successful")
		}
	}

	// Example 12: Failed Login Attempts (Rate Limiting)
	fmt.Println("\n=== Rate Limiting Example ===")
	for i := 0; i < 3; i++ {
		badLoginReq := &auth.AuthRequest{
			Identifier: "john.doe@example.com",
			Password:   "WrongPassword",
			IPAddress:  "192.168.1.200", // Different IP
			UserAgent:  "Example-Client/1.0",
		}

		_, err := authService.Login(ctx, badLoginReq)
		if err != nil {
			fmt.Printf("Failed login attempt %d: %v\n", i+1, err)
		}
	}

	// Clean up
	authService.Stop()

	fmt.Println("\n=== Authentication Example Complete ===")
}
