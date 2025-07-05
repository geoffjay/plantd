# PlantD gRPC API Reference

## Overview

This document provides a comprehensive API reference for the PlantD gRPC services, including all service interfaces, request/response formats, and usage examples. The PlantD project consists of several distributed services that provide building blocks for a distributed control system (DCS).

## Service Packages

The PlantD gRPC services are organized into the following packages:

- `plantd.health.v1` - Health check service
- `plantd.identity.v1` - Identity and authentication service
- `plantd.state.v1` - Distributed state management service
- `plantd.common.v1` - Common types and utilities

## Common Types

### Error Handling

All services use a standard error response format:

```proto
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
```

### Pagination

Standard pagination support across all services:

```proto
message PageRequest {
  int32 page_size = 1;
  string page_token = 2;
}

message PageResponse {
  string next_page_token = 1;
  int32 total_count = 2;
}
```

### Metadata

All entities include standard metadata:

```proto
message Metadata {
  string id = 1;
  google.protobuf.Timestamp created_at = 2;
  google.protobuf.Timestamp updated_at = 3;
  int64 version = 4;
  map<string, string> labels = 5;
}
```

## Health Service

The Health service provides health check functionality for all PlantD services, replacing the traditional MDP heartbeat mechanism.

### Service Definition

```proto
service HealthService {
  rpc Check(HealthCheckRequest) returns (HealthCheckResponse);
  rpc Watch(HealthWatchRequest) returns (stream HealthCheckResponse);
}
```

### Methods

#### Check

Performs a health check for a specific service.

**Request:**
```proto
message HealthCheckRequest {
  string service = 1;  // Service name to check
}
```

**Response:**
```proto
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
```

**Example:**
```bash
grpcurl -plaintext -d '{"service": "identity"}' localhost:8080 plantd.health.v1.HealthService/Check
```

#### Watch

Watches for health status changes of a service.

**Request:**
```proto
message HealthWatchRequest {
  string service = 1;
}
```

**Response:**
Stream of `HealthCheckResponse` messages whenever the service status changes.

**Example:**
```bash
grpcurl -plaintext -d '{"service": "identity"}' localhost:8080 plantd.health.v1.HealthService/Watch
```

## Identity Service

The Identity service provides comprehensive authentication, authorization, and user management capabilities.

### Service Definition

```proto
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
  rpc Health(google.protobuf.Empty) returns (plantd.health.v1.HealthCheckResponse);
}
```

### Authentication Methods

#### Login

Authenticates a user and returns access tokens.

**Request:**
```proto
message LoginRequest {
  string email = 1;
  string password = 2;
  string organization_id = 3;
}
```

**Response:**
```proto
message LoginResponse {
  string access_token = 1;
  string refresh_token = 2;
  google.protobuf.Timestamp expires_at = 3;
  User user = 4;
}
```

**Example:**
```bash
grpcurl -plaintext -d '{
  "email": "user@example.com",
  "password": "password123",
  "organization_id": "org-123"
}' localhost:8080 plantd.identity.v1.IdentityService/Login
```

#### Logout

Logs out a user and invalidates their tokens.

**Request:**
```proto
message LogoutRequest {
  string token = 1;
}
```

**Response:**
Empty response on success.

#### RefreshToken

Refreshes an access token using a refresh token.

**Request:**
```proto
message RefreshTokenRequest {
  string refresh_token = 1;
}
```

**Response:**
```proto
message RefreshTokenResponse {
  string access_token = 1;
  string refresh_token = 2;
  google.protobuf.Timestamp expires_at = 3;
}
```

#### ValidateToken

Validates a token and returns user information.

**Request:**
```proto
message ValidateTokenRequest {
  string token = 1;
  repeated string required_permissions = 2;
}
```

**Response:**
```proto
message ValidateTokenResponse {
  bool valid = 1;
  User user = 2;
  repeated string permissions = 3;
  google.protobuf.Timestamp expires_at = 4;
}
```

### User Management Methods

#### CreateUser

Creates a new user account.

**Request:**
```proto
message CreateUserRequest {
  string email = 1;
  string name = 2;
  string password = 3;
  string organization_id = 4;
  repeated string roles = 5;
}
```

