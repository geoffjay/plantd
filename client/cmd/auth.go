package cmd

import (
	"context"
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/geoffjay/plantd/client/auth"
	identityClient "github.com/geoffjay/plantd/identity/pkg/client"

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
	emailFlag    string
	passwordFlag string
	profileFlag  string
	forceFlag    bool
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

	// Global flags for profile selection
	authCmd.PersistentFlags().StringVar(&profileFlag, "profile", "default", "Authentication profile to use")
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

	// Get identity service endpoint from config
	identityEndpoint := getIdentityEndpoint()

	// Create identity client
	clientConfig := getIdentityClientConfig(identityEndpoint)
	client, err := identityClient.NewClient(clientConfig)
	if err != nil {
		log.WithError(err).Fatal("Failed to create identity client")
	}
	defer func() {
		if closeErr := client.Close(); closeErr != nil {
			log.WithError(closeErr).Warn("Failed to close identity client")
		}
	}()

	// Attempt login
	log.Info("Authenticating...")
	response, err := client.LoginWithEmail(ctx, emailFlag, password)
	if err != nil {
		log.WithError(err).Fatal("Authentication failed")
	}

	// Store tokens
	profile := &auth.TokenProfile{
		AccessToken:  response.AccessToken,
		RefreshToken: response.RefreshToken,
		ExpiresAt:    response.ExpiresAt,
		UserEmail:    response.User.Email,
		Endpoint:     identityEndpoint,
	}

	if err := tokenMgr.StoreTokens(profileFlag, profile); err != nil {
		log.WithError(err).Fatal("Failed to store authentication tokens")
	}

	log.Infof("Successfully authenticated as %s", response.User.Email)
	log.Infof("Access token expires at: %s", profile.ExpiresAtFormatted())
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

	// Get identity service endpoint
	identityEndpoint := getIdentityEndpoint()

	// Create identity client and logout
	clientConfig := getIdentityClientConfig(identityEndpoint)
	client, err := identityClient.NewClient(clientConfig)
	if err != nil {
		log.WithError(err).Error("Failed to create identity client")
	} else {
		defer func() {
			if closeErr := client.Close(); closeErr != nil {
				log.WithError(closeErr).Warn("Failed to close identity client")
			}
		}()
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
	log.Infof("Identity Endpoint: %s", profile.Endpoint)
	log.Infof("Token Status: %s", profile.TokenStatus())
	log.Infof("Expires At: %s", profile.ExpiresAtFormatted())

	// Check if token is actually valid by testing it
	if token, err := tokenMgr.GetValidToken(profileFlag); err == nil && token != "" {
		log.Infof("Authentication: ✓ Valid")
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

	// Create identity client
	clientConfig := getIdentityClientConfig(profile.Endpoint)
	client, err := identityClient.NewClient(clientConfig)
	if err != nil {
		log.WithError(err).Fatal("Failed to create identity client")
		os.Exit(1)
	}
	defer func() {
		if closeErr := client.Close(); closeErr != nil {
			log.WithError(closeErr).Warn("Failed to close identity client")
		}
	}()

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

	profile, err := tokenMgr.GetProfile(profileFlag)
	if err != nil {
		log.WithError(err).Fatal("Failed to get profile information")
		os.Exit(1)
	}

	// Create identity client
	clientConfig := getIdentityClientConfig(profile.Endpoint)
	client, err := identityClient.NewClient(clientConfig)
	if err != nil {
		log.WithError(err).Fatal("Failed to create identity client")
		os.Exit(1)
	}
	defer func() {
		if closeErr := client.Close(); closeErr != nil {
			log.WithError(closeErr).Warn("Failed to close identity client")
		}
	}()

	// Validate token to get current user information
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

	log.Infof("User ID: %d", *response.UserID)
	log.Infof("Email: %s", response.Email)
	log.Infof("Roles: %v", response.Roles)
	log.Infof("Permissions: %v", response.Permissions)
	if response.ExpiresAt != nil {
		log.Infof("Token Expires: %s", auth.FormatUnixTimestamp(*response.ExpiresAt))
	}
}

// getIdentityEndpoint returns the identity service endpoint from configuration
func getIdentityEndpoint() string {
	// Use configuration system to get the identity endpoint for the current profile
	return GetIdentityEndpoint(profileFlag)
}

// Helper functions for state command integration

// getContext returns a context for operations
func getContext() context.Context {
	return context.Background()
}

// getIdentityClientConfig creates a client config for the given endpoint
func getIdentityClientConfig(identityEndpoint string) *identityClient.Config {
	return &identityClient.Config{
		BrokerEndpoint: identityEndpoint,
		Timeout:        30 * time.Second, // Set explicit timeout
	}
}

// getIdentityClient creates a new identity client with the given config
func getIdentityClient(config *identityClient.Config) (*identityClient.Client, error) {
	return identityClient.NewClient(config)
}
