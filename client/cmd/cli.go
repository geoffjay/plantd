// Package cmd provides command-line interface functionality for the client.
package cmd

import (
	"log"

	cfg "github.com/geoffjay/plantd/core/config"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type server struct {
	Endpoint string `mapstructure:"endpoint"`
	Timeout  string `mapstructure:"timeout"`
	Retries  int    `mapstructure:"retries"`
}

type identity struct {
	Endpoint       string `mapstructure:"endpoint"`
	DefaultProfile string `mapstructure:"default_profile"`
	AutoRefresh    bool   `mapstructure:"auto_refresh"`
	CacheDuration  string `mapstructure:"cache_duration"`
}

type defaults struct {
	Service      string `mapstructure:"service"`
	OutputFormat string `mapstructure:"output_format"`
}

type profile struct {
	IdentityEndpoint string `mapstructure:"identity_endpoint"`
}

type clientConfig struct {
	Server   server             `mapstructure:"server"`
	Identity identity           `mapstructure:"identity"`
	Defaults defaults           `mapstructure:"defaults"`
	Profiles map[string]profile `mapstructure:"profiles"`
}

var (
	cfgFile  string
	config   clientConfig
	endpoint string
	// Verbose enables verbose output when set to true.
	Verbose bool

	cliCmd = &cobra.Command{
		Use:   "plant",
		Short: "Application to control plantd services",
		Long:  `A control utility for interacting with plantd services.`,
	}
)

// Execute runs the root command.
func Execute() {
	if err := cliCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	addCommands()

	// Setup command flags
	cliCmd.PersistentFlags().StringVar(
		&cfgFile,
		"config", "",
		"config file (default is $HOME/.config/plantd/client.yaml)",
	)
	cliCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "verbose output")

	if err := viper.BindPFlag("verbose", cliCmd.PersistentFlags().Lookup("verbose")); err != nil {
		log.Fatal(err)
	}
	viper.SetDefault("verbose", false)
}

func addCommands() {
	cliCmd.AddCommand(authCmd)
	cliCmd.AddCommand(configCmd)
	cliCmd.AddCommand(echoCmd)
	// cliCmd.AddCommand(jobCmd)
	cliCmd.AddCommand(stateCmd)

	// Miscellaneous commands
	cliCmd.AddCommand(versionCmd)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if err := cfg.LoadConfig("client", &config); err != nil {
		log.Fatalf("error reading config file: %s\n", err)
	}

	endpoint = config.Server.Endpoint
}

// GetIdentityEndpoint returns the identity endpoint for the given profile
func GetIdentityEndpoint(profileName string) string {
	if profile, exists := config.Profiles[profileName]; exists {
		return profile.IdentityEndpoint
	}
	// Fallback to default identity endpoint
	return config.Identity.Endpoint
}

// GetDefaultService returns the default service scope
func GetDefaultService() string {
	if config.Defaults.Service != "" {
		return config.Defaults.Service
	}
	return "org.plantd.Client"
}

// GetDefaultProfile returns the default authentication profile
func GetDefaultProfile() string {
	if config.Identity.DefaultProfile != "" {
		return config.Identity.DefaultProfile
	}
	return "default"
}