**Response:**
```proto
message CreateUserResponse {
  User user = 1;
}
```

#### GetUser

Retrieves user information by ID.

**Request:**
```proto
message GetUserRequest {
  string id = 1;
}
```

**Response:**
```proto
message GetUserResponse {
  User user = 1;
}
```

#### UpdateUser

Updates user information.

**Request:**
```proto
message UpdateUserRequest {
  string id = 1;
  string name = 2;
  string email = 3;
  repeated string roles = 4;
  bool active = 5;
}
```

**Response:**
```proto
message UpdateUserResponse {
  User user = 1;
}
```

#### DeleteUser

Deletes a user account.

**Request:**
```proto
message DeleteUserRequest {
  string id = 1;
}
```

**Response:**
Empty response on success.

#### ListUsers

Lists users with optional filtering and pagination.

**Request:**
```proto
message ListUsersRequest {
  string organization_id = 1;
  plantd.common.v1.PageRequest page = 2;
}
```

**Response:**
Streaming response of `ListUsersResponse` messages:
```proto
message ListUsersResponse {
  User user = 1;
  plantd.common.v1.PageResponse page = 2;
}
```

### Data Models

#### User

```proto
message User {
  string id = 1;
  string email = 2;
  string name = 3;
  repeated string roles = 4;
  string organization_id = 5;
  plantd.common.v1.Metadata metadata = 6;
  google.protobuf.Timestamp last_login = 7;
  bool active = 8;
}
```

#### Organization

```proto
message Organization {
  string id = 1;
  string name = 2;
  string domain = 3;
  plantd.common.v1.Metadata metadata = 4;
}
```

### Organization Management Methods

#### CreateOrganization

Creates a new organization.

**Request:**
```proto
message CreateOrganizationRequest {
  string name = 1;
  string domain = 2;
}
```

**Response:**
```proto
message CreateOrganizationResponse {
  Organization organization = 1;
}
```

#### GetOrganization

Retrieves organization information by ID.

**Request:**
```proto
message GetOrganizationRequest {
  string id = 1;
}
```

**Response:**
```proto
message GetOrganizationResponse {
  Organization organization = 1;
}
```

### Role and Permission Management

#### AssignRole

Assigns a role to a user within an organization.

**Request:**
```proto
message AssignRoleRequest {
  string user_id = 1;
  string role = 2;
  string organization_id = 3;
}
```

**Response:**
Empty response on success.

#### RevokeRole

Revokes a role from a user within an organization.

**Request:**
```proto
message RevokeRoleRequest {
  string user_id = 1;
  string role = 2;
  string organization_id = 3;
}
```

**Response:**
Empty response on success.

#### CheckPermission

Checks if a user has a specific permission.

**Request:**
```proto
message CheckPermissionRequest {
  string user_id = 1;
  string permission = 2;
  string organization_id = 3;
}
```

**Response:**
```proto
message CheckPermissionResponse {
  bool allowed = 1;
  string reason = 2;
}
```

## State Service

The State service provides distributed key-value state management, replacing the traditional MDP command-based approach with a modern gRPC interface.

### Service Definition

```proto
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
  rpc Health(google.protobuf.Empty) returns (plantd.health.v1.HealthCheckResponse);
}
```

### MDP Command Mapping

The State service provides gRPC equivalents for traditional MDP commands:

| MDP Command | gRPC Method | Description |
|-------------|-------------|-------------|
| `get` | `Get` | Retrieve a value by key |
| `set` | `Set` | Store a key-value pair |
| `delete` | `Delete` | Remove a key |
| `list` | `List` | List keys with optional prefix |
| `heartbeat` | `Health` | Service health check |

### Basic Operations

#### Get

Retrieves a value by key.

**Request:**
```proto
message GetRequest {
  string key = 1;
  string scope = 2;  // Organization/user scope
  bool include_metadata = 3;
}
```

**Response:**
```proto
message GetResponse {
  string value = 1;
  plantd.common.v1.Metadata metadata = 2;
}
```

**Example:**
```bash
grpcurl -plaintext -d '{
  "key": "sensor/temperature",
  "scope": "plant-floor-1",
  "include_metadata": true
}' localhost:8080 plantd.state.v1.StateService/Get
```

