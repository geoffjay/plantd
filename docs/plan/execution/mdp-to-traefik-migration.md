# MDP to Traefik + gRPC Migration Execution Plan

**Project**: plantd Distributed Control System  
**Migration**: ZeroMQ MDP → Traefik + gRPC/Connect  
**Timeline**: 8 weeks  
**Status**: Planning Phase  

## Executive Summary

This document provides a detailed execution plan for migrating plantd's request/reply messaging from ZeroMQ's Majordomo Protocol (MDP) to a modern Traefik + gRPC/Connect architecture. The migration preserves the centralized routing benefits of the current broker while dramatically simplifying the implementation.

## Migration Overview

### Current Architecture
```
CLI/Web → MDP Broker → MDP Workers (State, Identity, etc.)
```

### Target Architecture  
```
CLI/Web → Traefik Gateway → gRPC Services (State, Identity, etc.)
```

### Key Benefits
- 85% reduction in messaging infrastructure code
- Elimination of CGO-related reliability issues
- Standard HTTP/2 tooling and observability
- Simplified deployment and configuration

## Phase 1: Foundation Setup (Week 1)

### 1.1 Buf Tooling Setup

**Deliverables:**
- Buf CLI installation and workspace configuration
- Protocol buffer schema structure
- Code generation pipeline
- CI/CD integration

**Tasks:**

#### 1.1.1 Install and Configure Buf CLI
```bash
# Install Buf CLI
curl -sSL "https://github.com/bufbuild/buf/releases/latest/download/buf-$(uname -s)-$(uname -m)" -o "/usr/local/bin/buf"
chmod +x "/usr/local/bin/buf"

# Verify installation
buf --version
```

#### 1.1.2 Create Workspace Structure
```
plantd/
├── proto/
│   └── plantd/
│       ├── common/v1/           # Shared types
│       ├── health/v1/           # Health/heartbeat
│       ├── state/v1/            # State service
│       ├── identity/v1/         # Identity service
│       ├── broker/v1/           # Broker service  
│       └── gateway/v1/          # Gateway routing
├── buf.work.yaml
├── proto/buf.yaml
├── buf.gen.yaml
└── gen/
    └── proto/
        └── go/
            └── plantd/
```

#### 1.1.3 Configure Buf Workspace

**buf.work.yaml:**
```yaml
version: v2
directories:
  - proto
```

**proto/buf.yaml:**
```yaml
version: v2
name: buf.build/plantd/apis
deps:
  - buf.build/googleapis/googleapis
  - buf.build/connectrpc/connect
  - buf.build/protocolbuffers/wellknowntypes
breaking:
  use:
    - FILE
lint:
  use:
    - BASIC
    - COMMENTS
    - STYLE_GUIDE
  except:
    - UNARY_RPC  # Allow streaming where beneficial
```

**buf.gen.yaml:**
```yaml
version: v2
managed:
  enabled: true
  override:
    - file_option: go_package_prefix
      value: github.com/geoffjay/plantd/gen/proto/go
plugins:
  - remote: buf.build/protocolbuffers/go:v1.31.0
    out: gen/proto/go
    opt:
      - paths=source_relative
  - remote: buf.build/connectrpc/go:v1.12.0
    out: gen/proto/go
    opt:
      - paths=source_relative
  - remote: buf.build/grpc-ecosystem/gateway/v2:v2.18.0
    out: gen/proto/go
    opt:
      - paths=source_relative
      - generate_unbound_methods=true
```

### 1.2 Common Protocol Definitions

**Deliverables:**
- Shared types and error definitions
- Health check protocol
- Authentication/authorization types

#### 1.2.1 Common Types

