package cmd

import (
	"log"

	"github.com/geoffjay/plantd/core"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of plant",
	Long:  `Plant CLI version information.`,
	Run: func(_ *cobra.Command, _ []string) {
		log.Println(core.VERSION)
	},
}