#### Set

Stores a key-value pair.

**Request:**
```proto
message SetRequest {
  string key = 1;
  string value = 2;
  string scope = 3;
  bool create_only = 4;  // Fail if exists
  google.protobuf.Timestamp expires_at = 5;  // TTL support
}
```

**Response:**
```proto
message SetResponse {
  plantd.common.v1.Metadata metadata = 1;
}
```

**Example:**
```bash
grpcurl -plaintext -d '{
  "key": "sensor/temperature",
  "value": "23.5",
  "scope": "plant-floor-1"
}' localhost:8080 plantd.state.v1.StateService/Set
```

#### Delete

Removes a key from storage.

**Request:**
```proto
message DeleteRequest {
  string key = 1;
  string scope = 2;
}
```

**Response:**
```proto
message DeleteResponse {
  bool existed = 1;
  plantd.common.v1.Metadata last_metadata = 2;
}
```

### Bulk Operations

#### BatchGet

Retrieves multiple values in a single request.

**Request:**
```proto
message BatchGetRequest {
  repeated string keys = 1;
  string scope = 2;
}
```

**Response:**
```proto
message BatchGetResponse {
  map<string, GetResponse> results = 1;
  repeated string not_found = 2;
}
```

**Example:**
```bash
grpcurl -plaintext -d '{
  "keys": ["sensor/temperature", "sensor/pressure", "sensor/humidity"],
  "scope": "plant-floor-1"
}' localhost:8080 plantd.state.v1.StateService/BatchGet
```

#### BatchSet

Stores multiple key-value pairs in a single request.

**Request:**
```proto
message BatchSetRequest {
  message SetItem {
    string key = 1;
    string value = 2;
  }
  
  repeated SetItem items = 1;
  string scope = 2;
}
```

**Response:**
```proto
message BatchSetResponse {
  map<string, SetResponse> results = 1;
  repeated plantd.common.v1.Error errors = 2;
}
```

### Listing and Searching

#### List

Lists keys with optional prefix filtering.

**Request:**
```proto
message ListRequest {
  string prefix = 1;
  string scope = 2;
  plantd.common.v1.PageRequest page = 3;
  bool include_values = 4;
}
```

**Response:**
Streaming response of `ListResponse` messages:
```proto
message ListResponse {
  string key = 1;
  string value = 2;  // Only if include_values=true
  plantd.common.v1.Metadata metadata = 3;
  plantd.common.v1.PageResponse page = 4;
}
```

**Example:**
```bash
grpcurl -plaintext -d '{
  "prefix": "sensor/",
  "scope": "plant-floor-1",
  "include_values": true
}' localhost:8080 plantd.state.v1.StateService/List
```

#### Search

Searches for keys matching a pattern.

**Request:**
```proto
message SearchRequest {
  string query = 1;  // Search pattern
  string scope = 2;
  plantd.common.v1.PageRequest page = 3;
}
```

**Response:**
```proto
message SearchResponse {
  repeated ListResponse results = 1;
  plantd.common.v1.PageResponse page = 2;
}
```

### Real-time Updates

#### Watch

Watches for changes to keys matching a prefix.

**Request:**
```proto
message WatchRequest {
  string key_prefix = 1;
  string scope = 2;
  bool include_initial = 3;  // Send current values first
}
```

**Response:**
Streaming response of `WatchResponse` messages:
```proto
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
  plantd.common.v1.Metadata metadata = 4;
}
```

**Example:**
```bash
grpcurl -plaintext -d '{
  "key_prefix": "sensor/",
  "scope": "plant-floor-1",
  "include_initial": true
}' localhost:8080 plantd.state.v1.StateService/Watch
```

## Client Libraries

### Go Client

