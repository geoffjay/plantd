package main

import (
	"context"
	"sync"
	"testing"
	"time"

	"fmt"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// MockTransport is a mock implementation of Transport for testing
type MockTransport struct {
	mock.Mock
	name string
}

func (m *MockTransport) Start(ctx context.Context, wg *sync.WaitGroup) error {
	args := m.Called(ctx, wg)
	return args.Error(0)
}

func (m *MockTransport) Stop() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockTransport) Name() string {
	return m.name
}

type TransportManagerTestSuite struct {
	suite.Suite
	ctx    context.Context
	cancel context.CancelFunc
	wg     *sync.WaitGroup
}

func TestTransportManagerTestSuite(t *testing.T) {
	suite.Run(t, new(TransportManagerTestSuite))
}

func (suite *TransportManagerTestSuite) SetupTest() {
	suite.ctx, suite.cancel = context.WithCancel(context.Background())
	suite.wg = &sync.WaitGroup{}
}

func (suite *TransportManagerTestSuite) TearDownTest() {
	suite.cancel()
	// Wait briefly for goroutines to clean up
	done := make(chan struct{})
	go func() {
		suite.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Good, all goroutines finished
	case <-time.After(2 * time.Second):
		// Timeout waiting for cleanup
		suite.T().Log("Warning: Timeout waiting for transport cleanup")
	}
}

// Test TransportManager creation
func (suite *TransportManagerTestSuite) TestNewTransportManager() {
	tm := NewTransportManager()

	suite.NotNil(tm)
	suite.NotNil(tm.transports)
	suite.Len(tm.transports, 0)
}

// Test adding transports
func (suite *TransportManagerTestSuite) TestAddTransport() {
	tm := NewTransportManager()

	mockTransport := &MockTransport{name: "mock-transport"}

	tm.AddTransport(mockTransport)

	suite.Len(tm.transports, 1)
	suite.Contains(tm.transports, mockTransport)
}

// Test starting all transports
func (suite *TransportManagerTestSuite) TestStartAll_Success() {
	tm := NewTransportManager()

	// Create mock transports
	mock1 := &MockTransport{name: "transport1"}
	mock2 := &MockTransport{name: "transport2"}

	// Set up expectations
	mock1.On("Start", mock.Anything, mock.Anything).Return(nil)
	mock2.On("Start", mock.Anything, mock.Anything).Return(nil)

	tm.AddTransport(mock1)
	tm.AddTransport(mock2)

	err := tm.StartAll(suite.ctx, suite.wg)
	suite.NoError(err)

	// Verify mocks were called
	mock1.AssertExpectations(suite.T())
	mock2.AssertExpectations(suite.T())
}

// Test starting transports with one failure
func (suite *TransportManagerTestSuite) TestStartAll_PartialFailure() {
	tm := NewTransportManager()

	// Create mock transports
	mock1 := &MockTransport{name: "transport1"}
	mock2 := &MockTransport{name: "transport2"}

	// Set up expectations - second transport fails
	mock1.On("Start", mock.Anything, mock.Anything).Return(nil)
	mock2.On("Start", mock.Anything, mock.Anything).Return(assert.AnError)

	tm.AddTransport(mock1)
	tm.AddTransport(mock2)

	err := tm.StartAll(suite.ctx, suite.wg)
	suite.Error(err)
	suite.Contains(err.Error(), "failed to start transport")

	// Verify mocks were called
	mock1.AssertExpectations(suite.T())
	mock2.AssertExpectations(suite.T())
}

// Test stopping all transports
func (suite *TransportManagerTestSuite) TestStopAll_Success() {
	tm := NewTransportManager()

	// Create mock transports
	mock1 := &MockTransport{name: "transport1"}
	mock2 := &MockTransport{name: "transport2"}

	// Set up expectations
	mock1.On("Stop").Return(nil)
	mock2.On("Stop").Return(nil)

	tm.AddTransport(mock1)
	tm.AddTransport(mock2)

	err := tm.StopAll()
	suite.NoError(err)

	// Verify mocks were called
	mock1.AssertExpectations(suite.T())
	mock2.AssertExpectations(suite.T())
}

// Test stopping transports with one failure
func (suite *TransportManagerTestSuite) TestStopAll_PartialFailure() {
	tm := NewTransportManager()

	// Create mock transports
	mock1 := &MockTransport{name: "transport1"}
	mock2 := &MockTransport{name: "transport2"}

	// Set up expectations - second transport fails to stop
	mock1.On("Stop").Return(nil)
	mock2.On("Stop").Return(assert.AnError)

	tm.AddTransport(mock1)
	tm.AddTransport(mock2)

	err := tm.StopAll()
	suite.Error(err)
	suite.Contains(err.Error(), "failed to stop transport")

	// Verify mocks were called
	mock1.AssertExpectations(suite.T())
	mock2.AssertExpectations(suite.T())
}

// Test transport creation functions
type TransportCreationTestSuite struct {
	suite.Suite
	handler *Handler
}