**proto/plantd/common/v1/common.proto:**
```protobuf
syntax = "proto3";

package plantd.common.v1;

import "google/protobuf/timestamp.proto";
import "google/protobuf/any.proto";

option go_package = "github.com/geoffjay/plantd/gen/proto/go/plantd/common/v1;commonv1";

// Standard error response
message Error {
  enum Code {
    CODE_UNSPECIFIED = 0;
    CODE_INVALID_ARGUMENT = 1;
    CODE_NOT_FOUND = 2;
    CODE_ALREADY_EXISTS = 3;
    CODE_PERMISSION_DENIED = 4;
    CODE_UNAUTHENTICATED = 5;
    CODE_INTERNAL = 6;
    CODE_UNAVAILABLE = 7;
  }
  
  Code code = 1;
  string message = 2;
  map<string, string> details = 3;
}

// Pagination support
message PageRequest {
  int32 page_size = 1;
  string page_token = 2;
}

message PageResponse {
  string next_page_token = 1;
  int32 total_count = 2;
}

// Metadata for all entities
message Metadata {
  string id = 1;
  google.protobuf.Timestamp created_at = 2;
  google.protobuf.Timestamp updated_at = 3;
  int64 version = 4;
  map<string, string> labels = 5;
}
```

#### 1.2.2 Health Check Protocol

**proto/plantd/health/v1/health.proto:**
```protobuf
syntax = "proto3";

package plantd.health.v1;

import "google/protobuf/timestamp.proto";
import "google/protobuf/duration.proto";

option go_package = "github.com/geoffjay/plantd/gen/proto/go/plantd/health/v1;healthv1";

// Health service for all plantd services
service HealthService {
  // Check service health (replaces MDP heartbeat)
  rpc Check(HealthCheckRequest) returns (HealthCheckResponse);
  
  // Watch service health changes
  rpc Watch(HealthWatchRequest) returns (stream HealthCheckResponse);
}

message HealthCheckRequest {
  string service = 1;  // Service name to check
}

message HealthCheckResponse {
  enum ServingStatus {
    SERVING_STATUS_UNSPECIFIED = 0;
    SERVING_STATUS_SERVING = 1;
    SERVING_STATUS_NOT_SERVING = 2;
    SERVING_STATUS_SERVICE_UNKNOWN = 3;
  }
  
  ServingStatus status = 1;
  string message = 2;
  google.protobuf.Timestamp timestamp = 3;
  google.protobuf.Duration uptime = 4;
  map<string, string> details = 5;
}

message HealthWatchRequest {
  string service = 1;
}
```

### 1.3 Makefile Integration

**Update Makefile:**
```makefile
# Add to existing Makefile

# Protocol buffer generation
.PHONY: proto-gen proto-lint proto-breaking

proto-gen:
	buf generate

proto-lint:
	buf lint

proto-breaking:
	buf breaking --against '.git#branch=main'

proto-clean:
	rm -rf gen/proto

# Update build targets to include proto generation
build-all: proto-gen build-broker build-state build-identity build-app build-client

# Add proto validation to CI
ci-proto: proto-lint proto-breaking proto-gen
	@echo "Protocol buffer validation complete"
```

## Phase 2: Service Protocol Definitions (Week 2)

### 2.1 State Service Protocol

**Deliverables:**
- Complete State service gRPC definition
- Migration mapping from MDP commands
- Validation and error handling

#### 2.1.1 State Service Definition

