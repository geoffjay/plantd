package cmd

import (
	"log"

	plantd "github.com/geoffjay/plantd/core/service"

	"github.com/spf13/cobra"
)

var (
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
)

func init() {
	stateCmd.AddCommand(stateGetCmd)
	stateCmd.AddCommand(stateSetCmd)
}

func get(_ *cobra.Command, args []string) {
	log.Println(endpoint)
	client, err := plantd.NewClient(endpoint)
	if err != nil {
		log.Fatal(err)
	}

	key := args[0]
	request := &plantd.RawRequest{
		"service": "org.plantd.Client",
		"key":     key,
	}
	response, err := client.SendRawRequest("org.plantd.State", "state-get", request)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("%+v\n", response)
}

func set(_ *cobra.Command, args []string) {
	log.Println(endpoint)
	client, err := plantd.NewClient(endpoint)
	if err != nil {
		log.Fatal(err)
	}

	key := args[0]
	value := args[1]
	request := &plantd.RawRequest{
		"service": "org.plantd.Client",
		"key":     key,
		"value":   value,
	}
	response, err := client.SendRawRequest("org.plantd.State", "state-set", request)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("%+v\n", response)
}
