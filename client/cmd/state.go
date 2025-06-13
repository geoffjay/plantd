package cmd

import (
	"errors"

	"github.com/geoffjay/plantd/client/auth"
	plantd "github.com/geoffjay/plantd/core/service"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	serviceFlag string

	stateCmd = &cobra.Command{
		Use:   "state",
		Short: "Perform state related tasks",
	}
	stateGetCmd = &cobra.Command{
		Use:   "get",
		Short: "Get a state value",
		Long:  "Get a value from the state management service",
		Args:  cobra.ExactArgs(1),
		Run:   get,
	}
	stateSetCmd = &cobra.Command{
		Use:   "set",
		Short: "Set a state value by key",
		Long:  "Set a value by key in the state management service",
		Args:  cobra.ExactArgs(2),
		Run:   set,
	}
	stateListCmd = &cobra.Command{
		Use:   "list",
		Short: "List all keys in a service scope",
		Long:  "List all keys available in the specified service scope",
		Args:  cobra.NoArgs,
		Run:   list,
	}
	stateDeleteCmd = &cobra.Command{
		Use:   "delete",
		Short: "Delete a state value by key",
		Long:  "Delete a value by key from the state management service",
		Args:  cobra.ExactArgs(1),
		Run:   deleteKey,
	}
	stateCreateScopeCmd = &cobra.Command{
		Use:   "create-scope",
		Short: "Create a new service scope",
		Long:  "Create a new service scope in the state management service",
		Args:  cobra.NoArgs,
		Run:   createScope,
	}
	stateDeleteScopeCmd = &cobra.Command{
		Use:   "delete-scope",
		Short: "Delete an entire service scope",
		Long:  "Delete an entire service scope and all its data from the state management service",
		Args:  cobra.NoArgs,
		Run:   deleteScope,
	}
	stateListScopesCmd = &cobra.Command{
		Use:   "list-scopes",
		Short: "List all available service scopes",
		Long:  "List all available service scopes in the state management service",
		Args:  cobra.NoArgs,
		Run:   listScopes,
	}
)

func init() {
	stateCmd.AddCommand(stateGetCmd)
	stateCmd.AddCommand(stateSetCmd)
	stateCmd.AddCommand(stateListCmd)
	stateCmd.AddCommand(stateDeleteCmd)
	stateCmd.AddCommand(stateCreateScopeCmd)
	stateCmd.AddCommand(stateDeleteScopeCmd)
	stateCmd.AddCommand(stateListScopesCmd)

	// Add flags for service scope and authentication profile
	stateCmd.PersistentFlags().StringVar(&serviceFlag, "service", "org.plantd.Client", "Service scope for state operations")
	stateCmd.PersistentFlags().StringVar(&profileFlag, "profile", "default", "Authentication profile to use")
}

func get(_ *cobra.Command, args []string) {
	log.Println(endpoint)

	// Execute with authentication
	executeWithAuth(func(token string) error {
		client, err := plantd.NewClient(endpoint)
		if err != nil {
			return err
		}

		key := args[0]
		request := &plantd.RawRequest{
			"token":   token,       // NEW: Include authentication token
			"service": serviceFlag, // Use configurable service flag
			"key":     key,
		}
		response, err := client.SendRawRequest("org.plantd.State", "state-get", request)
		if err != nil {
			return err
		}

		log.Printf("%+v\n", response)
		return nil
	})
}

func set(_ *cobra.Command, args []string) {
	log.Println(endpoint)

	// Execute with authentication
	executeWithAuth(func(token string) error {
		client, err := plantd.NewClient(endpoint)
		if err != nil {
			return err
		}

		key := args[0]
		value := args[1]
		request := &plantd.RawRequest{
			"token":   token,       // NEW: Include authentication token
			"service": serviceFlag, // Use configurable service flag
			"key":     key,
			"value":   value,
		}
		response, err := client.SendRawRequest("org.plantd.State", "state-set", request)
		if err != nil {
			return err
		}

		log.Printf("%+v\n", response)
		return nil
	})
}

// executeWithAuth handles authentication for state operations with automatic token refresh
func executeWithAuth(operation func(token string) error) {
	tokenMgr := auth.NewTokenManager()

	// Get valid token
	token, err := tokenMgr.GetValidToken(profileFlag)
	if err != nil {
		if errors.Is(err, auth.ErrNotAuthenticated) {
			log.Fatal("Authentication required. Please run 'plant auth login' first.")
		}
		if errors.Is(err, auth.ErrTokenExpired) {
			log.Fatal("Token expired. Please run 'plant auth refresh' or 'plant auth login'.")
		}
		log.Fatal(err)
	}

	// Execute operation with current token
	err = operation(token)
	if err != nil {
		if isAuthError(err) {
			// Try token refresh once
			log.Println("Token may be expired, attempting refresh...")

			profile, profileErr := tokenMgr.GetProfile(profileFlag)
			if profileErr != nil {
				log.Fatal("Failed to get profile for refresh. Please login again with 'plant auth login'.")
			}

			// Attempt refresh using the identity client
			ctx := getContext()
			clientConfig := getIdentityClientConfig(profile.Endpoint)
			client, clientErr := getIdentityClient(clientConfig)
			if clientErr != nil {
				log.Fatal("Failed to create identity client for refresh. Please login again with 'plant auth login'.")
			}
			defer func() {
				if closeErr := client.Close(); closeErr != nil {
					log.WithError(closeErr).Warn("Failed to close identity client")
				}
			}()

			response, refreshErr := client.RefreshToken(ctx, profile.RefreshToken)
			if refreshErr != nil {
				log.Fatal("Failed to refresh token. Please login again with 'plant auth login'.")
			}

			// Update stored tokens
			profile.AccessToken = response.AccessToken
			profile.RefreshToken = response.RefreshToken
			profile.ExpiresAt = response.ExpiresAt

			if storeErr := tokenMgr.StoreTokens(profileFlag, profile); storeErr != nil {
				log.Printf("Warning: Failed to store refreshed token: %v\n", storeErr)
			}

			// Retry operation with new token
			log.Println("Token refreshed, retrying operation...")
			err = operation(response.AccessToken)
		}

		if err != nil {
			log.Fatal(formatError(err))
		}
	}
}