**proto/plantd/state/v1/state.proto:**
```protobuf
syntax = "proto3";

package plantd.state.v1;

import "google/protobuf/timestamp.proto";
import "google/protobuf/empty.proto";
import "plantd/common/v1/common.proto";
import "plantd/health/v1/health.proto";

option go_package = "github.com/geoffjay/plantd/gen/proto/go/plantd/state/v1;statev1";

// StateService manages distributed key-value state
service StateService {
  // Basic CRUD operations
  rpc Get(GetRequest) returns (GetResponse);
  rpc Set(SetRequest) returns (SetResponse);
  rpc Delete(DeleteRequest) returns (DeleteResponse);
  
  // Bulk operations
  rpc BatchGet(BatchGetRequest) returns (BatchGetResponse);
  rpc BatchSet(BatchSetRequest) returns (BatchSetResponse);
  
  // Listing and searching
  rpc List(ListRequest) returns (stream ListResponse);
  rpc Search(SearchRequest) returns (SearchResponse);
  
  // Real-time updates
  rpc Watch(WatchRequest) returns (stream WatchResponse);
  
  // Service management
  rpc Health(google.protobuf.Empty) returns (health.v1.HealthCheckResponse);
}

// MDP Command Mapping:
// "get" -> Get
// "set" -> Set  
// "delete" -> Delete
// "list" -> List
// "heartbeat" -> Health

message GetRequest {
  string key = 1;
  string scope = 2;  // Organization/user scope
  bool include_metadata = 3;
}

message GetResponse {
  string value = 1;
  common.v1.Metadata metadata = 2;
}

message SetRequest {
  string key = 1;
  string value = 2;
  string scope = 3;
  bool create_only = 4;  // Fail if exists
  google.protobuf.Timestamp expires_at = 5;  // TTL support
}

message SetResponse {
  common.v1.Metadata metadata = 1;
}

message DeleteRequest {
  string key = 1;
  string scope = 2;
}

message DeleteResponse {
  bool existed = 1;
  common.v1.Metadata last_metadata = 2;
}

message ListRequest {
  string prefix = 1;
  string scope = 2;
  common.v1.PageRequest page = 3;
  bool include_values = 4;
}

message ListResponse {
  string key = 1;
  string value = 2;  // Only if include_values=true
  common.v1.Metadata metadata = 3;
  common.v1.PageResponse page = 4;
}

message WatchRequest {
  string key_prefix = 1;
  string scope = 2;
  bool include_initial = 3;  // Send current values first
}

message WatchResponse {
  enum EventType {
    EVENT_TYPE_UNSPECIFIED = 0;
    EVENT_TYPE_CREATED = 1;
    EVENT_TYPE_UPDATED = 2;
    EVENT_TYPE_DELETED = 3;
  }
  
  EventType event_type = 1;
  string key = 2;
  string value = 3;
  common.v1.Metadata metadata = 4;
}

// Batch operations for efficiency
message BatchGetRequest {
  repeated string keys = 1;
  string scope = 2;
}

message BatchGetResponse {
  map<string, GetResponse> results = 1;
  repeated string not_found = 2;
}

message BatchSetRequest {
  message SetItem {
    string key = 1;
    string value = 2;
  }
  
  repeated SetItem items = 1;
  string scope = 2;
}

message BatchSetResponse {
  map<string, SetResponse> results = 1;
  repeated common.v1.Error errors = 2;
}

message SearchRequest {
  string query = 1;  // Search pattern
  string scope = 2;
  common.v1.PageRequest page = 3;
}

message SearchResponse {
  repeated ListResponse results = 1;
  common.v1.PageResponse page = 2;
}
```

### 2.2 Identity Service Protocol

