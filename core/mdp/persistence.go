package mdp

import (
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// PersistenceStore defines the interface for request persistence
type PersistenceStore interface {
	StoreRequest(id string, request *Request) error
	RetrieveRequest(id string) (*Request, error)
	DeleteRequest(id string) error
	ListPendingRequests() ([]string, error)
	Close() error
}

// Request represents a persisted request with metadata
type Request struct {
	ID         string        `json:"id"`
	Client     string        `json:"client"`
	Service    string        `json:"service"`
	Data       []string      `json:"data"`
	Timestamp  time.Time     `json:"timestamp"`
	Retries    int           `json:"retries"`
	MaxRetries int           `json:"max_retries"`
	TTL        time.Duration `json:"ttl"`
	Status     string        `json:"status"` // pending, processing, completed, failed
}

// MemoryPersistenceStore implements in-memory persistence for development and testing
type MemoryPersistenceStore struct {
	mu       sync.RWMutex
	requests map[string]*Request
}

// NewMemoryPersistenceStore creates a new in-memory persistence store
func NewMemoryPersistenceStore() PersistenceStore {
	return &MemoryPersistenceStore{
		requests: make(map[string]*Request),
	}
}

// StoreRequest stores a request in memory
func (m *MemoryPersistenceStore) StoreRequest(id string, request *Request) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if request == nil {
		return fmt.Errorf("request cannot be nil")
	}

	// Set default values if not provided
	if request.ID == "" {
		request.ID = id
	}
	if request.Timestamp.IsZero() {
		request.Timestamp = time.Now()
	}
	if request.Status == "" {
		request.Status = "pending"
	}
	if request.MaxRetries == 0 {
		request.MaxRetries = 3
	}
	if request.TTL == 0 {
		request.TTL = 5 * time.Minute
	}

	// Check if request has expired
	if time.Since(request.Timestamp) > request.TTL {
		return fmt.Errorf("request %s has expired", id)
	}

	m.requests[id] = request

	log.WithFields(log.Fields{
		"request_id": id,
		"client":     request.Client,
		"service":    request.Service,
		"status":     request.Status,
	}).Debug("stored request in memory")

	return nil
}

// RetrieveRequest retrieves a request from memory
func (m *MemoryPersistenceStore) RetrieveRequest(id string) (*Request, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	request, exists := m.requests[id]
	if !exists {
		return nil, fmt.Errorf("request %s not found", id)
	}

	// Check if request has expired
	if time.Since(request.Timestamp) > request.TTL {
		return nil, fmt.Errorf("request %s has expired", id)
	}

	return request, nil
}

// DeleteRequest removes a request from memory
func (m *MemoryPersistenceStore) DeleteRequest(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.requests[id]; !exists {
		return fmt.Errorf("request %s not found", id)
	}

	delete(m.requests, id)

	log.WithFields(log.Fields{
		"request_id": id,
	}).Debug("deleted request from memory")

	return nil
}

// ListPendingRequests returns all pending request IDs
func (m *MemoryPersistenceStore) ListPendingRequests() ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var pendingIDs []string
	now := time.Now()

	for id, request := range m.requests {
		// Skip expired requests
		if now.Sub(request.Timestamp) > request.TTL {
			continue
		}

		if request.Status == "pending" || request.Status == "processing" {
			pendingIDs = append(pendingIDs, id)
		}
	}

	return pendingIDs, nil
}

// Close cleans up the memory store
func (m *MemoryPersistenceStore) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.requests = make(map[string]*Request)
	return nil
}

// CleanupExpiredRequests removes expired requests from memory
func (m *MemoryPersistenceStore) CleanupExpiredRequests() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	removed := 0

	for id, request := range m.requests {
		if now.Sub(request.Timestamp) > request.TTL {
			delete(m.requests, id)
			removed++

			log.WithFields(log.Fields{
				"request_id": id,
				"age":        now.Sub(request.Timestamp),
				"ttl":        request.TTL,
			}).Debug("cleaned up expired request")
		}
	}

	if removed > 0 {
		log.WithFields(log.Fields{
			"removed_count": removed,
		}).Info("cleaned up expired requests")
	}

	return removed
}

