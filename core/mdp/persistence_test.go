package mdp

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryPersistenceStore(t *testing.T) {
	store := NewMemoryPersistenceStore()
	defer store.Close()

	t.Run("StoreAndRetrieveRequest", func(t *testing.T) {
		request := &Request{
			ID:         "test-001",
			Client:     "client-001",
			Service:    "echo",
			Data:       []string{"hello", "world"},
			Timestamp:  time.Now(),
			Retries:    0,
			MaxRetries: 3,
			TTL:        5 * time.Minute,
			Status:     "pending",
		}

		err := store.StoreRequest("test-001", request)
		assert.NoError(t, err)

		retrieved, err := store.RetrieveRequest("test-001")
		assert.NoError(t, err)
		assert.Equal(t, request.ID, retrieved.ID)
		assert.Equal(t, request.Client, retrieved.Client)
		assert.Equal(t, request.Service, retrieved.Service)
		assert.Equal(t, request.Data, retrieved.Data)
		assert.Equal(t, request.Status, retrieved.Status)
	})

	t.Run("RetrieveNonexistentRequest", func(t *testing.T) {
		_, err := store.RetrieveRequest("nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("DeleteRequest", func(t *testing.T) {
		request := &Request{
			ID:      "test-002",
			Client:  "client-002",
			Service: "echo",
			Data:    []string{"test"},
			Status:  "pending",
		}

		err := store.StoreRequest("test-002", request)
		require.NoError(t, err)

		err = store.DeleteRequest("test-002")
		assert.NoError(t, err)

		_, err = store.RetrieveRequest("test-002")
		assert.Error(t, err)
	})

	t.Run("ListPendingRequests", func(t *testing.T) {
		// Clear store first
		store.Close()
		store = NewMemoryPersistenceStore()

		requests := []*Request{
			{ID: "pending-001", Status: "pending"},
			{ID: "processing-001", Status: "processing"},
			{ID: "completed-001", Status: "completed"},
			{ID: "failed-001", Status: "failed"},
		}

		for _, req := range requests {
			err := store.StoreRequest(req.ID, req)
			require.NoError(t, err)
		}

		pendingIDs, err := store.ListPendingRequests()
		assert.NoError(t, err)
		assert.Len(t, pendingIDs, 2) // pending and processing
		assert.Contains(t, pendingIDs, "pending-001")
		assert.Contains(t, pendingIDs, "processing-001")
	})

	t.Run("ExpiredRequests", func(t *testing.T) {
		store.Close()
		store = NewMemoryPersistenceStore()

		// Create an expired request
		expiredRequest := &Request{
			ID:        "expired-001",
			Client:    "client-expired",
			Service:   "echo",
			Data:      []string{"expired"},
			Timestamp: time.Now().Add(-10 * time.Minute),
			TTL:       5 * time.Minute,
			Status:    "pending",
		}

		err := store.StoreRequest("expired-001", expiredRequest)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expired")

		// Test retrieval of expired request
		validRequest := &Request{
			ID:        "expired-002",
			Client:    "client-valid",
			Service:   "echo",
			Data:      []string{"valid"},
			Timestamp: time.Now().Add(-10 * time.Minute),
			TTL:       5 * time.Minute,
			Status:    "pending",
		}

		// Force store the expired request by bypassing the TTL check
		memStore := store.(*MemoryPersistenceStore)
		memStore.mu.Lock()
		memStore.requests["expired-002"] = validRequest
		memStore.mu.Unlock()

		// Now try to retrieve it - should fail due to expiration
		_, err = store.RetrieveRequest("expired-002")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expired")
	})

	t.Run("CleanupExpiredRequests", func(t *testing.T) {
		store.Close()
		memStore := NewMemoryPersistenceStore().(*MemoryPersistenceStore)

		// Add some requests with different timestamps
		memStore.mu.Lock()
		memStore.requests["current-001"] = &Request{
			ID:        "current-001",
			Timestamp: time.Now(),
			TTL:       5 * time.Minute,
			Status:    "pending",
		}
		memStore.requests["expired-001"] = &Request{
			ID:        "expired-001",
			Timestamp: time.Now().Add(-10 * time.Minute),
			TTL:       5 * time.Minute,
			Status:    "pending",
		}
		memStore.mu.Unlock()

		removed := memStore.CleanupExpiredRequests()
		assert.Equal(t, 1, removed)

		// Verify current request still exists
		_, err := memStore.RetrieveRequest("current-001")
		assert.NoError(t, err)

		// Verify expired request was removed
		_, err = memStore.RetrieveRequest("expired-001")
		assert.Error(t, err)
	})

	t.Run("GetStats", func(t *testing.T) {
		store.Close()
		memStore := NewMemoryPersistenceStore().(*MemoryPersistenceStore)

		requests := []*Request{
			{ID: "pending-001", Status: "pending"},
			{ID: "pending-002", Status: "pending"},
			{ID: "processing-001", Status: "processing"},
			{ID: "completed-001", Status: "completed"},
		}

		for _, req := range requests {
			err := memStore.StoreRequest(req.ID, req)
			require.NoError(t, err)
		}

		stats := memStore.GetStats()
		assert.Equal(t, 4, stats["total_requests"])
		assert.Equal(t, "memory", stats["store_type"])

		statusBreakdown := stats["status_breakdown"].(map[string]int)
		assert.Equal(t, 2, statusBreakdown["pending"])
		assert.Equal(t, 1, statusBreakdown["processing"])
		assert.Equal(t, 1, statusBreakdown["completed"])
	})
}

func TestRequestManager(t *testing.T) {
	store := NewMemoryPersistenceStore()
	manager := NewRequestManager(store)
	defer manager.Close()

	t.Run("CreateRequest", func(t *testing.T) {
		request, err := manager.CreateRequest("client-001", "echo", []string{"hello", "world"})
		assert.NoError(t, err)
		assert.NotEmpty(t, request.ID)
		assert.Equal(t, "client-001", request.Client)
		assert.Equal(t, "echo", request.Service)
		assert.Equal(t, []string{"hello", "world"}, request.Data)
		assert.Equal(t, "pending", request.Status)
		assert.Equal(t, 0, request.Retries)
		assert.Equal(t, 3, request.MaxRetries)
	})

	t.Run("MarkRequestProcessing", func(t *testing.T) {
		request, err := manager.CreateRequest("client-002", "echo", []string{"test"})
		require.NoError(t, err)

		err = manager.MarkRequestProcessing(request.ID)
		assert.NoError(t, err)

		retrieved, err := store.RetrieveRequest(request.ID)
		assert.NoError(t, err)
		assert.Equal(t, "processing", retrieved.Status)
	})

	t.Run("MarkRequestCompleted", func(t *testing.T) {
		request, err := manager.CreateRequest("client-003", "echo", []string{"test"})
		require.NoError(t, err)

		err = manager.MarkRequestCompleted(request.ID)
		assert.NoError(t, err)

		// Request should be deleted after completion
		_, err = store.RetrieveRequest(request.ID)
		assert.Error(t, err)
	})

	t.Run("RetryRequest", func(t *testing.T) {
		request, err := manager.CreateRequest("client-004", "echo", []string{"test"})
		require.NoError(t, err)

		// First retry
		retried, err := manager.RetryRequest(request.ID)
		assert.NoError(t, err)
		assert.Equal(t, 1, retried.Retries)
		assert.Equal(t, "pending", retried.Status)

		// Second retry
		retried, err = manager.RetryRequest(request.ID)
		assert.NoError(t, err)
		assert.Equal(t, 2, retried.Retries)
		assert.Equal(t, "pending", retried.Status)

		// Third retry - should fail
		retried, err = manager.RetryRequest(request.ID)
		assert.NoError(t, err)
		assert.Equal(t, 3, retried.Retries)
		assert.Equal(t, "failed", retried.Status)
	})

	t.Run("GetPendingRequests", func(t *testing.T) {
		// Clear previous requests
		store.Close()
		store = NewMemoryPersistenceStore()
		manager = NewRequestManager(store)

		// Create multiple requests
		req1, err := manager.CreateRequest("client-001", "service1", []string{"data1"})
		require.NoError(t, err)

		req2, err := manager.CreateRequest("client-002", "service2", []string{"data2"})
		require.NoError(t, err)

		// Mark one as processing
		err = manager.MarkRequestProcessing(req2.ID)
		require.NoError(t, err)

		pending, err := manager.GetPendingRequests()
		assert.NoError(t, err)
		assert.Len(t, pending, 2) // Both pending and processing are included

		// Verify the requests
		var foundPending, foundProcessing bool
		for _, req := range pending {
			if req.ID == req1.ID && req.Status == "pending" {
				foundPending = true
			}
			if req.ID == req2.ID && req.Status == "processing" {
				foundProcessing = true
			}
		}
		assert.True(t, foundPending)
		assert.True(t, foundProcessing)
	})

	t.Run("RequestManagerClose", func(t *testing.T) {
		tempStore := NewMemoryPersistenceStore()
		tempManager := NewRequestManager(tempStore)

		_, err := tempManager.CreateRequest("client-001", "service", []string{"data"})
		require.NoError(t, err)

		err = tempManager.Close()
		assert.NoError(t, err)
	})
}

func TestRequestLifecycle(t *testing.T) {
	store := NewMemoryPersistenceStore()
	manager := NewRequestManager(store)
	defer manager.Close()

	// Create request
	request, err := manager.CreateRequest("client-001", "calculator", []string{"add", "2", "3"})
	require.NoError(t, err)
	assert.Equal(t, "pending", request.Status)

	// Mark as processing
	err = manager.MarkRequestProcessing(request.ID)
	require.NoError(t, err)

	retrieved, err := store.RetrieveRequest(request.ID)
	require.NoError(t, err)
	assert.Equal(t, "processing", retrieved.Status)

	// Complete the request
	err = manager.MarkRequestCompleted(request.ID)
	require.NoError(t, err)

	// Request should be removed
	_, err = store.RetrieveRequest(request.ID)
	assert.Error(t, err)
}

func TestRequestRetryScenario(t *testing.T) {
	store := NewMemoryPersistenceStore()
	manager := NewRequestManager(store)
	defer manager.Close()

	// Create request
	request, err := manager.CreateRequest("client-retry", "flaky-service", []string{"operation"})
	require.NoError(t, err)

	// Simulate multiple failures and retries
	for i := 0; i < 3; i++ {
		// Mark as processing
		err = manager.MarkRequestProcessing(request.ID)
		require.NoError(t, err)

		// Simulate failure and retry
		retried, err := manager.RetryRequest(request.ID)
		require.NoError(t, err)

		if i < 2 {
			assert.Equal(t, "pending", retried.Status)
			assert.Equal(t, i+1, retried.Retries)
		} else {
			// Final retry should mark as failed
			assert.Equal(t, "failed", retried.Status)
			assert.Equal(t, 3, retried.Retries)
		}
	}
}

func TestConcurrentAccess(t *testing.T) {
	store := NewMemoryPersistenceStore()
	manager := NewRequestManager(store)
	defer manager.Close()

	// Test concurrent creation and retrieval
	const numGoroutines = 10
	const requestsPerGoroutine = 10

	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(workerID int) {
			defer func() { done <- true }()

			for j := 0; j < requestsPerGoroutine; j++ {
				clientID := fmt.Sprintf("client-%d-%d", workerID, j)
				request, err := manager.CreateRequest(clientID, "concurrent-service", []string{"data"})
				assert.NoError(t, err)
				assert.NotEmpty(t, request.ID)

				// Try to retrieve the request immediately
				retrieved, err := store.RetrieveRequest(request.ID)
				assert.NoError(t, err)
				assert.Equal(t, clientID, retrieved.Client)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify total number of requests
	pending, err := manager.GetPendingRequests()
	assert.NoError(t, err)
	assert.Len(t, pending, numGoroutines*requestsPerGoroutine)
}

func BenchmarkMemoryPersistenceStore(b *testing.B) {
	store := NewMemoryPersistenceStore()
	defer store.Close()

	requests := make([]*Request, b.N)
	for i := 0; i < b.N; i++ {
		requests[i] = &Request{
			ID:      fmt.Sprintf("bench-%d", i),
			Client:  fmt.Sprintf("client-%d", i),
			Service: "benchmark-service",
			Data:    []string{"benchmark", "data"},
			Status:  "pending",
		}
	}

	b.ResetTimer()

	b.Run("Store", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			err := store.StoreRequest(requests[i].ID, requests[i])
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Retrieve", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := store.RetrieveRequest(requests[i].ID)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkRequestManager(b *testing.B) {
	store := NewMemoryPersistenceStore()
	manager := NewRequestManager(store)
	defer manager.Close()

	b.ResetTimer()

	b.Run("CreateRequest", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := manager.CreateRequest(
				fmt.Sprintf("client-%d", i),
				"benchmark-service",
				[]string{"benchmark", "data"},
			)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
