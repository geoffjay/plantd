package cmd

import (
	"log"

	plantd "github.com/geoffjay/plantd/core/service"

	"github.com/spf13/cobra"
)

var (
	echoCmd = &cobra.Command{
		Use:   "echo",
		Short: "Echo a message",
		Long:  "Perform an echo service check",
		Args:  cobra.ExactArgs(1),
		Run:   echo,
	}
)

func echo(_ *cobra.Command, args []string) {
	client, err := plantd.NewClient(endpoint)
	if err != nil {
		log.Fatal(err)
	}

	message := args[0]
	request := &plantd.RawRequest{
		"service": "org.plantd.Client",
		"message": message,
	}
	response, err := client.SendRawRequest("org.plantd.module.Echo", "echo", request)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("%+v\n", response)
}
