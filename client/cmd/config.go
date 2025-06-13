package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

var (
	configCmd = &cobra.Command{
		Use:   "config",
		Short: "Configuration management",
		Long:  "Manage plantd client configuration settings",
	}

	configInitCmd = &cobra.Command{
		Use:   "init",
		Short: "Initialize default configuration",
		Long:  "Create a default configuration file in the user's config directory",
		Run:   configInitHandler,
	}

	configShowCmd = &cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		Long:  "Display the current configuration values",
		Run:   configShowHandler,
	}

	configSetCmd = &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Long:  "Set a configuration value (e.g., 'plant config set identity.endpoint tcp://prod:9797')",
		Args:  cobra.ExactArgs(2),
		Run:   configSetHandler,
	}

	configValidateCmd = &cobra.Command{
		Use:   "validate",
		Short: "Validate configuration",
		Long:  "Validate the current configuration for common errors",
		Run:   configValidateHandler,
	}

	profilesCmd = &cobra.Command{
		Use:   "profiles",
		Short: "Profile management",
		Long:  "Manage authentication profiles",
	}

	profilesListCmd = &cobra.Command{
		Use:   "list",
		Short: "List available profiles",
		Long:  "List all available authentication profiles",
		Run:   profilesListHandler,
	}

	profilesCreateCmd = &cobra.Command{
		Use:   "create <name> <endpoint>",
		Short: "Create new profile",
		Long:  "Create a new authentication profile with the specified endpoint",
		Args:  cobra.ExactArgs(2),
		Run:   profilesCreateHandler,
	}

	profilesDeleteCmd = &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete profile",
		Long:  "Delete an existing authentication profile",
		Args:  cobra.ExactArgs(1),
		Run:   profilesDeleteHandler,
	}
)

func init() {
	// Add subcommands
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configValidateCmd)

	profilesCmd.AddCommand(profilesListCmd)
	profilesCmd.AddCommand(profilesCreateCmd)
	profilesCmd.AddCommand(profilesDeleteCmd)

	configCmd.AddCommand(profilesCmd)
}

func configInitHandler(_ *cobra.Command, _ []string) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get home directory: %v\n", err)
		os.Exit(1)
	}

	configDir := filepath.Join(homeDir, ".config", "plantd")
	configFile := filepath.Join(configDir, "client.yaml")

	// Check if config already exists
	if _, err := os.Stat(configFile); err == nil {
		log.Infof("Configuration file already exists at: %s\n", configFile)
		log.Info("Use 'plant config show' to view current configuration")
		return
	}

	// Create config directory
	if err := os.MkdirAll(configDir, 0755); err != nil {
		log.WithError(err).Fatal("Failed to create config directory")
		os.Exit(1)
	}

	// Create default configuration
	defaultConfig := map[string]interface{}{
		"server": map[string]interface{}{
			"endpoint": "tcp://127.0.0.1:9797",
			"timeout":  "30s",
			"retries":  3,
		},
		"identity": map[string]interface{}{
			"endpoint":        "tcp://127.0.0.1:9797",
			"default_profile": "default",
			"auto_refresh":    true,
			"cache_duration":  "5m",
		},
		"defaults": map[string]interface{}{
			"service":       "org.plantd.Client",
			"output_format": "json",
		},
		"profiles": map[string]interface{}{
			"default": map[string]interface{}{
				"identity_endpoint": "tcp://127.0.0.1:9797",
			},
		},
	}

	// Write configuration file
	data, err := yaml.Marshal(defaultConfig)
	if err != nil {
		log.WithError(err).Fatal("Failed to marshal configuration")
	}

	if err := os.WriteFile(configFile, data, 0644); err != nil {
		log.WithError(err).Fatal("Failed to write configuration file")
	}

	log.Infof("Configuration initialized at: %s\n", configFile)
}

func configShowHandler(_ *cobra.Command, _ []string) {
	log.Printf("Configuration file: %s\n", viper.ConfigFileUsed())
	log.Println("Current configuration:")
	log.Printf("Server endpoint: %s\n", config.Server.Endpoint)
	log.Printf("Server timeout: %s\n", config.Server.Timeout)
	log.Printf("Server retries: %d\n", config.Server.Retries)
	log.Printf("Identity endpoint: %s\n", config.Identity.Endpoint)
	log.Printf("Default profile: %s\n", config.Identity.DefaultProfile)
	log.Printf("Auto refresh: %t\n", config.Identity.AutoRefresh)
	log.Printf("Cache duration: %s\n", config.Identity.CacheDuration)
	log.Printf("Default service: %s\n", config.Defaults.Service)
	log.Printf("Output format: %s\n", config.Defaults.OutputFormat)

	log.Println("Profiles:")
	for name, profile := range config.Profiles {
		log.Printf("  %s: %s\n", name, profile.IdentityEndpoint)
	}
}

