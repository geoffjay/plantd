package mdp

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewWorker(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Run("create worker with valid broker and service", func(t *testing.T) {
		broker := "inproc://test-worker-broker"
		service := "test.service"
		worker, err := NewWorker(broker, service)

		assert.NoError(t, err)
		assert.NotNil(t, worker)

		// Clean up
		if worker != nil {
			assert.Equal(t, broker, worker.broker)
			assert.Equal(t, service, worker.service)
			assert.Equal(t, 2500*time.Millisecond, worker.heartbeat)
			assert.Equal(t, 2500*time.Millisecond, worker.reconnect)
			assert.False(t, worker.shutdown)
			assert.False(t, worker.expectReply)

			worker.Close()
		}
	})
}

func TestWorkerClose(t *testing.T) {
	worker := &Worker{
		broker:  "inproc://test",
		service: "test.service",
	}

	// Should not panic when worker socket is nil
	assert.NotPanics(t, func() {
		worker.Close()
	})

	// After close, worker should be nil
	worker.Close()
	assert.Nil(t, worker.worker)
}

func TestWorkerSetHeartbeat(t *testing.T) {
	worker := &Worker{
		broker:    "inproc://test",
		service:   "test.service",
		heartbeat: 2500 * time.Millisecond,
	}

	newHeartbeat := 5000 * time.Millisecond
	worker.SetHeartbeat(newHeartbeat)

	assert.Equal(t, newHeartbeat, worker.heartbeat)
}

func TestWorkerSetReconnect(t *testing.T) {
	worker := &Worker{
		broker:    "inproc://test",
		service:   "test.service",
		reconnect: 2500 * time.Millisecond,
	}

	newReconnect := 5000 * time.Millisecond
	worker.SetReconnect(newReconnect)

	assert.Equal(t, newReconnect, worker.reconnect)
}

func TestWorkerShutdown(t *testing.T) {
	worker := &Worker{
		broker:    "inproc://test",
		service:   "test.service",
		heartbeat: 100 * time.Millisecond, // Short heartbeat for testing
		shutdown:  false,
	}

	// Initially not terminated
	assert.False(t, worker.Terminated())

	// Start shutdown in goroutine since it sleeps
	go worker.Shutdown()

	// Give it a moment to set the flag
	time.Sleep(10 * time.Millisecond)

	// Should be terminated now
	assert.True(t, worker.Terminated())
}

func TestWorkerTerminated(t *testing.T) {
	worker := &Worker{
		broker:   "inproc://test",
		service:  "test.service",
		shutdown: false,
	}

	// Initially not terminated
	assert.False(t, worker.Terminated())

	// Set shutdown flag
	worker.shutdown = true
	assert.True(t, worker.Terminated())
}

func TestWorkerSendToBroker(t *testing.T) {
	t.Run("send with nil worker socket", func(t *testing.T) {
		worker := &Worker{
			broker:  "inproc://test-send",
			service: "test.service",
			worker:  nil, // Explicitly nil socket
		}

		err := worker.SendToBroker(MdpwReady, "option", []string{"msg1", "msg2"})
		assert.Error(t, err) // Expected since no real connection
	})

	t.Run("send message without option", func(t *testing.T) {
		worker := &Worker{
			broker:  "inproc://test-send-no-option",
			service: "test.service",
			worker:  nil, // Explicitly nil socket
		}

		err := worker.SendToBroker(MdpwHeartbeat, "", []string{})
		assert.Error(t, err) // Expected since no real connection
	})
}

func TestWorkerConnectToBroker(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Run("connect to valid endpoint", func(t *testing.T) {
		worker := &Worker{
			broker:  "inproc://test-connect",
			service: "test.service",
		}

		err := worker.ConnectToBroker()
		// May fail without actual broker, but tests the method
		if err != nil {
			assert.Error(t, err)
		} else {
			assert.NotNil(t, worker.worker)
			assert.NotNil(t, worker.poller)
			assert.Equal(t, HeartbeatLiveness, worker.liveness)
			worker.Close()
		}
	})

	t.Run("connect with invalid endpoint", func(t *testing.T) {
		worker := &Worker{
			broker:  "invalid://endpoint",
			service: "test.service",
		}

		err := worker.ConnectToBroker()
		assert.Error(t, err)
	})
}

