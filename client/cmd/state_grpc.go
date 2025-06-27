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
	// gRPC endpoint flag (overrides MDP endpoint)
	grpcEndpointFlag string
	useGRPCFlag      bool

	stateGRPCCmd = &cobra.Command{
		Use:   "state-grpc",
		Short: "Perform state related tasks using gRPC",
		Long:  "State management commands using gRPC through Traefik gateway",
	}
	stateGRPCGetCmd = &cobra.Command{
		Use:   "get <key>",
		Short: "Get a state value via gRPC",
		Long:  "Get a value from the state management service using gRPC",
		Args:  cobra.ExactArgs(1),
		Run:   getGRPC,
	}
	stateGRPCSetCmd = &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a state value by key via gRPC",
		Long:  "Set a value by key in the state management service using gRPC",
		Args:  cobra.ExactArgs(2),
		Run:   setGRPC,
	}
	stateGRPCListCmd = &cobra.Command{
		Use:   "list",
		Short: "List all keys in a service scope via gRPC",
		Long:  "List all keys available in the specified service scope using gRPC",
		Args:  cobra.NoArgs,
		Run:   listGRPC,
	}
	stateGRPCDeleteCmd = &cobra.Command{
		Use:   "delete <key>",
		Short: "Delete a state value by key via gRPC",
		Long:  "Delete a value by key from the state management service using gRPC",
		Args:  cobra.ExactArgs(1),
		Run:   deleteGRPC,
	}
	stateGRPCListScopesCmd = &cobra.Command{
		Use:   "list-scopes",
		Short: "List all available service scopes via gRPC",
		Long:  "List all available service scopes in the state management service using gRPC",
		Args:  cobra.NoArgs,
		Run:   listScopesGRPC,
	}
)

func init() {
	// Add gRPC subcommands
	stateGRPCCmd.AddCommand(stateGRPCGetCmd)
	stateGRPCCmd.AddCommand(stateGRPCSetCmd)
	stateGRPCCmd.AddCommand(stateGRPCListCmd)
	stateGRPCCmd.AddCommand(stateGRPCDeleteCmd)
	stateGRPCCmd.AddCommand(stateGRPCListScopesCmd)

	// Add flags for gRPC configuration
	stateGRPCCmd.PersistentFlags().StringVar(&grpcEndpointFlag, "grpc-endpoint", "http://localhost:8080", "gRPC gateway endpoint")
	stateGRPCCmd.PersistentFlags().StringVar(&serviceFlag, "service", "org.plantd.Client", "Service scope for state operations")
	stateGRPCCmd.PersistentFlags().StringVar(&profileFlag, "profile", "default", "Authentication profile to use")

	// Add gRPC command to main state command
	stateCmd.AddCommand(stateGRPCCmd)

	// Add flag to enable gRPC mode for existing commands
	stateCmd.PersistentFlags().BoolVar(&useGRPCFlag, "use-grpc", false, "Use gRPC instead of MDP for state operations")
}

// createGRPCStateClient creates a gRPC state client with authentication.
func createGRPCStateClient() (*grpc.StateClient, error) {
	gatewayEndpoint := grpcEndpointFlag
	if gatewayEndpoint == "" {
		gatewayEndpoint = "http://localhost:8080"
	}

	// Create client configuration
	config := &grpc.StateClientConfig{
		BaseURL: gatewayEndpoint,
		Timeout: 30 * time.Second,
		AuthFunc: func() string {
			tokenMgr := auth.NewTokenManager()
			token, _ := tokenMgr.GetValidToken(profileFlag)
			return token
		},
	}

	return grpc.NewStateClient(config), nil
}

func getGRPC(_ *cobra.Command, args []string) {
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

func setGRPC(_ *cobra.Command, args []string) {
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

func listGRPC(_ *cobra.Command, _ []string) {
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

func deleteGRPC(_ *cobra.Command, args []string) {
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

func listScopesGRPC(_ *cobra.Command, _ []string) {
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