// isAuthError checks if an error is related to authentication
func isAuthError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	return containsAny(errStr, []string{
		"authentication failed",
		"invalid token",
		"token expired",
		"unauthorized",
		"401",
		"403",
	})
}

// isPermissionError checks if an error is related to permissions
func isPermissionError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	return containsAny(errStr, []string{
		"permission denied",
		"insufficient permissions",
		"forbidden",
		"403",
	})
}

// isNetworkError checks if an error is related to network connectivity
func isNetworkError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	return containsAny(errStr, []string{
		"connection refused",
		"network unreachable",
		"timeout",
		"no such host",
		"connection failed",
	})
}

// formatError provides user-friendly error messages
func formatError(err error) string {
	switch {
	case isAuthError(err):
		return "Authentication failed. Please run 'plant auth login' to reauthenticate."
	case isPermissionError(err):
		return "Permission denied. You don't have access to this resource."
	case isNetworkError(err):
		return "Unable to connect to plantd services. Please check your configuration and ensure services are running."
	default:
		return err.Error()
	}
}

// containsAny checks if a string contains any of the given substrings
func containsAny(str string, substrings []string) bool {
	for _, substr := range substrings {
		if len(str) >= len(substr) {
			for i := 0; i <= len(str)-len(substr); i++ {
				if str[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}

func list(_ *cobra.Command, _ []string) {
	log.Println(endpoint)

	// Execute with authentication
	executeWithAuth(func(token string) error {
		client, err := plantd.NewClient(endpoint)
		if err != nil {
			return err
		}

		request := &plantd.RawRequest{
			"token":   token,       // Include authentication token
			"service": serviceFlag, // Use configurable service flag
		}
		response, err := client.SendRawRequest("org.plantd.State", "list-keys", request)
		if err != nil {
			return err
		}

		log.Printf("%+v\n", response)
		return nil
	})
}

func deleteKey(_ *cobra.Command, args []string) {
	log.Println(endpoint)

	// Execute with authentication
	executeWithAuth(func(token string) error {
		client, err := plantd.NewClient(endpoint)
		if err != nil {
			return err
		}

		key := args[0]
		request := &plantd.RawRequest{
			"token":   token,       // Include authentication token
			"service": serviceFlag, // Use configurable service flag
			"key":     key,
		}
		response, err := client.SendRawRequest("org.plantd.State", "delete", request)
		if err != nil {
			return err
		}

		log.Printf("%+v\n", response)
		return nil
	})
}

func createScope(_ *cobra.Command, _ []string) {
	log.Println(endpoint)

	// Execute with authentication
	executeWithAuth(func(token string) error {
		client, err := plantd.NewClient(endpoint)
		if err != nil {
			return err
		}

		request := &plantd.RawRequest{
			"token":   token,       // Include authentication token
			"service": serviceFlag, // Use configurable service flag
		}
		response, err := client.SendRawRequest("org.plantd.State", "create-scope", request)
		if err != nil {
			return err
		}

		log.Printf("%+v\n", response)
		return nil
	})
}

func deleteScope(_ *cobra.Command, _ []string) {
	log.Println(endpoint)

	// Execute with authentication
	executeWithAuth(func(token string) error {
		client, err := plantd.NewClient(endpoint)
		if err != nil {
			return err
		}

		request := &plantd.RawRequest{
			"token":   token,       // Include authentication token
			"service": serviceFlag, // Use configurable service flag
		}
		response, err := client.SendRawRequest("org.plantd.State", "delete-scope", request)
		if err != nil {
			return err
		}

		log.Printf("%+v\n", response)
		return nil
	})
}

func listScopes(_ *cobra.Command, _ []string) {
	log.Println(endpoint)

	// Execute with authentication
	executeWithAuth(func(token string) error {
		client, err := plantd.NewClient(endpoint)
		if err != nil {
			return err
		}

		request := &plantd.RawRequest{
			"token": token, // Include authentication token
		}
		response, err := client.SendRawRequest("org.plantd.State", "list-scopes", request)
		if err != nil {
			return err
		}

		log.Printf("%+v\n", response)
		return nil
	})
}
