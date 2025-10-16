package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/types/known/emptypb"

	healthv1 "github.com/geoffjay/plantd/gen/proto/go/plantd/health/v1"
	statev1 "github.com/geoffjay/plantd/gen/proto/go/plantd/state/v1"
)

type StateGRPCServerTestSuite struct {
	suite.Suite
	server *StateGRPCServer
	store  *Store
	ctx    context.Context
}

func TestStateGRPCServerTestSuite(t *testing.T) {
	suite.Run(t, new(StateGRPCServerTestSuite))
}

func (suite *StateGRPCServerTestSuite) SetupTest() {
	suite.ctx = context.Background()
	suite.store = NewStore()

	// Initialize store with in-memory database for testing
	err := suite.store.Load(":memory:")
	suite.NoError(err)

	suite.server = NewStateGRPCServer(suite.store)

	// Create test scopes and pre-populate some test data
	suite.store.CreateScope("test-scope")
	suite.store.CreateScope("default")
	suite.store.Set("test-scope", "test-key", "test-value")
	suite.store.Set("default", "default-key", "default-value")
}

func (suite *StateGRPCServerTestSuite) TearDownTest() {
	// Clean up the store
	if suite.store != nil {
		suite.store.Unload()
	}
}

// Test Get method
func (suite *StateGRPCServerTestSuite) TestGet_Success() {
	req := connect.NewRequest(&statev1.GetRequest{
		Key:   "test-key",
		Scope: "test-scope",
	})

	resp, err := suite.server.Get(suite.ctx, req)
	suite.NoError(err)
	suite.NotNil(resp)
	suite.Equal("test-value", resp.Msg.Value)
}

func (suite *StateGRPCServerTestSuite) TestGet_DefaultScope() {
	req := connect.NewRequest(&statev1.GetRequest{
		Key: "default-key",
		// No scope provided - should use default
	})

	resp, err := suite.server.Get(suite.ctx, req)
	suite.NoError(err)
	suite.NotNil(resp)
	suite.Equal("default-value", resp.Msg.Value)
}

func (suite *StateGRPCServerTestSuite) TestGet_NotFound() {
	req := connect.NewRequest(&statev1.GetRequest{
		Key:   "nonexistent-key",
		Scope: "test-scope",
	})

	resp, err := suite.server.Get(suite.ctx, req)
	suite.Error(err)
	suite.Nil(resp)

	connectErr := err.(*connect.Error)
	suite.Equal(connect.CodeNotFound, connectErr.Code())
	suite.Contains(connectErr.Message(), "key not found")
}

// Test Set method
func (suite *StateGRPCServerTestSuite) TestSet_Success() {
	req := connect.NewRequest(&statev1.SetRequest{
		Key:   "new-key",
		Value: "new-value",
		Scope: "test-scope",
	})

	resp, err := suite.server.Set(suite.ctx, req)
	suite.NoError(err)
	suite.NotNil(resp)
	suite.NotNil(resp.Msg.Metadata)

	// Verify the value was actually set
	value, err := suite.store.Get("test-scope", "new-key")
	suite.NoError(err)
	suite.Equal("new-value", value)
}

func (suite *StateGRPCServerTestSuite) TestSet_DefaultScope() {
	req := connect.NewRequest(&statev1.SetRequest{
		Key:   "new-default-key",
		Value: "new-default-value",
		// No scope provided - should use default
	})

	resp, err := suite.server.Set(suite.ctx, req)
	suite.NoError(err)
	suite.NotNil(resp)
	suite.NotNil(resp.Msg.Metadata)

	// Verify the value was set in default scope
	value, err := suite.store.Get("default", "new-default-key")
	suite.NoError(err)
	suite.Equal("new-default-value", value)
}

func (suite *StateGRPCServerTestSuite) TestSet_EmptyKey() {
	req := connect.NewRequest(&statev1.SetRequest{
		Key:   "",
		Value: "some-value",
		Scope: "test-scope",
	})

	resp, err := suite.server.Set(suite.ctx, req)
	suite.Error(err)
	suite.Nil(resp)

	connectErr := err.(*connect.Error)
	suite.Equal(connect.CodeInvalidArgument, connectErr.Code())
}

// Test Delete method
func (suite *StateGRPCServerTestSuite) TestDelete_Success() {
	req := connect.NewRequest(&statev1.DeleteRequest{
		Key:   "test-key",
		Scope: "test-scope",
	})

	resp, err := suite.server.Delete(suite.ctx, req)
	suite.NoError(err)
	suite.NotNil(resp)
	suite.True(resp.Msg.Existed)

	// Verify the key was actually deleted
	_, err = suite.store.Get("test-scope", "test-key")
	suite.Error(err)
}

func (suite *StateGRPCServerTestSuite) TestDelete_NotFound() {
	req := connect.NewRequest(&statev1.DeleteRequest{
		Key:   "nonexistent-key",
		Scope: "test-scope",
	})

	resp, err := suite.server.Delete(suite.ctx, req)
	suite.NoError(err)
	suite.NotNil(resp)
	suite.False(resp.Msg.Existed)
}

// Test BatchGet method
func (suite *StateGRPCServerTestSuite) TestBatchGet_Success() {
	// Add test data
	suite.store.Set("batch-scope", "batch-key1", "batch-value1")
	suite.store.Set("batch-scope", "batch-key2", "batch-value2")

	req := connect.NewRequest(&statev1.BatchGetRequest{
		Keys:  []string{"batch-key1", "batch-key2"},
		Scope: "batch-scope",
	})

	resp, err := suite.server.BatchGet(suite.ctx, req)
	suite.NoError(err)
	suite.NotNil(resp)
	suite.Len(resp.Msg.Results, 2)

	// Check results
	suite.Equal("batch-value1", resp.Msg.Results["batch-key1"].Value)
	suite.Equal("batch-value2", resp.Msg.Results["batch-key2"].Value)
}

