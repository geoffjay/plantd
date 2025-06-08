package mdp

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	testServiceName = "test.service"
	testBroker      = "inproc://test-broker"
)

func TestNewClient(t *testing.T) {
	t.Run("successful creation", func(t *testing.T) {
		client, err := NewClient(testBroker)

		assert.NoError(t, err)
		assert.NotNil(t, client)

		if client != nil {
			assert.Equal(t, testBroker, client.broker)
			assert.Equal(t, time.Duration(2500), client.timeout)

			err := client.Close()
			assert.NoError(t, err)
		}
	})
}

func TestClientClose(t *testing.T) {
	client, err := NewClient(testBroker)
	if !assert.NoError(t, err) || !assert.NotNil(t, client) {
		t.Fatal("Failed to create client")
	}

	t.Run("close client", func(t *testing.T) {
		err := client.Close()
		assert.NoError(t, err)
	})

	t.Run("double close", func(t *testing.T) {
		// Second close should also be safe
		err := client.Close()
		assert.NoError(t, err)
	})
}

func TestClientSetTimeout(t *testing.T) {
	client, err := NewClient(testBroker)
	if !assert.NoError(t, err) || !assert.NotNil(t, client) {
		t.Fatal("Failed to create client")
	}
	defer func() {
		if client != nil {
			err := client.Close()
			assert.NoError(t, err)
		}
	}()

	t.Run("set timeout", func(t *testing.T) {
		newTimeout := time.Duration(5000)
		client.SetTimeout(newTimeout)
		assert.Equal(t, newTimeout, client.timeout)
	})
}

func TestClientSend(t *testing.T) {
	client, err := NewClient(testBroker)
	if !assert.NoError(t, err) || !assert.NotNil(t, client) {
		t.Fatal("Failed to create client")
	}
	defer func() {
		if client != nil {
			err := client.Close()
			assert.NoError(t, err)
		}
	}()

	t.Run("send message", func(t *testing.T) {
		// This will fail because we're not connected to a real broker
		// but we can test that the method exists and handles the call
		err := client.Send(testServiceName, "test message")
		// We expect an error since no broker is running
		assert.Error(t, err)
	})

	t.Run("send empty service", func(t *testing.T) {
		err := client.Send("", "test message")
		// Should still try to send but fail due to no broker
		assert.Error(t, err)
	})
}

func TestClientRecv(t *testing.T) {
	client, err := NewClient(testBroker)
	if !assert.NoError(t, err) || !assert.NotNil(t, client) {
		t.Fatal("Failed to create client")
	}
	defer func() {
		if client != nil {
			err := client.Close()
			assert.NoError(t, err)
		}
	}()

	t.Run("receive timeout", func(t *testing.T) {
		// Set short timeout for this test
		client.SetTimeout(time.Duration(100))

		msg, err := client.Recv()
		// Should timeout since no broker is running
		assert.Error(t, err)
		assert.Nil(t, msg)
	})
}
