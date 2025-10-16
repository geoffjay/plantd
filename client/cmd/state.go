package cmd

import (
	"context"
	"encoding/json"
	"time"

	"github.com/geoffjay/plantd/client/auth"
	"github.com/geoffjay/plantd/client/internal/grpc"

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

// createGRPCStateClient creates a gRPC state client with authentication.
func createGRPCStateClient() (*grpc.StateClient, error) {
	stateEndpoint := GetStateEndpoint(profileFlag)

	// Use default timeout if config function not available
	timeoutDuration := 30 * time.Second

	// Create client configuration
	config := &grpc.StateClientConfig{
		BaseURL: stateEndpoint,
		Timeout: timeoutDuration,
		AuthFunc: func() string {
			tokenMgr := auth.NewTokenManager()
			token, _ := tokenMgr.GetValidToken(profileFlag)
			return token
		},
	}

	return grpc.NewStateClient(config), nil
}

func get(_ *cobra.Command, args []string) {
	ctx := context.Background()
	key := args[0]

	// Execute with authentication
	executeWithAuth(func(token string) error {
		client, err := createGRPCStateClient()
		if err != nil {
			return err
		}

		value, err := client.Get(ctx, serviceFlag, key)
		if err != nil {
			return err
		}

		// Format response similar to MDP version
		response := map[string]interface{}{
			"success": true,
			"data": map[string]interface{}{
				"scope": serviceFlag,
				"key":   key,
				"value": value,
			},
		}

		// Print JSON response
		responseJSON, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			log.Printf("Value: %s\n", value)
		} else {
			log.Printf("%s\n", responseJSON)
		}

		return nil
	})
}

func set(_ *cobra.Command, args []string) {
	ctx := context.Background()
	key := args[0]
	value := args[1]

	// Execute with authentication
	executeWithAuth(func(token string) error {
		client, err := createGRPCStateClient()
		if err != nil {
			return err
		}

		err = client.Set(ctx, serviceFlag, key, value)
		if err != nil {
			return err
		}

		// Format response similar to MDP version
		response := map[string]interface{}{
			"success": true,
			"message": "Value set successfully",
			"data": map[string]interface{}{
				"scope": serviceFlag,
				"key":   key,
				"value": value,
			},
		}

		// Print JSON response
		responseJSON, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			log.Printf("Set %s = %s successfully\n", key, value)
		} else {
			log.Printf("%s\n", responseJSON)
		}

		return nil
	})
}

func list(_ *cobra.Command, _ []string) {
	ctx := context.Background()

	// Execute with authentication
	executeWithAuth(func(token string) error {
		client, err := createGRPCStateClient()
		if err != nil {
			return err
		}

		keys, err := client.List(ctx, serviceFlag)
		if err != nil {
			return err
		}

		// Format response similar to MDP version
		response := map[string]interface{}{
			"success": true,
			"data": map[string]interface{}{
				"scope": serviceFlag,
				"keys":  keys,
				"count": len(keys),
			},
		}

		// Print JSON response
		responseJSON, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			log.Printf("Keys: %v\n", keys)
		} else {
			log.Printf("%s\n", responseJSON)
		}

		return nil
	})
}

func deleteKey(_ *cobra.Command, args []string) {
	ctx := context.Background()
	key := args[0]

	// Execute with authentication
	executeWithAuth(func(token string) error {
		client, err := createGRPCStateClient()
		if err != nil {
			return err
		}

		err = client.Delete(ctx, serviceFlag, key)
		if err != nil {
			return err
		}

		// Format response similar to MDP version
		response := map[string]interface{}{
			"success": true,
			"message": "Value deleted successfully",
			"data": map[string]interface{}{
				"scope": serviceFlag,
				"key":   key,
			},
		}

		// Print JSON response
		responseJSON, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			log.Printf("Deleted %s successfully\n", key)
		} else {
			log.Printf("%s\n", responseJSON)
		}

		return nil
	})
}

func listScopes(_ *cobra.Command, _ []string) {
	ctx := context.Background()

	// Execute with authentication
	executeWithAuth(func(token string) error {
		client, err := createGRPCStateClient()
		if err != nil {
			return err
		}

		scopes, err := client.ListScopes(ctx)
		if err != nil {
			return err
		}

		// Format response similar to MDP version
		response := map[string]interface{}{
			"success": true,
			"data": map[string]interface{}{
				"scopes": scopes,
				"count":  len(scopes),
			},
		}

		// Print JSON response
		responseJSON, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			log.Printf("Scopes: %v\n", scopes)
		} else {
			log.Printf("%s\n", responseJSON)
		}

		return nil
	})
}

func createScope(_ *cobra.Command, _ []string) {
	log.Error("Create scope operation not available in gRPC mode")
	log.Info("Scopes are created automatically when first used")
}

func deleteScope(_ *cobra.Command, _ []string) {
	log.Error("Delete scope operation not available in gRPC mode")
	log.Info("Please contact administrator for scope management")
}

// executeWithAuth handles authentication for state operations with automatic token refresh
func executeWithAuth(operation func(token string) error) {
	tokenMgr := auth.NewTokenManager()

	// Get valid token
	token, err := tokenMgr.GetValidToken(profileFlag)
	if err != nil {
		log.Fatal("Authentication required. Please run 'plant auth login' first.")
	}

	// Execute operation with current token
	err = operation(token)
	if err != nil {
		log.WithError(err).Fatal("Operation failed")
	}
}