func TestTransportCreationTestSuite(t *testing.T) {
	suite.Run(t, new(TransportCreationTestSuite))
}

func (suite *TransportCreationTestSuite) SetupTest() {
	suite.handler = NewHandler()
}

// Test NewMDPTransport
func (suite *TransportCreationTestSuite) TestNewMDPTransport() {
	endpoint := "tcp://localhost:7200"
	service := "org.plantd.State"

	transport := NewMDPTransport(endpoint, service, suite.handler)

	suite.NotNil(transport)
	suite.Equal(endpoint, transport.endpoint)
	suite.Equal(service, transport.service)
	suite.Equal(suite.handler, transport.handler)
	suite.Equal("MDP", transport.Name())
}

// Test NewGRPCTransport
func (suite *TransportCreationTestSuite) TestNewGRPCTransport() {
	port := "8080"
	store := NewStore()

	transport := NewGRPCTransport(port, store)

	suite.NotNil(transport)
	suite.Equal(port, transport.port)
	suite.Equal(store, transport.store)
	suite.Equal("gRPC", transport.Name())
}

// Test MDPTransport Name method
func (suite *TransportCreationTestSuite) TestMDPTransport_Name() {
	transport := &MDPTransport{}
	suite.Equal("MDP", transport.Name())
}

// Test GRPCTransport Name method
func (suite *TransportCreationTestSuite) TestGRPCTransport_Name() {
	transport := &GRPCTransport{}
	suite.Equal("gRPC", transport.Name())
}

// Integration tests with real transports (but mocked dependencies)
type TransportIntegrationTestSuite struct {
	suite.Suite
	ctx    context.Context
	cancel context.CancelFunc
	wg     *sync.WaitGroup
	store  *Store
}

func TestTransportIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(TransportIntegrationTestSuite))
}

func (suite *TransportIntegrationTestSuite) SetupTest() {
	suite.ctx, suite.cancel = context.WithCancel(context.Background())
	suite.wg = &sync.WaitGroup{}
	suite.store = NewStore()

	// Initialize store with in-memory database
	err := suite.store.Load(":memory:")
	suite.NoError(err)
}

func (suite *TransportIntegrationTestSuite) TearDownTest() {
	if suite.store != nil {
		suite.store.Unload()
	}

	suite.cancel()

	// Wait for cleanup with timeout
	done := make(chan struct{})
	go func() {
		suite.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Good, all goroutines finished
	case <-time.After(2 * time.Second):
		// Timeout waiting for cleanup
		suite.T().Log("Warning: Timeout waiting for integration test cleanup")
	}
}

// Test GRPCTransport with real implementation (but no actual network)
func (suite *TransportIntegrationTestSuite) TestGRPCTransport_StartStop() {
	// Use an unusual port to avoid conflicts
	transport := NewGRPCTransport("9999", suite.store)

	// Start should succeed (but may fail if port is in use)
	err := transport.Start(suite.ctx, suite.wg)
	if err != nil {
		// If we can't bind to the port, skip this test
		suite.T().Skipf("Cannot bind to port 9999: %v", err)
		return
	}

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	// Stop should succeed
	err = transport.Stop()
	suite.NoError(err)
}

// Test that concurrent transport operations work correctly
func (suite *TransportIntegrationTestSuite) TestConcurrentTransports() {
	tm := NewTransportManager()

	// Create multiple gRPC transports on different ports
	transport1 := NewGRPCTransport("19991", suite.store)
	transport2 := NewGRPCTransport("19992", suite.store)

	tm.AddTransport(transport1)
	tm.AddTransport(transport2)

	// Start all transports
	err := tm.StartAll(suite.ctx, suite.wg)
	if err != nil {
		// If we can't bind to ports, skip this test
		suite.T().Skipf("Cannot bind to test ports: %v", err)
		return
	}

	// Give the servers a moment to start
	time.Sleep(200 * time.Millisecond)

	// Stop all transports
	err = tm.StopAll()
	suite.NoError(err)
}

// Benchmark tests for transport operations
func BenchmarkTransportManager_StartStop(b *testing.B) {
	for i := 0; i < b.N; i++ {
		tm := NewTransportManager()

		// Add mock transports
		for j := 0; j < 5; j++ {
			mockTransport := &MockTransport{name: fmt.Sprintf("transport-%d", j)}
			mockTransport.On("Start", mock.Anything, mock.Anything).Return(nil)
			mockTransport.On("Stop").Return(nil)
			tm.AddTransport(mockTransport)
		}

		ctx, cancel := context.WithCancel(context.Background())
		wg := &sync.WaitGroup{}

		// Start all
		err := tm.StartAll(ctx, wg)
		if err != nil {
			b.Fatal(err)
		}

		// Stop all
		err = tm.StopAll()
		if err != nil {
			b.Fatal(err)
		}

		cancel()
	}
}

// Helper to verify mock import is working
func init() {
	// This ensures the mock package is properly imported
	_ = mock.Anything
}
