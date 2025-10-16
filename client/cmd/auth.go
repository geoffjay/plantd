package cmd

import (
	"context"
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/geoffjay/plantd/client/auth"
	"github.com/geoffjay/plantd/client/internal/grpc"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	authCmd = &cobra.Command{
		Use:   "auth",
		Short: "Authentication management",
		Long:  "Manage authentication tokens and user sessions for plantd services",
	}

	authLoginCmd = &cobra.Command{
		Use:   "login",
		Short: "Login to plantd system",
		Long:  "Authenticate with the plantd identity service and store tokens locally",
		Run:   loginHandler,
	}

	authLogoutCmd = &cobra.Command{
		Use:   "logout",
		Short: "Logout from plantd system",
		Long:  "Clear stored authentication tokens and end session",
		Run:   logoutHandler,
	}

	authStatusCmd = &cobra.Command{
		Use:   "status",
		Short: "Show authentication status",
		Long:  "Display current authentication status and token information",
		Run:   statusHandler,
	}

	authRefreshCmd = &cobra.Command{
		Use:   "refresh",
		Short: "Refresh authentication token",
		Long:  "Refresh the current access token using the stored refresh token",
		Run:   refreshHandler,
	}

	authWhoamiCmd = &cobra.Command{
		Use:   "whoami",
		Short: "Show current user information",
		Long:  "Display information about the currently authenticated user",
		Run:   whoamiHandler,
	}

	// Command flags
	emailFlag        string
	passwordFlag     string
	profileFlag      string
	forceFlag        bool
	grpcEndpointFlag string
)

func init() {
	// Add subcommands
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authLogoutCmd)
	authCmd.AddCommand(authStatusCmd)
	authCmd.AddCommand(authRefreshCmd)
	authCmd.AddCommand(authWhoamiCmd)

	// Login flags
	authLoginCmd.Flags().StringVarP(&emailFlag, "email", "e", "", "Email address for authentication")
	authLoginCmd.Flags().StringVarP(&passwordFlag, "password", "p", "", "Password (will prompt if not provided)")
	authLoginCmd.Flags().BoolVar(&forceFlag, "force", false, "Force login even if already authenticated")

	// Global flags for profile selection and gRPC endpoint
	authCmd.PersistentFlags().StringVar(&profileFlag, "profile", "default", "Authentication profile to use")
	authCmd.PersistentFlags().StringVar(&grpcEndpointFlag, "grpc-endpoint", "", "gRPC gateway endpoint (defaults to config)")
}

// createGRPCIdentityClient creates a gRPC identity client.
func createGRPCIdentityClient() (*grpc.IdentityClient, error) {
	gatewayEndpoint := grpcEndpointFlag
	if gatewayEndpoint == "" {
		gatewayEndpoint = GetGRPCGatewayEndpoint()
	}

	// Create client configuration
	config := &grpc.IdentityClientConfig{
		BaseURL: gatewayEndpoint,
		Timeout: 30 * time.Second,
	}

	return grpc.NewIdentityClient(config), nil
}

func loginHandler(_ *cobra.Command, _ []string) {
	ctx := context.Background()
	tokenMgr := auth.NewTokenManager()

	// Check if already authenticated unless force flag is set
	if !forceFlag {
		if token, err := tokenMgr.GetValidToken(profileFlag); err == nil && token != "" {
			log.Info("Already authenticated. Use --force to reauthenticate.")
			return
		}
	}

	// Get email if not provided
	if emailFlag == "" {
		log.Info("Email: ")
		if _, err := fmt.Scanln(&emailFlag); err != nil {
			log.WithError(err).Fatal("Error reading email")
		}
	}

	// Get password if not provided
	password := passwordFlag
	if password == "" {
		log.Info("Password: ")
		passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			log.WithError(err).Fatal("Error reading password")
		}
		password = string(passwordBytes)
		log.Info("") // Add newline after password input
	}

	// Create gRPC identity client
	client, err := createGRPCIdentityClient()
	if err != nil {
		log.WithError(err).Fatal("Failed to create gRPC identity client")
	}

	// Perform login via gRPC
	loginResp, err := client.Login(ctx, emailFlag, password)
	if err != nil {
		log.WithError(err).Fatal("Login failed")
	}

	// Store tokens locally
	tokenData := &auth.TokenProfile{
		AccessToken:  loginResp.AccessToken,
		RefreshToken: loginResp.RefreshToken,
		ExpiresAt:    loginResp.ExpiresAt,
		UserEmail:    loginResp.Email,
		Endpoint:     grpcEndpointFlag,
	}

	if err := tokenMgr.StoreTokens(profileFlag, tokenData); err != nil {
		log.WithError(err).Fatal("Failed to save authentication tokens")
	}

	log.WithFields(log.Fields{
		"email":   loginResp.Email,
		"user_id": loginResp.UserID,
		"profile": profileFlag,
	}).Info("Login successful")
}

