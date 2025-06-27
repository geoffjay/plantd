package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	commonv1 "github.com/geoffjay/plantd/gen/proto/go/plantd/common/v1"
	healthv1 "github.com/geoffjay/plantd/gen/proto/go/plantd/health/v1"
	statev1 "github.com/geoffjay/plantd/gen/proto/go/plantd/state/v1"
)

var (
	ErrNotFound      = errors.New("key not found")
	ErrAlreadyExists = errors.New("key already exists")
)

type StateGRPCServer struct {
	store     *Store    // Existing store implementation
	startTime time.Time // Server start time for uptime calculation
}

func NewStateGRPCServer(store *Store) *StateGRPCServer {
	return &StateGRPCServer{
		store:     store,
		startTime: time.Now(),
	}
}

func (s *StateGRPCServer) Get(ctx context.Context, req *connect.Request[statev1.GetRequest]) (*connect.Response[statev1.GetResponse], error) {
	log.Printf("State.Get: key=%s scope=%s", req.Msg.Key, req.Msg.Scope)

	// Use default scope if not provided
	scope := req.Msg.Scope
	if scope == "" {
		scope = "default"
	}

	value, err := s.store.Get(scope, req.Msg.Key)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("key not found: %s", req.Msg.Key))
	}

	if value == "" {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("key not found: %s", req.Msg.Key))
	}

	response := &statev1.GetResponse{
		Value: value,
	}

	// Add basic metadata if requested
	if req.Msg.IncludeMetadata {
		response.Metadata = &commonv1.Metadata{
			Id:        fmt.Sprintf("%s/%s", scope, req.Msg.Key),
			CreatedAt: timestamppb.Now(), // Note: bolt store doesn't track creation time
			UpdatedAt: timestamppb.Now(),
			Version:   1, // Note: bolt store doesn't track versions
		}
	}

	return connect.NewResponse(response), nil
}

func (s *StateGRPCServer) Set(ctx context.Context, req *connect.Request[statev1.SetRequest]) (*connect.Response[statev1.SetResponse], error) {
	log.Printf("State.Set: key=%s scope=%s", req.Msg.Key, req.Msg.Scope)

	// Use default scope if not provided
	scope := req.Msg.Scope
	if scope == "" {
		scope = "default"
	}

	// Check if key exists for create_only mode
	if req.Msg.CreateOnly {
		existing, err := s.store.Get(scope, req.Msg.Key)
		if err == nil && existing != "" {
			return nil, connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("key already exists: %s", req.Msg.Key))
		}
	}

	// Note: Current store doesn't support TTL (expires_at), so we ignore it for now
	if req.Msg.ExpiresAt != nil {
		log.Printf("Warning: TTL not supported by current store implementation")
	}

	err := s.store.Set(scope, req.Msg.Key, req.Msg.Value)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	response := &statev1.SetResponse{
		Metadata: &commonv1.Metadata{
			Id:        fmt.Sprintf("%s/%s", scope, req.Msg.Key),
			CreatedAt: timestamppb.Now(),
			UpdatedAt: timestamppb.Now(),
			Version:   1, // Note: bolt store doesn't track versions
		},
	}

	return connect.NewResponse(response), nil
}