```go
import (
    "context"
    "google.golang.org/grpc"
    identityv1 "github.com/geoffjay/plantd/gen/proto/go/plantd/identity/v1"
    statev1 "github.com/geoffjay/plantd/gen/proto/go/plantd/state/v1"
    healthv1 "github.com/geoffjay/plantd/gen/proto/go/plantd/health/v1"
)

// Connect to services
conn, err := grpc.Dial("localhost:8080", grpc.WithInsecure())
if err != nil {
    log.Fatal(err)
}
defer conn.Close()

// Create service clients
identityClient := identityv1.NewIdentityServiceClient(conn)
stateClient := statev1.NewStateServiceClient(conn)
healthClient := healthv1.NewHealthServiceClient(conn)

// Use the clients
ctx := context.Background()
loginResp, err := identityClient.Login(ctx, &identityv1.LoginRequest{
    Email:    "user@example.com",
    Password: "password123",
})
```

### Authentication

All services except the Health service require authentication. Include the access token in the request metadata:

```go
import "google.golang.org/grpc/metadata"

// Add token to context
ctx := context.Background()
ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+accessToken)

// Use authenticated context
user, err := identityClient.GetUser(ctx, &identityv1.GetUserRequest{
    Id: "user-123",
})
```

## Error Handling

All services use standard gRPC status codes and include detailed error information:

```go
import "google.golang.org/grpc/status"

resp, err := client.SomeMethod(ctx, req)
if err != nil {
    if st, ok := status.FromError(err); ok {
        switch st.Code() {
        case codes.NotFound:
            // Handle not found
        case codes.PermissionDenied:
            // Handle permission denied
        case codes.InvalidArgument:
            // Handle invalid argument
        default:
            // Handle other errors
        }
    }
}
```

## Service Discovery

Services can be discovered using the Health service:

```bash
# Check if identity service is available
grpcurl -plaintext -d '{"service": "identity"}' localhost:8080 plantd.health.v1.HealthService/Check

# Watch for service status changes
grpcurl -plaintext -d '{"service": "identity"}' localhost:8080 plantd.health.v1.HealthService/Watch
```

## Testing

### grpcurl Examples

```bash
# Test health service
grpcurl -plaintext localhost:8080 list plantd.health.v1.HealthService

# Test identity service login
grpcurl -plaintext -d '{
  "email": "admin@example.com",
  "password": "admin123"
}' localhost:8080 plantd.identity.v1.IdentityService/Login

# Test state service
grpcurl -plaintext -d '{
  "key": "test-key",
  "value": "test-value"
}' localhost:8080 plantd.state.v1.StateService/Set
```

### Postman/Insomnia

Import the proto files to generate requests in Postman or Insomnia for GUI-based testing.

## Security

### Authentication

- All services except Health require JWT authentication
- Tokens are passed in the `authorization` metadata header
- Format: `Bearer <access_token>`

### Authorization

- The Identity service provides role-based access control
- Use `CheckPermission` to verify user permissions
- Scope permissions to organizations when applicable

### TLS

Production deployments should use TLS:

```go
conn, err := grpc.Dial("plantd.example.com:443", grpc.WithTransportCredentials(
    credentials.NewTLS(&tls.Config{}),
))
```

## Monitoring

### Health Checks

- All services implement the Health service interface
- Use `Health` method on individual services
- Use `HealthService/Check` for centralized health monitoring

### Metrics

Services expose metrics compatible with Prometheus:

```bash
curl http://localhost:8080/metrics
```

## Deployment

### Docker

```bash
# Build services
make build-all

# Run with Docker Compose
docker-compose up -d
```

### Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: plantd-identity
spec:
  replicas: 3
  selector:
    matchLabels:
      app: plantd-identity
  template:
    metadata:
      labels:
        app: plantd-identity
    spec:
      containers:
      - name: identity
        image: plantd/identity:latest
        ports:
        - containerPort: 8080
        env:
        - name: PLANTD_IDENTITY_GRPC_PORT
          value: "8080"
```

## Troubleshooting

### Common Issues

1. **Connection refused**: Check if the service is running on the expected port
2. **Permission denied**: Verify authentication token is valid and has required permissions
3. **Service unavailable**: Check service health using the Health service

### Debug Logging

Enable debug logging for detailed request/response information:

```bash
export GRPC_GO_LOG_VERBOSITY_LEVEL=99
export GRPC_GO_LOG_SEVERITY_LEVEL=info
```

### Service Status

Check service status using overmind:

```bash
overmind status
overmind restart identity
overmind echo identity
```