func logoutHandler(_ *cobra.Command, _ []string) {
	ctx := context.Background()
	tokenMgr := auth.NewTokenManager()

	// Get current token if it exists
	token, err := tokenMgr.GetValidToken(profileFlag)
	if err != nil {
		log.Infof("No active session found for profile '%s'", profileFlag)
		return
	}

	// Create gRPC identity client and logout
	client, err := createGRPCIdentityClient()
	if err != nil {
		log.WithError(err).Error("Failed to create gRPC identity client")
	} else {
		// Attempt to logout from server (invalidate token)
		if err := client.Logout(ctx, token); err != nil {
			log.WithError(err).Warn("Failed to logout from server")
		}
	}

	// Clear local tokens regardless of server logout result
	if err := tokenMgr.ClearTokens(profileFlag); err != nil {
		log.WithError(err).Fatal("Failed to clear local tokens")
	}

	log.Infof("Logged out from profile '%s'", profileFlag)
}

func statusHandler(_ *cobra.Command, _ []string) {
	tokenMgr := auth.NewTokenManager()

	profile, err := tokenMgr.GetProfile(profileFlag)
	if err != nil {
		log.Infof("Profile '%s': Not authenticated", profileFlag)
		return
	}

	log.Infof("Profile: %s", profileFlag)
	log.Infof("User: %s", profile.UserEmail)
	log.Infof("Endpoint: %s", profile.Endpoint)
	log.Infof("Token Status: %s", profile.TokenStatus())
	log.Infof("Expires At: %s", profile.ExpiresAtFormatted())

	// Check if token is actually valid by testing it via gRPC
	if token, err := tokenMgr.GetValidToken(profileFlag); err == nil && token != "" {
		ctx := context.Background()
		client, err := createGRPCIdentityClient()
		if err != nil {
			log.Infof("Authentication: ✗ Cannot verify (gRPC client error)")
			return
		}

		_, err = client.ValidateToken(ctx, token)
		if err != nil {
			log.Infof("Authentication: ✗ Invalid or expired")
		} else {
			log.Infof("Authentication: ✓ Valid")
		}
	} else {
		log.Infof("Authentication: ✗ Invalid or expired")
	}
}

func refreshHandler(_ *cobra.Command, _ []string) {
	ctx := context.Background()
	tokenMgr := auth.NewTokenManager()

	profile, err := tokenMgr.GetProfile(profileFlag)
	if err != nil {
		log.Errorf("No authentication found for profile '%s'. Please login first.", profileFlag)
		os.Exit(1)
	}

	// Create gRPC identity client
	client, err := createGRPCIdentityClient()
	if err != nil {
		log.WithError(err).Fatal("Failed to create gRPC identity client")
	}

	// Refresh token
	log.Info("Refreshing authentication token...")
	response, err := client.RefreshToken(ctx, profile.RefreshToken)
	if err != nil {
		log.WithError(err).Fatal("Failed to refresh token")
		log.Info("Please login again with 'plant auth login'")
		os.Exit(1)
	}

	// Update stored tokens
	profile.AccessToken = response.AccessToken
	profile.RefreshToken = response.RefreshToken
	profile.ExpiresAt = response.ExpiresAt

	if err := tokenMgr.StoreTokens(profileFlag, profile); err != nil {
		log.WithError(err).Fatal("Failed to store refreshed tokens")
		os.Exit(1)
	}

	log.Info("Token refreshed successfully")
	log.Infof("New expiry: %s", profile.ExpiresAtFormatted())
}

func whoamiHandler(_ *cobra.Command, _ []string) {
	ctx := context.Background()
	tokenMgr := auth.NewTokenManager()

	token, err := tokenMgr.GetValidToken(profileFlag)
	if err != nil {
		log.Error("Not authenticated. Please login first with 'plant auth login'")
		os.Exit(1)
	}

	// Create gRPC identity client
	client, err := createGRPCIdentityClient()
	if err != nil {
		log.WithError(err).Fatal("Failed to create gRPC identity client")
	}

	// Validate token to get current user information via gRPC
	response, err := client.ValidateToken(ctx, token)
	if err != nil {
		log.WithError(err).Fatal("Failed to validate token")
		log.Info("Please login again with 'plant auth login'")
		os.Exit(1)
	}

	if !response.Valid {
		log.Error("Token is not valid. Please login again.")
		os.Exit(1)
	}

	if response.UserID != nil {
		log.Infof("User ID: %d", *response.UserID)
	}
	log.Infof("Email: %s", response.Email)
	log.Infof("Roles: %v", response.Roles)
	log.Infof("Permissions: %v", response.Permissions)
	if response.ExpiresAt != nil {
		log.Infof("Token Expires: %s", auth.FormatUnixTimestamp(*response.ExpiresAt))
	}
}

// Helper functions for state command integration

// getContext returns a context for operations
func getContext() context.Context {
	return context.Background()
}