**proto/plantd/identity/v1/identity.proto:**
```protobuf
syntax = "proto3";

package plantd.identity.v1;

import "google/protobuf/timestamp.proto";
import "google/protobuf/empty.proto";
import "plantd/common/v1/common.proto";
import "plantd/health/v1/health.proto";

option go_package = "github.com/geoffjay/plantd/gen/proto/go/plantd/identity/v1;identityv1";

// IdentityService handles authentication and authorization
service IdentityService {
  // Authentication
  rpc Login(LoginRequest) returns (LoginResponse);
  rpc Logout(LogoutRequest) returns (google.protobuf.Empty);
  rpc RefreshToken(RefreshTokenRequest) returns (RefreshTokenResponse);
  rpc ValidateToken(ValidateTokenRequest) returns (ValidateTokenResponse);
  
  // User management
  rpc CreateUser(CreateUserRequest) returns (CreateUserResponse);
  rpc GetUser(GetUserRequest) returns (GetUserResponse);
  rpc UpdateUser(UpdateUserRequest) returns (UpdateUserResponse);
  rpc DeleteUser(DeleteUserRequest) returns (google.protobuf.Empty);
  rpc ListUsers(ListUsersRequest) returns (stream ListUsersResponse);
  
  // Organization management
  rpc CreateOrganization(CreateOrganizationRequest) returns (CreateOrganizationResponse);
  rpc GetOrganization(GetOrganizationRequest) returns (GetOrganizationResponse);
  
  // Role and permission management
  rpc AssignRole(AssignRoleRequest) returns (google.protobuf.Empty);
  rpc RevokeRole(RevokeRoleRequest) returns (google.protobuf.Empty);
  rpc CheckPermission(CheckPermissionRequest) returns (CheckPermissionResponse);
  
  // Service management
  rpc Health(google.protobuf.Empty) returns (health.v1.HealthCheckResponse);
}

// Authentication messages
message LoginRequest {
  string email = 1;
  string password = 2;
  string organization_id = 3;
}

message LoginResponse {
  string access_token = 1;
  string refresh_token = 2;
  google.protobuf.Timestamp expires_at = 3;
  User user = 4;
}

message ValidateTokenRequest {
  string token = 1;
  repeated string required_permissions = 2;
}

message ValidateTokenResponse {
  bool valid = 1;
  User user = 2;
  repeated string permissions = 3;
  google.protobuf.Timestamp expires_at = 4;
}

// User management messages
message User {
  string id = 1;
  string email = 2;
  string name = 3;
  repeated string roles = 4;
  string organization_id = 5;
  common.v1.Metadata metadata = 6;
  google.protobuf.Timestamp last_login = 7;
  bool active = 8;
}

message CreateUserRequest {
  string email = 1;
  string name = 2;
  string password = 3;
  string organization_id = 4;
  repeated string roles = 5;
}

message CreateUserResponse {
  User user = 1;
}

// ... other message definitions
```

### 2.3 Protocol Generation and Validation

**Tasks:**
```bash
# Generate initial code
make proto-gen

# Validate schemas
make proto-lint

# Test breaking changes
make proto-breaking

# Commit protocol definitions
git add proto/ gen/
git commit -m "feat: Add gRPC protocol definitions for State and Identity services"
```

## Phase 3: Core Service Implementation (Week 3)

### 3.1 State Service gRPC Implementation

**Deliverables:**
- Complete State service gRPC server
- MDP compatibility layer
- Unit tests

#### 3.1.1 State Service Server

