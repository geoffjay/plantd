//go:build integration
// +build integration

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
)

// nolint: funlen
func TestClient_Integration(t *testing.T) {
	ctx := context.Background()

	sharedNetwork, err := network.New(
		ctx,
		network.WithCheckDuplicate(),
		// network.WithAttachable(),
		network.WithInternal(),
	)
	if err != nil {
		t.Fatal(err)
	}

	networkName := sharedNetwork.Name

	// TODO: when broker has been migrated to this repo it should be used instead
	brokerContainer, err := testcontainers.GenericContainer(
		ctx,
		testcontainers.GenericContainerRequest{
			ContainerRequest: testcontainers.ContainerRequest{
				Image:        "registry.gitlab.com/plantd/broker:staging",
				Name:         "broker",
				ExposedPorts: []string{"9797/tcp"},
				WaitingFor:   wait.ForListeningPort("9797/tcp"),
				Networks:     []string{networkName},
				NetworkAliases: map[string][]string{
					networkName: {"broker"},
				},
				Env: map[string]string{
					"PLANTD_BROKER_ENDPOINT":  "tcp://*:9797",
					"PLANTD_BROKER_LOG_LEVEL": "debug",
				},
			},
			Started: true,
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	workerContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "org.plantd.module.echo:latest",
			Name:         "worker",
			ExposedPorts: []string{"5001/tcp"},
			// WaitingFor:   wait.ForHTTP("/api/v1/health"),
			WaitingFor: wait.ForListeningPort("5001/tcp"),
			Networks:   []string{networkName},
			NetworkAliases: map[string][]string{
				networkName: {"worker"},
			},
			Env: map[string]string{
				"PLANTD_BROKER_ENDPOINT": "tcp://broker:9797",
			},
		},
		Started: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if err := brokerContainer.Terminate(ctx); err != nil {
			t.Fatal(err)
		}

		if err := workerContainer.Terminate(ctx); err != nil {
			t.Fatal(err)
		}

		require.NoError(t, sharedNetwork.Remove(ctx))
	})

	// Test NewClient
	client, err := NewClient("tcp://127.0.0.1:9797")
	assert.Nil(t, err)
	assert.NotNil(t, client)

	// Test Client.SendRawRequest
	request := &RawRequest{
		"service": "org.plantd.Client",
		"message": "foo",
	}
	resp, err := client.SendRawRequest("org.plantd.module.Echo", "echo", request)

	assert.Nil(t, err)
	assert.Equal(t, "foo", resp["message"])
}