func (s *StateGRPCServer) Delete(ctx context.Context, req *connect.Request[statev1.DeleteRequest]) (*connect.Response[statev1.DeleteResponse], error) {
	log.Printf("State.Delete: key=%s scope=%s", req.Msg.Key, req.Msg.Scope)

	// Use default scope if not provided
	scope := req.Msg.Scope
	if scope == "" {
		scope = "default"
	}

	// Check if key exists before deletion
	existing, err := s.store.Get(scope, req.Msg.Key)
	existed := err == nil && existing != ""

	if existed {
		err = s.store.Delete(scope, req.Msg.Key)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	response := &statev1.DeleteResponse{
		Existed: existed,
	}

	if existed {
		response.LastMetadata = &commonv1.Metadata{
			Id:        fmt.Sprintf("%s/%s", scope, req.Msg.Key),
			CreatedAt: timestamppb.Now(), // Note: We don't have actual creation time
			UpdatedAt: timestamppb.Now(),
			Version:   1,
		}
	}

	return connect.NewResponse(response), nil
}

func (s *StateGRPCServer) BatchGet(ctx context.Context, req *connect.Request[statev1.BatchGetRequest]) (*connect.Response[statev1.BatchGetResponse], error) {
	log.Printf("State.BatchGet: keys=%v scope=%s", req.Msg.Keys, req.Msg.Scope)

	// Use default scope if not provided
	scope := req.Msg.Scope
	if scope == "" {
		scope = "default"
	}

	results := make(map[string]*statev1.GetResponse)
	var notFound []string

	for _, key := range req.Msg.Keys {
		value, err := s.store.Get(scope, key)
		if err != nil || value == "" {
			notFound = append(notFound, key)
			continue
		}

		results[key] = &statev1.GetResponse{
			Value: value,
			Metadata: &commonv1.Metadata{
				Id:        fmt.Sprintf("%s/%s", scope, key),
				CreatedAt: timestamppb.Now(),
				UpdatedAt: timestamppb.Now(),
				Version:   1,
			},
		}
	}

	response := &statev1.BatchGetResponse{
		Results:  results,
		NotFound: notFound,
	}

	return connect.NewResponse(response), nil
}

func (s *StateGRPCServer) BatchSet(ctx context.Context, req *connect.Request[statev1.BatchSetRequest]) (*connect.Response[statev1.BatchSetResponse], error) {
	log.Printf("State.BatchSet: items=%d scope=%s", len(req.Msg.Items), req.Msg.Scope)

	// Use default scope if not provided
	scope := req.Msg.Scope
	if scope == "" {
		scope = "default"
	}

	results := make(map[string]*statev1.SetResponse)
	var errors []*commonv1.Error

	for _, item := range req.Msg.Items {
		err := s.store.Set(scope, item.Key, item.Value)
		if err != nil {
			errors = append(errors, &commonv1.Error{
				Code:    commonv1.Error_CODE_INTERNAL,
				Message: fmt.Sprintf("Failed to set key %s: %v", item.Key, err),
				Details: map[string]string{"key": item.Key},
			})
			continue
		}

		results[item.Key] = &statev1.SetResponse{
			Metadata: &commonv1.Metadata{
				Id:        fmt.Sprintf("%s/%s", scope, item.Key),
				CreatedAt: timestamppb.Now(),
				UpdatedAt: timestamppb.Now(),
				Version:   1,
			},
		}
	}

	response := &statev1.BatchSetResponse{
		Results: results,
		Errors:  errors,
	}

	return connect.NewResponse(response), nil
}

func (s *StateGRPCServer) List(ctx context.Context, req *connect.Request[statev1.ListRequest], stream *connect.ServerStream[statev1.ListResponse]) error {
	log.Printf("State.List: prefix=%s scope=%s", req.Msg.Prefix, req.Msg.Scope)

	// Use default scope if not provided
	scope := req.Msg.Scope
	if scope == "" {
		scope = "default"
	}

	// Get all keys in the scope
	keys, err := s.store.ListAllKeys(scope)
	if err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}

	// Filter by prefix if provided
	filteredKeys := keys
	if req.Msg.Prefix != "" {
		filteredKeys = []string{}
		for _, key := range keys {
			if strings.HasPrefix(key, req.Msg.Prefix) {
				filteredKeys = append(filteredKeys, key)
			}
		}
	}

	// Stream the results
	for _, key := range filteredKeys {
		response := &statev1.ListResponse{
			Key: key,
			Metadata: &commonv1.Metadata{
				Id:        fmt.Sprintf("%s/%s", scope, key),
				CreatedAt: timestamppb.Now(),
				UpdatedAt: timestamppb.Now(),
				Version:   1,
			},
		}

		// Include values if requested
		if req.Msg.IncludeValues {
			value, err := s.store.Get(scope, key)
			if err == nil {
				response.Value = value
			}
		}

		if err := stream.Send(response); err != nil {
			return connect.NewError(connect.CodeInternal, err)
		}
	}

	return nil
}

func (s *StateGRPCServer) Search(ctx context.Context, req *connect.Request[statev1.SearchRequest]) (*connect.Response[statev1.SearchResponse], error) {
	log.Printf("State.Search: query=%s scope=%s", req.Msg.Query, req.Msg.Scope)

	// Use default scope if not provided
	scope := req.Msg.Scope
	if scope == "" {
		scope = "default"
	}

	// Simple search implementation - get all keys and filter by query
	keys, err := s.store.ListAllKeys(scope)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var results []*statev1.ListResponse
	for _, key := range keys {
		// Simple substring search
		if strings.Contains(key, req.Msg.Query) {
			value, err := s.store.Get(scope, key)
			if err != nil {
				continue
			}

			result := &statev1.ListResponse{
				Key:   key,
				Value: value,
				Metadata: &commonv1.Metadata{
					Id:        fmt.Sprintf("%s/%s", scope, key),
					CreatedAt: timestamppb.Now(),
					UpdatedAt: timestamppb.Now(),
					Version:   1,
				},
			}
			results = append(results, result)
		}
	}

	response := &statev1.SearchResponse{
		Results: results,
	}

	return connect.NewResponse(response), nil
}

func (s *StateGRPCServer) Watch(ctx context.Context, req *connect.Request[statev1.WatchRequest], stream *connect.ServerStream[statev1.WatchResponse]) error {
	log.Printf("State.Watch: key_prefix=%s scope=%s", req.Msg.KeyPrefix, req.Msg.Scope)

	// Note: Current store doesn't support watching, so we return an error for now
	// In a real implementation, this would need to be added to the store interface
	return connect.NewError(connect.CodeUnimplemented, fmt.Errorf("watch functionality not yet implemented"))
}

func (s *StateGRPCServer) Health(ctx context.Context, req *connect.Request[emptypb.Empty]) (*connect.Response[healthv1.HealthCheckResponse], error) {
	// Simple health check - verify store is accessible
	healthy := true
	message := "State service is healthy"

	// Try to access the store
	scopes := s.store.ListAllScope()
	if scopes == nil {
		healthy = false
		message = "State service is unhealthy - cannot access store"
	}

	status := healthv1.HealthCheckResponse_SERVING_STATUS_SERVING
	if !healthy {
		status = healthv1.HealthCheckResponse_SERVING_STATUS_NOT_SERVING
	}

	response := &healthv1.HealthCheckResponse{
		Status:    status,
		Message:   message,
		Timestamp: timestamppb.Now(),
		Uptime:    durationpb.New(time.Since(s.startTime)),
	}

	return connect.NewResponse(response), nil
}
