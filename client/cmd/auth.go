package cmd

import (
	"context"
	"fmt"
	"os"
	"syscall"

	"github.com/geoffjay/plantd/client/auth"
	identityClient "github.com/geoffjay/plantd/identity/pkg/client"

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

func loginHandler(cmd *cobra.Command, args []string) {
	ctx := context.Background()
	tokenMgr := auth.NewTokenManager()

	// Check if already authenticated unless force flag is set
	if !forceFlag {
		if token, err := tokenMgr.GetValidToken(profileFlag); err == nil && token != "" {
			fmt.Println("Already authenticated. Use --force to reauthenticate.")
			return
		}
	}

	// Get email if not provided
	if emailFlag == "" {
		fmt.Print("Email: ")
		if _, err := fmt.Scanln(&emailFlag); err != nil {
			fmt.Fprintf(os.Stderr, "Error reading email: %v\n", err)
			os.Exit(1)
		}
	}

	// Get password if not provided
	password := passwordFlag
	if password == "" {
		fmt.Print("Password: ")
		passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading password: %v\n", err)
			os.Exit(1)
		}
		password = string(passwordBytes)
		fmt.Println() // Add newline after password input
	}

	// Get identity service endpoint from config
	identityEndpoint := getIdentityEndpoint()

	// Create identity client
	clientConfig := &identityClient.Config{
		BrokerEndpoint: identityEndpoint,
	}
	client, err := identityClient.NewClient(clientConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create identity client: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	// Attempt login
	fmt.Println("Authenticating...")
	response, err := client.LoginWithEmail(ctx, emailFlag, password)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Authentication failed: %v\n", err)
		os.Exit(1)
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
		fmt.Fprintf(os.Stderr, "Failed to store authentication tokens: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully authenticated as %s\n", response.User.Email)
	fmt.Printf("Access token expires at: %s\n", profile.ExpiresAtFormatted())
}

func logoutHandler(cmd *cobra.Command, args []string) {
	ctx := context.Background()
	tokenMgr := auth.NewTokenManager()

	// Get current token if it exists
	token, err := tokenMgr.GetValidToken(profileFlag)
	if err != nil {
		fmt.Printf("No active session found for profile '%s'\n", profileFlag)
		return
	}

	// Get identity service endpoint
	identityEndpoint := getIdentityEndpoint()

	// Create identity client and logout
	clientConfig := &identityClient.Config{
		BrokerEndpoint: identityEndpoint,
	}
	client, err := identityClient.NewClient(clientConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create identity client: %v\n", err)
	} else {
		defer client.Close()
		// Attempt to logout from server (invalidate token)
		if err := client.Logout(ctx, token); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to logout from server: %v\n", err)
		}
	}

	// Clear local tokens regardless of server logout result
	if err := tokenMgr.ClearTokens(profileFlag); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to clear local tokens: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Logged out from profile '%s'\n", profileFlag)
}

func statusHandler(cmd *cobra.Command, args []string) {
	tokenMgr := auth.NewTokenManager()

	profile, err := tokenMgr.GetProfile(profileFlag)
	if err != nil {
		fmt.Printf("Profile '%s': Not authenticated\n", profileFlag)
		return
	}

	fmt.Printf("Profile: %s\n", profileFlag)
	fmt.Printf("User: %s\n", profile.UserEmail)
	fmt.Printf("Identity Endpoint: %s\n", profile.Endpoint)
	fmt.Printf("Token Status: %s\n", profile.TokenStatus())
	fmt.Printf("Expires At: %s\n", profile.ExpiresAtFormatted())

	// Check if token is actually valid by testing it
	if token, err := tokenMgr.GetValidToken(profileFlag); err == nil && token != "" {
		fmt.Printf("Authentication: ✓ Valid\n")
	} else {
		fmt.Printf("Authentication: ✗ Invalid or expired\n")
	}
}

func refreshHandler(cmd *cobra.Command, args []string) {
	ctx := context.Background()
	tokenMgr := auth.NewTokenManager()

	profile, err := tokenMgr.GetProfile(profileFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "No authentication found for profile '%s'. Please login first.\n", profileFlag)
		os.Exit(1)
	}

	// Create identity client
	clientConfig := &identityClient.Config{
		BrokerEndpoint: profile.Endpoint,
	}
	client, err := identityClient.NewClient(clientConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create identity client: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	// Refresh token
	fmt.Println("Refreshing authentication token...")
	response, err := client.RefreshToken(ctx, profile.RefreshToken)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to refresh token: %v\n", err)
		fmt.Println("Please login again with 'plant auth login'")
		os.Exit(1)
	}

	// Update stored tokens
	profile.AccessToken = response.AccessToken
	profile.RefreshToken = response.RefreshToken
	profile.ExpiresAt = response.ExpiresAt

	if err := tokenMgr.StoreTokens(profileFlag, profile); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to store refreshed tokens: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Token refreshed successfully\n")
	fmt.Printf("New expiry: %s\n", profile.ExpiresAtFormatted())
}

func whoamiHandler(cmd *cobra.Command, args []string) {
	ctx := context.Background()
	tokenMgr := auth.NewTokenManager()

	token, err := tokenMgr.GetValidToken(profileFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Not authenticated. Please login first with 'plant auth login'\n")
		os.Exit(1)
	}

	profile, err := tokenMgr.GetProfile(profileFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get profile information: %v\n", err)
		os.Exit(1)
	}

	// Create identity client
	clientConfig := &identityClient.Config{
		BrokerEndpoint: profile.Endpoint,
	}
	client, err := identityClient.NewClient(clientConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create identity client: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	// Validate token to get current user information
	response, err := client.ValidateToken(ctx, token)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to validate token: %v\n", err)
		fmt.Println("Please login again with 'plant auth login'")
		os.Exit(1)
	}

	if !response.Valid {
		fmt.Fprintf(os.Stderr, "Token is not valid. Please login again.\n")
		os.Exit(1)
	}

	fmt.Printf("User ID: %d\n", *response.UserID)
	fmt.Printf("Email: %s\n", response.Email)
	fmt.Printf("Roles: %v\n", response.Roles)
	fmt.Printf("Permissions: %v\n", response.Permissions)
	if response.ExpiresAt != nil {
		fmt.Printf("Token Expires: %s\n", auth.FormatUnixTimestamp(*response.ExpiresAt))
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
	}
}

// getIdentityClient creates a new identity client with the given config
func getIdentityClient(config *identityClient.Config) (*identityClient.Client, error) {
	return identityClient.NewClient(config)
}