func configSetHandler(_ *cobra.Command, args []string) {
	key := args[0]
	value := args[1]

	// Set the value in viper
	viper.Set(key, value)

	// Write the configuration back to file
	if err := viper.WriteConfig(); err != nil {
		// If WriteConfig fails, try WriteConfigAs with the config file path
		configFile := viper.ConfigFileUsed()
		if configFile == "" {
			homeDir, _ := os.UserHomeDir()
			configFile = filepath.Join(homeDir, ".config", "plantd", "client.yaml")
		}
		if err := viper.WriteConfigAs(configFile); err != nil {
			log.WithError(err).Fatal("Failed to write configuration")
		}
	}

	log.Infof("Configuration updated: %s = %s\n", key, value)
}

func configValidateHandler(_ *cobra.Command, _ []string) {
	log.Info("Validating configuration...")

	errors := []string{}

	// Validate server configuration
	if config.Server.Endpoint == "" {
		errors = append(errors, "server.endpoint is required")
	}
	if !strings.HasPrefix(config.Server.Endpoint, "tcp://") {
		errors = append(errors, "server.endpoint must start with 'tcp://'")
	}

	// Validate identity configuration
	if config.Identity.Endpoint == "" {
		errors = append(errors, "identity.endpoint is required")
	}
	if !strings.HasPrefix(config.Identity.Endpoint, "tcp://") {
		errors = append(errors, "identity.endpoint must start with 'tcp://'")
	}

	// Validate profiles
	if len(config.Profiles) == 0 {
		errors = append(errors, "at least one profile is required")
	} else {
		// Check if default profile exists
		if _, exists := config.Profiles[config.Identity.DefaultProfile]; !exists {
			errors = append(errors, fmt.Sprintf("default profile '%s' does not exist", config.Identity.DefaultProfile))
		}

		// Validate each profile
		for name, profile := range config.Profiles {
			if profile.IdentityEndpoint == "" {
				errors = append(errors, fmt.Sprintf("profile '%s' is missing identity_endpoint", name))
			}
			if !strings.HasPrefix(profile.IdentityEndpoint, "tcp://") {
				errors = append(errors, fmt.Sprintf("profile '%s' identity_endpoint must start with 'tcp://'", name))
			}
		}
	}

	// Report validation results
	if len(errors) == 0 {
		log.Info("✓ Configuration is valid")
	} else {
		log.Info("✗ Configuration has errors:")
		for _, err := range errors {
			log.Infof("  - %s\n", err)
		}
		log.Fatal("Configuration is invalid")
	}
}

func profilesListHandler(_ *cobra.Command, _ []string) {
	log.Info("Available profiles:")
	if len(config.Profiles) == 0 {
		log.Info("  No profiles configured")
		return
	}

	for name, profile := range config.Profiles {
		indicator := ""
		if name == config.Identity.DefaultProfile {
			indicator = " (default)"
		}
		log.Infof("  %s: %s%s\n", name, profile.IdentityEndpoint, indicator)
	}
}

func profilesCreateHandler(_ *cobra.Command, args []string) {
	profileName := args[0]
	endpoint := args[1]

	// Validate endpoint format
	if !strings.HasPrefix(endpoint, "tcp://") {
		log.Fatal("Endpoint must start with 'tcp://'")
	}

	// Check if profile already exists
	if _, exists := config.Profiles[profileName]; exists {
		log.Fatalf("Profile '%s' already exists\n", profileName)
	}

	// Set the new profile
	profileKey := fmt.Sprintf("profiles.%s.identity_endpoint", profileName)
	viper.Set(profileKey, endpoint)

	// Write the configuration back to file
	if err := viper.WriteConfig(); err != nil {
		configFile := viper.ConfigFileUsed()
		if configFile == "" {
			homeDir, _ := os.UserHomeDir()
			configFile = filepath.Join(homeDir, ".config", "plantd", "client.yaml")
		}
		if err := viper.WriteConfigAs(configFile); err != nil {
			log.WithError(err).Fatal("Failed to write configuration")
		}
	}

	log.Infof("Profile '%s' created with endpoint: %s\n", profileName, endpoint)
}

func profilesDeleteHandler(_ *cobra.Command, args []string) {
	profileName := args[0]

	// Check if profile exists
	if _, exists := config.Profiles[profileName]; !exists {
		log.Fatalf("Profile '%s' does not exist\n", profileName)
	}

	// Don't allow deleting the default profile
	if profileName == config.Identity.DefaultProfile {
		log.Fatalf("Cannot delete default profile '%s'. Change the default profile first.\n", profileName)
	}

	// Remove the profile from viper
	profiles := viper.GetStringMap("profiles")
	delete(profiles, profileName)
	viper.Set("profiles", profiles)

	// Write the configuration back to file
	if err := viper.WriteConfig(); err != nil {
		configFile := viper.ConfigFileUsed()
		if configFile == "" {
			homeDir, _ := os.UserHomeDir()
			configFile = filepath.Join(homeDir, ".config", "plantd", "client.yaml")
		}
		if err := viper.WriteConfigAs(configFile); err != nil {
			log.WithError(err).Fatal("Failed to write configuration")
		}
	}

	log.Infof("Profile '%s' deleted\n", profileName)
}