**state/grpc_server.go:**
```go
package main

import (
    "context"
    "fmt"
    "log"
    "net/http"
    
    "connectrpc.com/connect"
    
    statev1 "github.com/geoffjay/plantd/gen/proto/go/plantd/state/v1"
    "github.com/geoffjay/plantd/gen/proto/go/plantd/state/v1/statev1connect"
    healthv1 "github.com/geoffjay/plantd/gen/proto/go/plantd/health/v1"
)

type StateGRPCServer struct {
    store *Store  // Existing store implementation
}

func NewStateGRPCServer(store *Store) *StateGRPCServer {
    return &StateGRPCServer{store: store}
}

func (s *StateGRPCServer) Get(ctx context.Context, req *connect.Request[statev1.GetRequest]) (*connect.Response[statev1.GetResponse], error) {
    log.Printf("State.Get: key=%s scope=%s", req.Msg.Key, req.Msg.Scope)
    
    value, err := s.store.Get(req.Msg.Key, req.Msg.Scope)
    if err != nil {
        if err == ErrNotFound {
            return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("key not found: %s", req.Msg.Key))
        }
        return nil, connect.NewError(connect.CodeInternal, err)
    }
    
    response := &statev1.GetResponse{
        Value: value.Data,
    }
    
    if req.Msg.IncludeMetadata {
        response.Metadata = &commonv1.Metadata{
            Id:        value.ID,
            CreatedAt: timestamppb.New(value.Created),
            UpdatedAt: timestamppb.New(value.Updated),
            Version:   value.Version,
        }
    }
    
    return connect.NewResponse(response), nil
}

func (s *StateGRPCServer) Set(ctx context.Context, req *connect.Request[statev1.SetRequest]) (*connect.Response[statev1.SetResponse], error) {
    log.Printf("State.Set: key=%s scope=%s", req.Msg.Key, req.Msg.Scope)
    
    opts := &SetOptions{
        CreateOnly: req.Msg.CreateOnly,
    }
    
    if req.Msg.ExpiresAt != nil {
        opts.ExpiresAt = req.Msg.ExpiresAt.AsTime()
    }
    
    result, err := s.store.Set(req.Msg.Key, req.Msg.Value, req.Msg.Scope, opts)
    if err != nil {
        if err == ErrAlreadyExists {
            return nil, connect.NewError(connect.CodeAlreadyExists, err)
        }
        return nil, connect.NewError(connect.CodeInternal, err)
    }
    
    response := &statev1.SetResponse{
        Metadata: &commonv1.Metadata{
            Id:        result.ID,
            CreatedAt: timestamppb.New(result.Created),
            UpdatedAt: timestamppb.New(result.Updated),
            Version:   result.Version,
        },
    }
    
    return connect.NewResponse(response), nil
}

func (s *StateGRPCServer) Health(ctx context.Context, req *connect.Request[emptypb.Empty]) (*connect.Response[healthv1.HealthCheckResponse], error) {
    // Check store health
    healthy := s.store.IsHealthy()
    
    status := healthv1.HealthCheckResponse_SERVING_STATUS_SERVING
    message := "State service is healthy"
    
    if !healthy {
        status = healthv1.HealthCheckResponse_SERVING_STATUS_NOT_SERVING
        message = "State service is unhealthy"
    }
    
    response := &healthv1.HealthCheckResponse{
        Status:    status,
        Message:   message,
        Timestamp: timestamppb.Now(),
        Uptime:    durationpb.New(time.Since(s.startTime)),
    }
    
    return connect.NewResponse(response), nil
}

// ... implement other methods
```

#### 3.1.2 MDP Compatibility Layer

**state/mdp_compat.go:**
```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "strings"
    
    statev1 "github.com/geoffjay/plantd/gen/proto/go/plantd/state/v1"
    "github.com/geoffjay/plantd/gen/proto/go/plantd/state/v1/statev1connect"
)

// MDPCompatibilityHandler wraps gRPC service for MDP compatibility
type MDPCompatibilityHandler struct {
    grpcClient statev1connect.StateServiceClient
}

func NewMDPCompatibilityHandler(grpcClient statev1connect.StateServiceClient) *MDPCompatibilityHandler {
    return &MDPCompatibilityHandler{grpcClient: grpcClient}
}

// HandleMDPRequest processes MDP-style requests and converts to gRPC
func (h *MDPCompatibilityHandler) HandleMDPRequest(command string, args []string) ([]string, error) {
    ctx := context.Background()
    
    switch command {
    case "get":
        if len(args) < 1 {
            return nil, fmt.Errorf("get requires key argument")
        }
        
        req := &statev1.GetRequest{
            Key:   args[0],
            Scope: "default", // Default scope for MDP compatibility
        }
        
        resp, err := h.grpcClient.Get(ctx, connect.NewRequest(req))
        if err != nil {
            return nil, err
        }
        
        return []string{resp.Msg.Value}, nil
        
    case "set":
        if len(args) < 2 {
            return nil, fmt.Errorf("set requires key and value arguments")
        }
        
        req := &statev1.SetRequest{
            Key:   args[0],
            Value: args[1],
            Scope: "default",
        }
        
        _, err := h.grpcClient.Set(ctx, connect.NewRequest(req))
        if err != nil {
            return nil, err
        }
        
        return []string{"OK"}, nil
        
    case "delete":
        if len(args) < 1 {
            return nil, fmt.Errorf("delete requires key argument")
        }
        
        req := &statev1.DeleteRequest{
            Key:   args[0],
            Scope: "default",
        }
        
        resp, err := h.grpcClient.Delete(ctx, connect.NewRequest(req))
        if err != nil {
            return nil, err
        }
        
        if resp.Msg.Existed {
            return []string{"DELETED"}, nil
        }
        return []string{"NOT_FOUND"}, nil
        
    default:
        return nil, fmt.Errorf("unknown command: %s", command)
    }
}
```