func (suite *StateGRPCServerTestSuite) TestBatchGet_PartialSuccess() {
	// Add only one key
	suite.store.Set("batch-scope", "exists", "value")

	req := connect.NewRequest(&statev1.BatchGetRequest{
		Keys:  []string{"exists", "not-exists"},
		Scope: "batch-scope",
	})

	resp, err := suite.server.BatchGet(suite.ctx, req)
	suite.NoError(err)
	suite.NotNil(resp)
	suite.Len(resp.Msg.Results, 1)  // Only existing key should be in results
	suite.Len(resp.Msg.NotFound, 1) // One key should be in not found

	suite.Equal("value", resp.Msg.Results["exists"].Value)
	suite.Contains(resp.Msg.NotFound, "not-exists")
}

// Test BatchSet method
func (suite *StateGRPCServerTestSuite) TestBatchSet_Success() {
	req := connect.NewRequest(&statev1.BatchSetRequest{
		Items: []*statev1.BatchSetRequest_SetItem{
			{Key: "batch-set-key1", Value: "batch-set-value1"},
			{Key: "batch-set-key2", Value: "batch-set-value2"},
		},
		Scope: "batch-scope",
	})

	resp, err := suite.server.BatchSet(suite.ctx, req)
	suite.NoError(err)
	suite.NotNil(resp)
	suite.Len(resp.Msg.Results, 2)
	suite.Len(resp.Msg.Errors, 0)

	// Verify both values were set
	value1, err := suite.store.Get("batch-scope", "batch-set-key1")
	suite.NoError(err)
	suite.Equal("batch-set-value1", value1)

	value2, err := suite.store.Get("batch-scope", "batch-set-key2")
	suite.NoError(err)
	suite.Equal("batch-set-value2", value2)
}

func (suite *StateGRPCServerTestSuite) TestBatchSet_EmptyKey() {
	req := connect.NewRequest(&statev1.BatchSetRequest{
		Items: []*statev1.BatchSetRequest_SetItem{
			{Key: "", Value: "value"},
		},
		Scope: "batch-scope",
	})

	resp, err := suite.server.BatchSet(suite.ctx, req)
	suite.Error(err)
	suite.Nil(resp)

	connectErr := err.(*connect.Error)
	suite.Equal(connect.CodeInvalidArgument, connectErr.Code())
}

// Test CreateScope method (if implemented)
func (suite *StateGRPCServerTestSuite) TestCreateScope_Success() {
	// Check if the method exists - CreateScope might not be implemented in gRPC server
	// For now, we'll test the store directly
	err := suite.store.CreateScope("new-scope")
	suite.NoError(err)

	// Verify scope exists by trying to set a value in it
	err = suite.store.Set("new-scope", "test", "value")
	suite.NoError(err)
}

// Test Health Check
func (suite *StateGRPCServerTestSuite) TestHealth_Healthy() {
	req := connect.NewRequest(&emptypb.Empty{})

	resp, err := suite.server.Health(suite.ctx, req)
	suite.NoError(err)
	suite.NotNil(resp)
	suite.Equal(healthv1.HealthCheckResponse_SERVING_STATUS_SERVING, resp.Msg.Status)
	suite.NotEmpty(resp.Msg.Message)
	suite.NotNil(resp.Msg.Timestamp)
	suite.NotNil(resp.Msg.Uptime)
}

// Test Watch method (streaming)
func (suite *StateGRPCServerTestSuite) TestWatch_Implementation() {
	// Note: Watch method requires streaming interface testing
	// Since Watch is currently unimplemented, we'll verify it returns an error
	// This test would need to be expanded when Watch is implemented

	// For now, we can test that the method exists but returns unimplemented
	// This is more of a placeholder test until Watch functionality is added
	suite.T().Skip("Watch method testing requires streaming interface - to be implemented when Watch functionality is added")
}

// Test server initialization
func (suite *StateGRPCServerTestSuite) TestNewStateGRPCServer() {
	store := NewStore()
	server := NewStateGRPCServer(store)

	suite.NotNil(server)
	suite.Equal(store, server.store)
	suite.WithinDuration(time.Now(), server.startTime, time.Second)
}

// Benchmark tests
func BenchmarkStateGRPCServer_Get(b *testing.B) {
	store := NewStore()
	err := store.Load(":memory:")
	if err != nil {
		b.Fatal(err)
	}
	defer store.Unload()

	server := NewStateGRPCServer(store)
	ctx := context.Background()

	// Setup test data
	store.CreateScope("bench-scope")
	store.Set("bench-scope", "bench-key", "bench-value")

	req := connect.NewRequest(&statev1.GetRequest{
		Key:   "bench-key",
		Scope: "bench-scope",
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := server.Get(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStateGRPCServer_Set(b *testing.B) {
	store := NewStore()
	err := store.Load(":memory:")
	if err != nil {
		b.Fatal(err)
	}
	defer store.Unload()

	server := NewStateGRPCServer(store)
	ctx := context.Background()

	// Create test scope
	store.CreateScope("bench-scope")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := connect.NewRequest(&statev1.SetRequest{
			Key:   fmt.Sprintf("bench-key-%d", i),
			Value: fmt.Sprintf("bench-value-%d", i),
			Scope: "bench-scope",
		})

		_, err := server.Set(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}