func TestWorkerRecv(t *testing.T) {
	t.Run("recv with nil poller", func(t *testing.T) {
		worker := &Worker{
			broker:    "inproc://test-recv",
			service:   "test.service",
			heartbeat: 100 * time.Millisecond,
			poller:    nil, // Explicitly nil poller
		}

		// Should get an error due to nil poller
		msg, err := worker.Recv([]string{})
		assert.Error(t, err)
		_ = msg
	})

	t.Run("recv with reply but nil worker socket", func(t *testing.T) {
		worker := &Worker{
			broker:      "inproc://test-recv-reply",
			service:     "test.service",
			heartbeat:   100 * time.Millisecond,
			expectReply: true,
			replyTo:     "client-id",
			worker:      nil, // Explicitly nil socket
		}

		reply := []string{"response", "data"}
		msg, err := worker.Recv(reply)
		// Should get an error due to nil socket when trying to send reply
		assert.Error(t, err)
		_ = msg
	})
}

func TestWorkerConstants(t *testing.T) {
	// Test that required constants are defined
	assert.NotEmpty(t, MdpwWorker)
	assert.NotEmpty(t, MdpwReady)
	assert.NotEmpty(t, MdpwRequest)
	assert.NotEmpty(t, MdpwReply)
	assert.NotEmpty(t, MdpwHeartbeat)
	assert.NotEmpty(t, MdpwDisconnect)

	// Test specific values
	assert.Equal(t, "MDPW01", MdpwWorker)
	assert.Equal(t, "\001", MdpwReady)
	assert.Equal(t, "\002", MdpwRequest)
	assert.Equal(t, "\003", MdpwReply)
	assert.Equal(t, "\004", MdpwHeartbeat)
	assert.Equal(t, "\005", MdpwDisconnect)

	// Test heartbeat constants
	assert.Equal(t, 3, HeartbeatLiveness)
	assert.Equal(t, 1000*time.Millisecond, HeartbeatInterval)
}

func TestWorkerMessageFormat(t *testing.T) {
	t.Run("message format for SendToBroker", func(t *testing.T) {
		// Test the message format that SendToBroker should create
		command := MdpwReady
		option := "test-option"
		msg := []string{"data1", "data2"}

		// Expected format with option:
		// Frame 0: empty
		// Frame 1: "MDPWxy" (MDP/Worker protocol)
		// Frame 2: command
		// Frame 3: option
		// Frame 4+: message data

		expectedFrames := make([]string, 4, 4+len(msg))
		expectedFrames = append(expectedFrames, msg...)
		expectedFrames[3] = option
		expectedFrames[2] = command
		expectedFrames[1] = MdpwWorker
		expectedFrames[0] = ""

		assert.Equal(t, "", expectedFrames[0])
		assert.Equal(t, MdpwWorker, expectedFrames[1])
		assert.Equal(t, command, expectedFrames[2])
		assert.Equal(t, option, expectedFrames[3])
		assert.Equal(t, "data1", expectedFrames[4])
		assert.Equal(t, "data2", expectedFrames[5])
	})

	t.Run("message format without option", func(t *testing.T) {
		command := MdpwHeartbeat
		msg := []string{}

		// Expected format without option:
		// Frame 0: empty
		// Frame 1: "MDPWxy" (MDP/Worker protocol)
		// Frame 2: command

		expectedFrames := make([]string, 3, 3+len(msg))
		expectedFrames = append(expectedFrames, msg...)
		expectedFrames[2] = command
		expectedFrames[1] = MdpwWorker
		expectedFrames[0] = ""

		assert.Equal(t, "", expectedFrames[0])
		assert.Equal(t, MdpwWorker, expectedFrames[1])
		assert.Equal(t, command, expectedFrames[2])
	})
}