### 3.2 Identity Service gRPC Implementation

**identity/grpc_server.go:**
```go
package main

import (
    "context"
    "log"
    
    "connectrpc.com/connect"
    
    identityv1 "github.com/geoffjay/plantd/gen/proto/go/plantd/identity/v1"
    "github.com/geoffjay/plantd/gen/proto/go/plantd/identity/v1/identityv1connect"
)

type IdentityGRPCServer struct {
    authService *AuthService  // Existing auth service
    userService *UserService  // Existing user service
}

func NewIdentityGRPCServer(authService *AuthService, userService *UserService) *IdentityGRPCServer {
    return &IdentityGRPCServer{
        authService: authService,
        userService: userService,
    }
}

func (s *IdentityGRPCServer) Login(ctx context.Context, req *connect.Request[identityv1.LoginRequest]) (*connect.Response[identityv1.LoginResponse], error) {
    log.Printf("Identity.Login: email=%s org=%s", req.Msg.Email, req.Msg.OrganizationId)
    
    // Use existing auth service
    token, user, err := s.authService.Login(req.Msg.Email, req.Msg.Password, req.Msg.OrganizationId)
    if err != nil {
        return nil, connect.NewError(connect.CodeUnauthenticated, err)
    }
    
    response := &identityv1.LoginResponse{
        AccessToken:  token.AccessToken,
        RefreshToken: token.RefreshToken,
        ExpiresAt:    timestamppb.New(token.ExpiresAt),
        User:         convertUserToProto(user),
    }
    
    return connect.NewResponse(response), nil
}

func (s *IdentityGRPCServer) ValidateToken(ctx context.Context, req *connect.Request[identityv1.ValidateTokenRequest]) (*connect.Response[identityv1.ValidateTokenResponse], error) {
    user, permissions, err := s.authService.ValidateToken(req.Msg.Token)
    if err != nil {
        return &connect.Response[identityv1.ValidateTokenResponse]{
            Msg: &identityv1.ValidateTokenResponse{Valid: false},
        }, nil
    }
    
    response := &identityv1.ValidateTokenResponse{
        Valid:       true,
        User:        convertUserToProto(user),
        Permissions: permissions,
    }
    
    return connect.NewResponse(response), nil
}

// ... implement other methods
```

### 3.3 Service Integration Tests

**state/grpc_test.go:**
```go
package main

import (
    "context"
    "net/http"
    "testing"
    
    "connectrpc.com/connect"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    
    statev1 "github.com/geoffjay/plantd/gen/proto/go/plantd/state/v1"
    "github.com/geoffjay/plantd/gen/proto/go/plantd/state/v1/statev1connect"
)

func TestStateGRPCServer(t *testing.T) {
    // Setup test server
    store := NewMemoryStore()
    server := NewStateGRPCServer(store)
    
    mux := http.NewServeMux()
    path, handler := statev1connect.NewStateServiceHandler(server)
    mux.Handle(path, handler)
    
    // Start test server
    testServer := httptest.NewServer(mux)
    defer testServer.Close()
    
    // Create client
    client := statev1connect.NewStateServiceClient(
        http.DefaultClient,
        testServer.URL,
    )
    
    ctx := context.Background()
    
    t.Run("Set and Get", func(t *testing.T) {
        // Set value
        setReq := &statev1.SetRequest{
            Key:   "test-key",
            Value: "test-value",
            Scope: "test-scope",
        }
        
        setResp, err := client.Set(ctx, connect.NewRequest(setReq))
        require.NoError(t, err)
        assert.NotNil(t, setResp.Msg.Metadata)
        
        // Get value
        getReq := &statev1.GetRequest{
            Key:   "test-key",
            Scope: "test-scope",
        }
        
        getResp, err := client.Get(ctx, connect.NewRequest(getReq))
        require.NoError(t, err)
        assert.Equal(t, "test-value", getResp.Msg.Value)
    })
    
    t.Run("Get Non-existent Key", func(t *testing.T) {
        getReq := &statev1.GetRequest{
            Key:   "non-existent",
            Scope: "test-scope",
        }
        
        _, err := client.Get(ctx, connect.NewRequest(getReq))
        require.Error(t, err)
        
        connectErr := err.(*connect.Error)
        assert.Equal(t, connect.CodeNotFound, connectErr.Code())
    })
}
```

