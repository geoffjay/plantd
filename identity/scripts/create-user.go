package main

import (
	"context"
	"fmt"
	"log"
	"syscall"

	"github.com/geoffjay/plantd/identity/internal/auth"
	"github.com/geoffjay/plantd/identity/internal/config"
	"github.com/geoffjay/plantd/identity/internal/models"
	"github.com/geoffjay/plantd/identity/internal/repositories"
	"github.com/geoffjay/plantd/identity/internal/services"
	"github.com/sirupsen/logrus"
	"golang.org/x/term"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	fmt.Println("=== PlantD User Creation Script ===")

	// Get user input
	var email, username, password, firstName, lastName string

	fmt.Print("Email: ")
	if _, err := fmt.Scanln(&email); err != nil {
		log.Fatal("Error reading email:", err)
	}

	fmt.Print("Username: ")
	if _, err := fmt.Scanln(&username); err != nil {
		log.Fatal("Error reading username:", err)
	}

	fmt.Print("Password: ")
	passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		log.Fatal("Error reading password:", err)
	}
	password = string(passwordBytes)
	fmt.Println() // Add newline after password input

	fmt.Print("First Name (optional): ")
	fmt.Scanln(&firstName) // OK if this fails

	fmt.Print("Last Name (optional): ")
	fmt.Scanln(&lastName) // OK if this fails

	// Initialize logger
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	// Load configuration
	cfg := config.GetConfig()

	// Use SQLite database (same as the identity service uses)
	dbPath := cfg.Database.DSN
	if dbPath == "" {
		dbPath = "identity.db"
	}

	// Initialize database connection
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
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

	// Initialize registration service
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

	// Create user
	fmt.Println("Creating user account...")
	registrationReq := &auth.RegistrationRequest{
		Email:     email,
		Username:  username,
		Password:  password,
		FirstName: firstName,
		LastName:  lastName,
		IPAddress: "127.0.0.1",
		UserAgent: "PlantD-CreateUser-Script/1.0",
	}

	registrationResp, err := registrationService.Register(ctx, registrationReq)
	if err != nil {
		log.Fatalf("Registration failed: %v", err)
	}

	fmt.Printf("✅ Successfully created user account!\n")
	fmt.Printf("   Email: %s\n", registrationResp.User.Email)
	fmt.Printf("   Username: %s\n", registrationResp.User.Username)
	fmt.Printf("   User ID: %d\n", registrationResp.User.ID)

	if registrationResp.RequiresVerification {
		fmt.Println("   Note: Email verification may be required")
	}

	fmt.Println("\n🚀 You can now login using:")
	fmt.Printf("   plant auth login --email=%s\n", email)
	fmt.Println("   or")
	fmt.Printf("   plant auth login (and enter credentials interactively)\n")
}