// GetStats returns statistics about the persistence store
func (m *MemoryPersistenceStore) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]interface{})
	statusCounts := make(map[string]int)

	for _, request := range m.requests {
		statusCounts[request.Status]++
	}

	stats["total_requests"] = len(m.requests)
	stats["status_breakdown"] = statusCounts
	stats["store_type"] = "memory"

	return stats
}

// RequestManager handles request lifecycle and retry logic
type RequestManager struct {
	store PersistenceStore
	mu    sync.RWMutex
}

// NewRequestManager creates a new request manager
func NewRequestManager(store PersistenceStore) *RequestManager {
	return &RequestManager{
		store: store,
	}
}

// CreateRequest creates and stores a new request
func (rm *RequestManager) CreateRequest(client, service string, data []string) (*Request, error) {
	id := generateRequestID()

	request := &Request{
		ID:         id,
		Client:     client,
		Service:    service,
		Data:       data,
		Timestamp:  time.Now(),
		Retries:    0,
		MaxRetries: 3,
		TTL:        5 * time.Minute,
		Status:     "pending",
	}

	err := rm.store.StoreRequest(id, request)
	if err != nil {
		return nil, fmt.Errorf("failed to store request: %w", err)
	}

	log.WithFields(log.Fields{
		"request_id": id,
		"client":     client,
		"service":    service,
	}).Info("created new request")

	return request, nil
}

// MarkRequestProcessing marks a request as being processed
func (rm *RequestManager) MarkRequestProcessing(id string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	request, err := rm.store.RetrieveRequest(id)
	if err != nil {
		return fmt.Errorf("failed to retrieve request: %w", err)
	}

	request.Status = "processing"

	err = rm.store.StoreRequest(id, request)
	if err != nil {
		return fmt.Errorf("failed to update request status: %w", err)
	}

	return nil
}

// MarkRequestCompleted marks a request as completed and removes it from storage
func (rm *RequestManager) MarkRequestCompleted(id string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	request, err := rm.store.RetrieveRequest(id)
	if err != nil {
		return fmt.Errorf("failed to retrieve request: %w", err)
	}

	request.Status = "completed"

	// Remove completed requests from storage
	err = rm.store.DeleteRequest(id)
	if err != nil {
		log.WithFields(log.Fields{
			"request_id": id,
			"error":      err,
		}).Warn("failed to delete completed request")
	}

	log.WithFields(log.Fields{
		"request_id": id,
	}).Info("request completed successfully")

	return nil
}

// RetryRequest increments retry count and updates request status
func (rm *RequestManager) RetryRequest(id string) (*Request, error) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	request, err := rm.store.RetrieveRequest(id)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve request: %w", err)
	}

	request.Retries++

	if request.Retries >= request.MaxRetries {
		request.Status = "failed"

		log.WithFields(log.Fields{
			"request_id":  id,
			"retries":     request.Retries,
			"max_retries": request.MaxRetries,
		}).Error("request failed after maximum retries")

		// Keep failed requests for analysis
		err = rm.store.StoreRequest(id, request)
		return request, err
	}

	request.Status = "pending"

	err = rm.store.StoreRequest(id, request)
	if err != nil {
		return nil, fmt.Errorf("failed to update request: %w", err)
	}

	log.WithFields(log.Fields{
		"request_id": id,
		"retries":    request.Retries,
	}).Info("request scheduled for retry")

	return request, nil
}

// GetPendingRequests returns all pending requests for retry
func (rm *RequestManager) GetPendingRequests() ([]*Request, error) {
	pendingIDs, err := rm.store.ListPendingRequests()
	if err != nil {
		return nil, fmt.Errorf("failed to list pending requests: %w", err)
	}

	var requests []*Request
	for _, id := range pendingIDs {
		request, err := rm.store.RetrieveRequest(id)
		if err != nil {
			log.WithFields(log.Fields{
				"request_id": id,
				"error":      err,
			}).Warn("failed to retrieve pending request")
			continue
		}
		requests = append(requests, request)
	}

	return requests, nil
}

// Close closes the request manager and underlying store
func (rm *RequestManager) Close() error {
	return rm.store.Close()
}

// generateRequestID generates a unique request ID
func generateRequestID() string {
	return fmt.Sprintf("req_%d_%d", time.Now().UnixNano(), randomInt(1000, 9999))
}

// randomInt generates a random integer between min and max
func randomInt(min, max int) int {
	return min + int(time.Now().UnixNano()%int64(max-min))
}