## Phase 4: Traefik Gateway Setup (Week 4)

### 4.1 Traefik Configuration

**Deliverables:**
- Development Traefik configuration
- Docker Compose setup
- Service discovery configuration
- Health check integration

#### 4.1.1 Development Configuration

**traefik/traefik.dev.yml:**
```yaml
# Traefik development configuration
api:
  dashboard: true
  debug: true

entryPoints:
  grpc:
    address: ":8080"
  web:
    address: ":80"
  traefik:
    address: ":8081"

providers:
  docker:
    endpoint: "unix:///var/run/docker.sock"
    exposedByDefault: false
    network: plantd-network
  
  file:
    filename: /etc/traefik/dynamic.yml
    watch: true

certificatesResolvers:
  letsencrypt:
    acme:
      email: admin@plantd.local
      storage: /letsencrypt/acme.json
      httpChallenge:
        entryPoint: web

log:
  level: DEBUG

accessLog: {}

metrics:
  prometheus:
    addEntryPointsLabels: true
    addServicesLabels: true

tracing:
  jaeger:
    samplingParam: 1.0
    localAgentHostPort: jaeger:6831
```

**traefik/dynamic.yml:**
```yaml
# Dynamic configuration for service routing
http:
  middlewares:
    auth:
      forwardAuth:
        address: "http://identity-service:8002/validate"
        authResponseHeaders:
          - "X-User-Id"
          - "X-User-Email"
          - "X-User-Roles"
    
    cors:
      headers:
        accessControlAllowMethods:
          - GET
          - POST
          - PUT
          - DELETE
        accessControlAllowOriginList:
          - "*"
        accessControlAllowHeaders:
          - "*"
    
    retry:
      attempts: 3
      initialInterval: 100ms
    
    circuit-breaker:
      expression: "NetworkErrorRatio() > 0.30"

  routers:
    # State service routing
    state-service:
      rule: "PathPrefix(`/plantd.state.v1.StateService`)"
      service: state-service
      entryPoints:
        - grpc
      middlewares:
        - auth
        - retry
        - circuit-breaker
    
    # Identity service routing  
    identity-service:
      rule: "PathPrefix(`/plantd.identity.v1.IdentityService`)"
      service: identity-service
      entryPoints:
        - grpc
      middlewares:
        - retry
        - circuit-breaker
    
    # Health checks (no auth required)
    health-checks:
      rule: "PathPrefix(`/plantd.health.v1.HealthService`)"
      service: health-service
      entryPoints:
        - grpc

  services:
    state-service:
      loadBalancer:
        servers:
          - url: "h2c://state-service:8001"
        healthCheck:
          path: "/plantd.health.v1.HealthService/Check"
          interval: "30s"
          timeout: "5s"
    
    identity-service:
      loadBalancer:
        servers:
          - url: "h2c://identity-service:8002"
        healthCheck:
          path: "/plantd.health.v1.HealthService/Check"
          interval: "30s"
          timeout: "5s"
```

#### 4.1.2 Docker Compose Integration

