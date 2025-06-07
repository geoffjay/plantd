package cmd

import (
	"log"

	cfg "github.com/geoffjay/plantd/core/config"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type server struct {
	Endpoint string `mapstructure:"endpoint"`
}

type clientConfig struct {
	Server server `mapstructure:"server"`
}

var (
	cfgFile  string
	config   clientConfig
	endpoint string
	Verbose  bool

	cliCmd = &cobra.Command{
		Use:   "plant",
		Short: "Application to control plantd services",
		Long:  `A control utility for interacting with plantd services.`,
	}
)

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