**docker-compose.plantd.yml:**
```yaml
version: '3.8'

networks:
  plantd-network:
    driver: bridge

services:
  traefik:
    image: traefik:v3.0
    container_name: plantd-traefik
    command:
      - --configFile=/etc/traefik/traefik.yml
    ports:
      - "8080:8080"  # gRPC gateway
      - "80:80"      # HTTP
      - "8081:8081"  # Traefik dashboard
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - ./traefik/traefik.dev.yml:/etc/traefik/traefik.yml:ro
      - ./traefik/dynamic.yml:/etc/traefik/dynamic.yml:ro
      - ./data/letsencrypt:/letsencrypt
    networks:
      - plantd-network
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.traefik.rule=Host(`traefik.plantd.local`)"
      - "traefik.http.routers.traefik.entrypoints=web"
    restart: unless-stopped

  state-service:
    build: 
      context: ./state
      dockerfile: Dockerfile.grpc
    container_name: plantd-state
    environment:
      - PLANTD_STATE_GRPC_PORT=8001
      - PLANTD_STATE_STORE_PATH=/data/state
    volumes:
      - ./data/state:/data/state
    networks:
      - plantd-network
    labels:
      - "traefik.enable=true"
      - "traefik.http.services.state.loadbalancer.server.scheme=h2c"
      - "traefik.http.services.state.loadbalancer.server.port=8001"
    restart: unless-stopped
    depends_on:
      - traefik

  identity-service:
    build:
      context: ./identity
      dockerfile: Dockerfile.grpc
    container_name: plantd-identity
    environment:
      - PLANTD_IDENTITY_GRPC_PORT=8002
      - PLANTD_IDENTITY_DB_HOST=postgres
      - PLANTD_IDENTITY_DB_NAME=identity
    networks:
      - plantd-network
    labels:
      - "traefik.enable=true"
      - "traefik.http.services.identity.loadbalancer.server.scheme=h2c"
      - "traefik.http.services.identity.loadbalancer.server.port=8002"
    restart: unless-stopped
    depends_on:
      - postgres
      - traefik

  postgres:
    image: postgres:15
    container_name: plantd-postgres
    environment:
      - POSTGRES_DB=identity
      - POSTGRES_USER=plantd
      - POSTGRES_PASSWORD=plantd-dev
    volumes:
      - ./data/postgres:/var/lib/postgresql/data
    networks:
      - plantd-network
    restart: unless-stopped

  # Observability stack
  prometheus:
    image: prom/prometheus:latest
    container_name: plantd-prometheus
    volumes:
      - ./monitoring/prometheus.yml:/etc/prometheus/prometheus.yml
    networks:
      - plantd-network
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.prometheus.rule=Host(`prometheus.plantd.local`)"

  jaeger:
    image: jaegertracing/all-in-one:latest
    container_name: plantd-jaeger
    networks:
      - plantd-network
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.jaeger.rule=Host(`jaeger.plantd.local`)"
```

### 4.2 Service Dockerfiles

**state/Dockerfile.grpc:**
```dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o state-grpc ./cmd/grpc

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/state-grpc .
COPY --from=builder /app/config.yml .

EXPOSE 8001

CMD ["./state-grpc"]
```

### 4.3 Development Scripts

**scripts/dev-start.sh:**
```bash
#!/bin/bash
set -e

echo "Starting plantd development environment with Traefik..."

# Generate protocols if needed
if [ ! -d "gen/proto" ]; then
    echo "Generating protocol buffers..."
    make proto-gen
fi

# Create data directories
mkdir -p data/{state,postgres,letsencrypt}

# Start services
docker-compose -f docker-compose.plantd.yml up -d

echo "Services starting..."
echo "- Traefik Dashboard: http://localhost:8081"
echo "- gRPC Gateway: http://localhost:8080"
echo "- Prometheus: http://prometheus.plantd.local"
echo "- Jaeger: http://jaeger.plantd.local"

# Wait for services to be healthy
echo "Waiting for services to be healthy..."
sleep 10

# Test connectivity
echo "Testing service connectivity..."
grpcurl -plaintext localhost:8080 plantd.health.v1.HealthService/Check

echo "Development environment ready!"
```

This completes the first half of the execution plan. The plan continues with client implementation, integration testing, production deployment, and documentation phases. Would you like me to continue with the remaining phases? 
