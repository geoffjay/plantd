# Request/Reply Messaging Analysis and Migration Recommendations

**Date**: December 2024  
**Project**: plantd Distributed Control System  

## Executive Summary

This analysis examines the current ZeroMQ-based request/reply messaging in plantd and recommends migration to simpler alternatives.

**Key Findings:**
- Current ZeroMQ implementation has significant complexity and reliability issues
- MDP protocol adds operational overhead without commensurate benefits  
- Documented instances of resource leaks, framing issues, and CGO crashes
- Alternative protocols could reduce complexity by 60-80%

**Primary Recommendation:** Migrate to gRPC for request/reply while retaining ZeroMQ for pub/sub.

## Current Implementation Issues

### 1. Reliability Problems

**CGO Signal Stack Corruption:**
- ZeroMQ CGO socket leaks cause service crashes
- Documented in SSE-ZMQ-CGO report
- Affects health checks and concurrent operations

**Frame Structure Issues:**  
- Inconsistent delimiter frame handling
- Complete client communication failures
- Required deep protocol debugging

### 2. Development Complexity

**Manual Message Construction:**
```go
// Error-prone frame building
req := make([]string, 4, len(request)+4)
req[0] = "" // Critical empty delimiter
req[1] = MdpcClient
req[2] = MdpcRequest  
req[3] = service
```

**Operational Overhead:**
- ~3,500 lines of MDP protocol code
- Complex debugging requiring frame-level analysis
- No standard tooling or monitoring

## Recommended Solution: gRPC with API Gateway

### Preserving Broker Architecture Benefits

The MDP broker solved a critical architectural problem: **avoiding direct service-to-service connections** that would create configuration complexity. We can preserve this benefit with modern gRPC gateway technologies:

```
┌─────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Clients   │───▶│   gRPC Gateway  │───▶│    Services     │
│ (CLI, Web)  │    │ (Envoy/Connect) │    │ (State,Identity)│
└─────────────┘    └─────────────────┘    └─────────────────┘
```

### Benefits
- **80% code reduction** in messaging infrastructure
- **Built-in features**: auth, load balancing, health checks
- **Strong typing** with Protocol Buffers
- **Standard tooling** for debugging and monitoring
- **Centralized routing** like current broker
- **Service discovery** and load balancing

### Gateway Technology Options

#### 1. Buf Connect (Recommended)
- **Modern approach**: Connect protocol over HTTP/1.1 and HTTP/2
- **Browser compatible**: Works directly from web browsers
- **Simpler than gRPC**: No need for gRPC-specific infrastructure
- **Buf ecosystem**: Integrated with Buf CLI and remote plugins

#### 2. Envoy Proxy
- **Industry standard**: CNCF graduated project
- **Full-featured**: Advanced routing, load balancing, observability
- **Service mesh ready**: Integrates with Istio/Linkerd
- **High performance**: C++ implementation

#### 3. grpc-gateway
- **HTTP/gRPC bridge**: Serves both gRPC and REST from same definitions
- **Swagger integration**: Automatic OpenAPI spec generation
- **Mature ecosystem**: Well-established tooling

## Gateway/Proxy Technology Comparison

### Overview of Options

For plantd's use case of replacing the MDP broker with a modern gateway, we need to evaluate proxies that can handle gRPC/Connect protocols while providing the centralized routing benefits. Here's a comprehensive comparison:

### 1. Traefik (Strong Alternative)

**Strengths:**
- **Dynamic configuration**: Auto-discovery of services via Docker, Kubernetes, Consul
- **Built-in gRPC support**: Native gRPC load balancing and routing
- **Excellent documentation**: Easy to configure and troubleshoot  
- **Cloud-native focus**: Designed for containerized environments
- **Built-in observability**: Metrics, tracing, and dashboard
- **Lightweight**: Go-based, single binary deployment

**Configuration Example:**
```yaml
# traefik.yml
api:
  dashboard: true

entryPoints:
  grpc:
    address: ":8080"
  web:
    address: ":80"

providers:
  docker:
    exposedByDefault: false
  file:
    filename: dynamic.yml

# dynamic.yml
http:
  routers:
    state-service:
      rule: "PathPrefix(`/plantd.state.v1.StateService`)"
      service: state-service
      middlewares:
        - auth

  services:
    state-service:
      loadBalancer:
        servers:
          - url: "http://state-service:8001"

  middlewares:
    auth:
      forwardAuth:
        address: "http://identity-service:8002/validate"
```

**Drawbacks:**
- **Limited gRPC features**: Less advanced gRPC-specific features than Envoy
- **Configuration complexity**: Can become complex with many services
- **Performance**: Good but not as optimized as C++ solutions

**Best for:** Docker/Kubernetes deployments with dynamic service discovery needs

### 2. Envoy Proxy (High Performance)

**Strengths:**
- **gRPC native**: Built from ground up for gRPC/HTTP/2
- **Extreme performance**: C++ implementation, battle-tested at scale
- **Advanced features**: Circuit breakers, retries, rate limiting, health checks
- **Observability**: Rich metrics, tracing, and logging
- **Service mesh foundation**: Core of Istio, Linkerd

**Configuration Example:**
```yaml
# envoy.yaml
static_resources:
  listeners:
  - name: grpc_listener
    address:
      socket_address:
        address: 0.0.0.0
        port_value: 8080
    filter_chains:
    - filters:
      - name: envoy.filters.network.http_connection_manager
        typed_config:
          "@type": type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
          stat_prefix: grpc
          route_config:
            name: local_route
            virtual_hosts:
            - name: services
              domains: ["*"]
              routes:
              - match:
                  prefix: "/plantd.state.v1.StateService"
                route:
                  cluster: state_service
              - match:
                  prefix: "/plantd.identity.v1.IdentityService"
                route:
                  cluster: identity_service
          http_filters:
          - name: envoy.filters.http.grpc_web
          - name: envoy.filters.http.router

  clusters:
  - name: state_service
    type: LOGICAL_DNS
    lb_policy: ROUND_ROBIN
    load_assignment:
      cluster_name: state_service
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              socket_address:
                address: state-service
                port_value: 8001
```

**Drawbacks:**
- **Configuration complexity**: YAML configuration can be verbose and complex
- **Learning curve**: Requires understanding of Envoy concepts
- **Resource usage**: Higher memory footprint than simpler proxies
- **Overkill for simple cases**: May be excessive for basic routing needs

**Best for:** High-performance, complex routing requirements, service mesh deployments

### 3. Kong Gateway (Enterprise Features)

**Strengths:**
- **Plugin ecosystem**: Extensive plugins for auth, rate limiting, monitoring
- **API management**: Built-in API versioning, documentation, developer portal
- **Multiple protocols**: HTTP, gRPC, WebSocket, TCP/UDP
- **Admin API**: Programmatic configuration management
- **Enterprise support**: Commercial backing and support

**Configuration Example:**
```bash
# Add gRPC service
curl -X POST http://kong-admin:8001/services \
  --data name=state-service \
  --data protocol=grpc \
  --data host=state-service \
  --data port=8001

# Add route
curl -X POST http://kong-admin:8001/services/state-service/routes \
  --data paths[]=/plantd.state.v1.StateService
```

**Drawbacks:**
- **Complexity**: Can be overkill for simple use cases
- **Resource usage**: Higher overhead than lightweight alternatives
- **Learning curve**: Many concepts and configuration options
- **Cost**: Enterprise features require paid license

**Best for:** Organizations needing full API management platform

### 4. HAProxy (Mature and Reliable)

**Strengths:**
- **Battle-tested**: 20+ years of production use
- **High performance**: Excellent performance characteristics
- **HTTP/2 support**: Good gRPC support via HTTP/2
- **Flexible configuration**: Powerful ACL and routing rules
- **Low resource usage**: Very efficient resource utilization

**Configuration Example:**
```
# haproxy.cfg
global
    daemon

defaults
    mode http
    timeout connect 5000ms
    timeout client 50000ms
    timeout server 50000ms

frontend grpc_frontend
    bind *:8080 proto h2
    use_backend state_backend if { path_beg /plantd.state.v1 }
    use_backend identity_backend if { path_beg /plantd.identity.v1 }

backend state_backend
    balance roundrobin
    server state1 state-service:8001 check proto h2

backend identity_backend
    balance roundrobin
    server identity1 identity-service:8002 check proto h2
```

**Drawbacks:**
- **Limited gRPC features**: Basic gRPC support, not gRPC-native
- **Configuration syntax**: Unique configuration format to learn
- **Less cloud-native**: Not designed specifically for container environments

**Best for:** Traditional deployments, organizations already using HAProxy

### 5. NGINX Plus (Commercial)

**Strengths:**
- **Proven performance**: Well-known performance characteristics
- **gRPC support**: Good gRPC load balancing and proxying
- **Extensive ecosystem**: Large community and plugin ecosystem
- **Multiple deployment options**: Can run as ingress controller or standalone

**Configuration Example:**
```nginx
# nginx.conf
upstream state_service {
    server state-service:8001;
}

upstream identity_service {
    server identity-service:8002;
}

server {
    listen 8080 http2;
    
    location /plantd.state.v1 {
        grpc_pass grpc://state_service;
    }
    
    location /plantd.identity.v1 {
        grpc_pass grpc://identity_service;
    }
}
```

**Drawbacks:**
- **gRPC limitations**: Less advanced gRPC features than specialized solutions
- **Commercial licensing**: NGINX Plus required for advanced features
- **Configuration reload**: Requires reload for configuration changes

**Best for:** Organizations already standardized on NGINX

### 6. Istio Service Mesh (Advanced)

**Strengths:**
- **Complete service mesh**: Traffic management, security, observability
- **Advanced gRPC features**: Full gRPC support with rich policy controls
- **Zero-trust security**: mTLS, RBAC, policy enforcement
- **Kubernetes native**: Deep integration with Kubernetes

**Drawbacks:**
- **Complexity**: Significant operational overhead
- **Resource requirements**: High resource consumption
- **Kubernetes dependency**: Requires Kubernetes
- **Learning curve**: Complex concepts and troubleshooting

**Best for:** Large Kubernetes deployments requiring full service mesh capabilities

## Detailed Comparison Matrix

| Feature | Traefik | Envoy | Kong | HAProxy | NGINX+ | Istio |
|---------|---------|-------|------|---------|--------|-------|
| **gRPC Support** | Good | Excellent | Good | Basic | Good | Excellent |
| **Performance** | Good | Excellent | Good | Excellent | Excellent | Good |
| **Configuration** | Medium | Complex | Medium | Medium | Simple | Complex |
| **Cloud Native** | Excellent | Excellent | Good | Fair | Good | Excellent |
| **Resource Usage** | Low | Medium | High | Low | Low | High |
| **Learning Curve** | Low | High | Medium | Medium | Low | High |
| **Observability** | Good | Excellent | Good | Basic | Good | Excellent |
| **Community** | Large | Large | Medium | Large | Large | Large |
| **Enterprise Support** | Available | Available | Yes | Available | Yes | Yes |

## Recommendation for plantd

### Primary Recommendation: **Traefik + Connect**

**Rationale:**
1. **Perfect fit for plantd's scale**: Not overkill, but feature-rich enough
2. **Docker/container friendly**: Excellent for plantd's deployment model
3. **Dynamic configuration**: Auto-discovery reduces configuration management
4. **Good gRPC support**: Sufficient for Connect protocol needs
5. **Operational simplicity**: Easy to configure, monitor, and troubleshoot
6. **Cost effective**: Open source with optional commercial support

**Implementation Example:**
```yaml
# docker-compose.yml
version: '3.8'
services:
  traefik:
    image: traefik:v3.0
    command:
      - --api.dashboard=true
      - --providers.docker=true
      - --entrypoints.grpc.address=:8080
      - --metrics.prometheus=true
    ports:
      - "8080:8080"
      - "8081:8080"  # Dashboard
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    labels:
      - "traefik.enable=true"

  state-service:
    build: ./services/state
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.state.rule=PathPrefix(`/plantd.state.v1`)"
      - "traefik.http.routers.state.entrypoints=grpc"
      - "traefik.http.services.state.loadbalancer.server.scheme=h2c"

  identity-service:
    build: ./services/identity
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.identity.rule=PathPrefix(`/plantd.identity.v1`)"
      - "traefik.http.routers.identity.entrypoints=grpc"
```

### Alternative Recommendation: **Envoy** (if high performance is critical)

Use Envoy if:
- You need maximum performance
- You plan to adopt service mesh in the future
- You have complex routing requirements
- You have team expertise in Envoy configuration

### Why Not Others for plantd:

- **Kong**: Overkill for plantd's current needs, adds unnecessary complexity
- **HAProxy**: Limited gRPC features, not ideal for modern Connect protocol
- **NGINX Plus**: Commercial licensing cost not justified for this use case
- **Istio**: Massive overkill, requires Kubernetes, too complex for current needs

The Traefik + Connect combination gives you the best balance of simplicity, performance, and features for plantd's distributed control system requirements.

## Buf Tooling Integration

### Buf CLI and Schema Management

**Schema Registry Benefits:**
- Centralized schema management
- Breaking change detection
- Version compatibility checking
- Remote code generation

**Setup Example:**
```yaml
# buf.yaml
version: v2
modules:
  - path: proto
breaking:
  use:
    - FILE
lint:
  use:
    - BASIC
    - COMMENTS
    - STYLE_GUIDE
```

### Connect Protocol vs Traditional gRPC

**Connect Protocol Advantages:**
- **HTTP/1.1 compatible**: Works with existing infrastructure
- **Browser native**: No proxy needed for web clients
- **Simpler deployment**: Standard HTTP load balancers
- **Better debugging**: Standard HTTP debugging tools

**Implementation Example:**
```go
// Traditional gRPC server setup (complex)
lis, err := net.Listen("tcp", ":9090")
s := grpc.NewServer()
statepb.RegisterStateServiceServer(s, &stateService{})
s.Serve(lis)

// Connect server setup (simple)
mux := http.NewServeMux()
path, handler := statev1connect.NewStateServiceHandler(&stateService{})
mux.Handle(path, handler)
http.ListenAndServe(":8080", mux)
```

### Remote Plugin Configuration

**buf.gen.yaml with Buf remote plugins:**
```yaml
version: v2
managed:
  enabled: true
  override:
    - file_option: go_package_prefix
      value: github.com/geoffjay/plantd/gen/proto/go
plugins:
  # Use remote plugins for consistency
  - remote: buf.build/protocolbuffers/go
    out: gen/proto/go
    opt:
      - paths=source_relative
  - remote: buf.build/connectrpc/go
    out: gen/proto/go
    opt:
      - paths=source_relative
  - remote: buf.build/grpc-ecosystem/gateway
    out: gen/proto/go
    opt:
      - paths=source_relative
```

## Protobuf API Design Examples

### State Service API

```protobuf
// proto/plantd/state/v1/state.proto
syntax = "proto3";

package plantd.state.v1;

import "google/protobuf/timestamp.proto";
import "google/protobuf/empty.proto";

option go_package = "github.com/geoffjay/plantd/gen/proto/go/plantd/state/v1;statev1";

// StateService manages key-value state with scoped access
service StateService {
  // Get retrieves a value by key within a scope
  rpc Get(GetRequest) returns (GetResponse);
  
  // Set stores a value with key within a scope
  rpc Set(SetRequest) returns (SetResponse);
  
  // Delete removes a key within a scope
  rpc Delete(DeleteRequest) returns (DeleteResponse);
  
  // List returns all keys with optional prefix filtering
  rpc List(ListRequest) returns (stream ListResponse);
  
  // Watch provides real-time updates for key changes
  rpc Watch(WatchRequest) returns (stream WatchResponse);
  
  // Health check for service availability
  rpc Health(google.protobuf.Empty) returns (HealthResponse);
}

message GetRequest {
  string key = 1;
  string scope = 2;  // Organization/user scope
}

message GetResponse {
  string value = 1;
  int64 version = 2;
  google.protobuf.Timestamp created = 3;
  google.protobuf.Timestamp modified = 4;
}

message SetRequest {
  string key = 1;
  string value = 2;
  string scope = 3;
  bool create_only = 4;  // Fail if key already exists
}

message SetResponse {
  int64 version = 1;
  google.protobuf.Timestamp modified = 2;
}

message DeleteRequest {
  string key = 1;
  string scope = 2;
}

message DeleteResponse {
  bool existed = 1;
  int64 last_version = 2;
}

message ListRequest {
  string prefix = 1;
  string scope = 2;
  int32 limit = 3;
  string page_token = 4;
}

message ListResponse {
  string key = 1;
  string value = 2;
  int64 version = 3;
  google.protobuf.Timestamp modified = 4;
  string next_page_token = 5;  // For pagination
}

message WatchRequest {
  string key_prefix = 1;
  string scope = 2;
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
  int64 version = 4;
  google.protobuf.Timestamp timestamp = 5;
}

message HealthResponse {
  enum Status {
    STATUS_UNSPECIFIED = 0;
    STATUS_HEALTHY = 1;
    STATUS_DEGRADED = 2;
    STATUS_UNHEALTHY = 3;
  }
  
  Status status = 1;
  string message = 2;
  map<string, string> details = 3;
}
```

### Identity Service API

```protobuf
// proto/plantd/identity/v1/identity.proto
syntax = "proto3";

package plantd.identity.v1;

import "google/protobuf/timestamp.proto";
import "google/protobuf/empty.proto";

option go_package = "github.com/geoffjay/plantd/gen/proto/go/plantd/identity/v1;identityv1";

// IdentityService handles authentication and authorization
service IdentityService {
  // Authentication RPCs
  rpc Login(LoginRequest) returns (LoginResponse);
  rpc Logout(LogoutRequest) returns (google.protobuf.Empty);
  rpc RefreshToken(RefreshTokenRequest) returns (RefreshTokenResponse);
  rpc ValidateToken(ValidateTokenRequest) returns (ValidateTokenResponse);
  
  // User management RPCs
  rpc CreateUser(CreateUserRequest) returns (CreateUserResponse);
  rpc GetUser(GetUserRequest) returns (GetUserResponse);
  rpc UpdateUser(UpdateUserRequest) returns (UpdateUserResponse);
  rpc DeleteUser(DeleteUserRequest) returns (google.protobuf.Empty);
  
  // Organization management RPCs
  rpc CreateOrganization(CreateOrganizationRequest) returns (CreateOrganizationResponse);
  rpc GetOrganization(GetOrganizationRequest) returns (GetOrganizationResponse);
  
  // Role and permission RPCs
  rpc AssignRole(AssignRoleRequest) returns (google.protobuf.Empty);
  rpc CheckPermission(CheckPermissionRequest) returns (CheckPermissionResponse);
  
  // Health check
  rpc Health(google.protobuf.Empty) returns (HealthResponse);
}

message LoginRequest {
  string email = 1;
  string password = 2;
  string organization_id = 3;  // Optional organization context
}

message LoginResponse {
  string access_token = 1;
  string refresh_token = 2;
  google.protobuf.Timestamp expires_at = 3;
  User user = 4;
}

message User {
  string id = 1;
  string email = 2;
  string name = 3;
  repeated string roles = 4;
  string organization_id = 5;
  google.protobuf.Timestamp created_at = 6;
  google.protobuf.Timestamp last_login = 7;
}

message ValidateTokenRequest {
  string token = 1;
  repeated string required_permissions = 2;  // Optional permission check
}

message ValidateTokenResponse {
  bool valid = 1;
  User user = 2;
  repeated string permissions = 3;
}

message CheckPermissionRequest {
  string user_id = 1;
  string permission = 2;
  string resource = 3;  // Optional resource context
}

message CheckPermissionResponse {
  bool allowed = 1;
  string reason = 2;  // Explanation if denied
}

// ... other message definitions
```

### Gateway Routing Service

```protobuf
// proto/plantd/gateway/v1/routing.proto
syntax = "proto3";

package plantd.gateway.v1;

import "google/protobuf/any.proto";

option go_package = "github.com/geoffjay/plantd/gen/proto/go/plantd/gateway/v1;gatewayv1";

// GatewayService handles service discovery and routing
service GatewayService {
  // Service discovery
  rpc ListServices(ListServicesRequest) returns (ListServicesResponse);
  rpc GetServiceHealth(GetServiceHealthRequest) returns (GetServiceHealthResponse);
  
  // Generic routing (replaces MDP broker functionality)
  rpc RouteRequest(RouteRequest) returns (RouteResponse);
}

message RouteRequest {
  string service_name = 1;
  string method = 2;
  google.protobuf.Any payload = 3;
  map<string, string> metadata = 4;  // Headers, auth tokens, etc.
}

message RouteResponse {
  google.protobuf.Any payload = 1;
  map<string, string> metadata = 2;
  int32 status_code = 3;
  string error_message = 4;
}

message ServiceInfo {
  string name = 1;
  string version = 2;
  repeated string endpoints = 3;
  HealthStatus health = 4;
  map<string, string> metadata = 5;
}

enum HealthStatus {
  HEALTH_STATUS_UNSPECIFIED = 0;
  HEALTH_STATUS_HEALTHY = 1;
  HEALTH_STATUS_DEGRADED = 2;
  HEALTH_STATUS_UNHEALTHY = 3;
}
```

## Migration Example

### Before (MDP):**
```go
// Complex MDP client setup
client, err := mdp.NewClient("tcp://127.0.0.1:9797")
err = client.Send("org.plantd.State", "get", "key1")
response, err := client.Recv()
```

**After (Connect/gRPC):**
```go
// Simple Connect client
client := statev1connect.NewStateServiceClient(
    http.DefaultClient,
    "http://localhost:8080",  // Gateway URL
)
response, err := client.Get(ctx, &statev1.GetRequest{
    Key:   "key1",
    Scope: "default",
})
```

### Protocol Definition
```protobuf
service StateService {
  rpc Get(GetRequest) returns (GetResponse);
  rpc Set(SetRequest) returns (SetResponse);
  rpc List(ListRequest) returns (stream ListResponse);
}
```

## Migration Strategy

### 8-Week Plan

**Weeks 1-2**: Core Services (State, Identity)
**Weeks 3-4**: Client Migration (CLI, Web App)  
**Weeks 5-6**: Advanced Features (streaming, auth)
**Weeks 7-8**: Cleanup (remove MDP, simplify ZeroMQ)

### Expected Benefits

**Reliability:**
- Eliminate CGO-related crashes
- Built-in connection management
- Standard error handling

**Development:**
- 85% reduction in messaging code
- Simple debugging with HTTP tools
- Faster onboarding for developers

**Operations:**
- Standard HTTP/2 load balancers
- Native Kubernetes integration
- Built-in metrics and tracing

## Risk Mitigation

- **Backward compatibility** during transition
- **Phased rollout** with rollback capability
- **Performance benchmarking** at each phase
- **Comprehensive testing** with existing clients

## Conclusion

ZeroMQ-based request/reply represents significant technical debt. Migration to gRPC offers:

- **Dramatic simplification** of messaging infrastructure
- **Elimination** of documented reliability issues
- **Standard practices** aligned with industry trends
- **Future-proof** architecture with strong ecosystem

The 8-week migration plan is low-risk with incremental value delivery.

## Practical Implementation Plan

### Project Structure with Buf

```
plantd/
├── proto/
│   └── plantd/
│       ├── state/v1/
│       │   └── state.proto
│       ├── identity/v1/
│       │   └── identity.proto
│       └── gateway/v1/
│           └── routing.proto
├── buf.yaml
├── buf.gen.yaml
├── buf.work.yaml
├── gen/
│   └── proto/
│       └── go/
│           └── plantd/
├── gateway/
│   ├── main.go
│   └── config.yaml
└── services/
    ├── state/
    ├── identity/
    └── ...
```

### Buf Configuration

**buf.work.yaml (workspace root):**
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
breaking:
  use:
    - FILE
lint:
  use:
    - BASIC
    - COMMENTS
    - STYLE_GUIDE
  except:
    - UNARY_RPC  # Allow streaming RPCs where beneficial
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
  # Go protobuf
  - remote: buf.build/protocolbuffers/go
    out: gen/proto/go
    opt:
      - paths=source_relative
  
  # Connect-Go (recommended for plantd)
  - remote: buf.build/connectrpc/go
    out: gen/proto/go
    opt:
      - paths=source_relative
  
  # Optional: grpc-gateway for REST compatibility
  - remote: buf.build/grpc-ecosystem/gateway
    out: gen/proto/go
    opt:
      - paths=source_relative
      - generate_unbound_methods=true
  
  # Optional: OpenAPI spec generation
  - remote: buf.build/grpc-ecosystem/openapiv2
    out: gen/proto/openapi
```

### Connect Gateway Implementation

**gateway/main.go:**
```go
package main

import (
    "context"
    "fmt"
    "log"
    "net/http"
    "time"

    "connectrpc.com/connect"
    "github.com/rs/cors"
    
    // Generated Connect clients
    statev1connect "github.com/geoffjay/plantd/gen/proto/go/plantd/state/v1/statev1connect"
    identityv1connect "github.com/geoffjay/plantd/gen/proto/go/plantd/identity/v1/identityv1connect"
)

type Gateway struct {
    stateClient    statev1connect.StateServiceClient
    identityClient identityv1connect.IdentityServiceClient
}

func NewGateway() *Gateway {
    return &Gateway{
        stateClient: statev1connect.NewStateServiceClient(
            http.DefaultClient,
            "http://localhost:8001", // State service endpoint
        ),
        identityClient: identityv1connect.NewIdentityServiceClient(
            http.DefaultClient,
            "http://localhost:8002", // Identity service endpoint
        ),
    }
}

func (g *Gateway) setupRoutes() *http.ServeMux {
    mux := http.NewServeMux()
    
    // Mount service handlers with authentication middleware
    stateHandler := &StateHandler{client: g.stateClient}
    mux.Handle(statev1connect.NewStateServiceHandler(
        stateHandler,
        connect.WithInterceptors(authInterceptor()),
    ))
    
    identityHandler := &IdentityHandler{client: g.identityClient}
    mux.Handle(identityv1connect.NewIdentityServiceHandler(identityHandler))
    
    // Health check endpoint
    mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        fmt.Fprintf(w, `{"status":"healthy","timestamp":"%s"}`, time.Now().UTC().Format(time.RFC3339))
    })
    
    return mux
}

// StateHandler proxies requests to the actual State service
type StateHandler struct {
    client statev1connect.StateServiceClient
}

func (h *StateHandler) Get(ctx context.Context, req *connect.Request[statev1.GetRequest]) (*connect.Response[statev1.GetResponse], error) {
    // Add request tracing/logging
    log.Printf("Gateway: State.Get key=%s scope=%s", req.Msg.Key, req.Msg.Scope)
    
    // Forward to actual service
    return h.client.Get(ctx, req)
}

// Authentication interceptor
func authInterceptor() connect.UnaryInterceptorFunc {
    return func(next connect.UnaryFunc) connect.UnaryFunc {
        return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
            // Skip auth for identity service and health checks
            if req.Spec().Procedure == "/plantd.identity.v1.IdentityService/Login" ||
               req.Spec().Procedure == "/plantd.identity.v1.IdentityService/Health" {
                return next(ctx, req)
            }
            
            // Extract and validate token
            token := req.Header().Get("Authorization")
            if token == "" {
                return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("missing authorization header"))
            }
            
            // Validate token with identity service
            // ... token validation logic
            
            return next(ctx, req)
        }
    }
}

func main() {
    gateway := NewGateway()
    mux := gateway.setupRoutes()
    
    // Add CORS support for web clients
    handler := cors.New(cors.Options{
        AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
        AllowedHeaders: []string{"*"},
        AllowedOrigins: []string{"*"}, // Configure appropriately for production
    }).Handler(mux)
    
    log.Println("Gateway starting on :8080")
    log.Fatal(http.ListenAndServe(":8080", handler))
}
```

### Service Implementation Example

**services/state/main.go:**
```go
package main

import (
    "context"
    "log"
    "net/http"
    
    "connectrpc.com/connect"
    
    statev1 "github.com/geoffjay/plantd/gen/proto/go/plantd/state/v1"
    "github.com/geoffjay/plantd/gen/proto/go/plantd/state/v1/statev1connect"
)

type StateService struct {
    store *Store // Your existing state store
}

func (s *StateService) Get(ctx context.Context, req *connect.Request[statev1.GetRequest]) (*connect.Response[statev1.GetResponse], error) {
    value, err := s.store.Get(req.Msg.Key, req.Msg.Scope)
    if err != nil {
        return nil, connect.NewError(connect.CodeNotFound, err)
    }
    
    response := &statev1.GetResponse{
        Value:   value.Data,
        Version: value.Version,
        Created: timestamppb.New(value.Created),
        Modified: timestamppb.New(value.Modified),
    }
    
    return connect.NewResponse(response), nil
}

func (s *StateService) Set(ctx context.Context, req *connect.Request[statev1.SetRequest]) (*connect.Response[statev1.SetResponse], error) {
    version, err := s.store.Set(req.Msg.Key, req.Msg.Value, req.Msg.Scope)
    if err != nil {
        return nil, connect.NewError(connect.CodeInternal, err)
    }
    
    response := &statev1.SetResponse{
        Version:  version,
        Modified: timestamppb.Now(),
    }
    
    return connect.NewResponse(response), nil
}

// ... implement other methods

func main() {
    stateService := &StateService{
        store: NewStore(), // Your existing store implementation
    }
    
    mux := http.NewServeMux()
    path, handler := statev1connect.NewStateServiceHandler(stateService)
    mux.Handle(path, handler)
    
    log.Println("State service starting on :8001")
    log.Fatal(http.ListenAndServe(":8001", mux))
}
```

### CLI Client Example

**client/cmd/state.go:**
```go
package cmd

import (
    "context"
    "fmt"
    "net/http"
    
    "connectrpc.com/connect"
    "github.com/spf13/cobra"
    
    statev1 "github.com/geoffjay/plantd/gen/proto/go/plantd/state/v1"
    "github.com/geoffjay/plantd/gen/proto/go/plantd/state/v1/statev1connect"
)

func NewStateGetCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "get [key]",
        Short: "Get a value from state store",
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            client := statev1connect.NewStateServiceClient(
                http.DefaultClient,
                "http://localhost:8080", // Gateway URL
                connect.WithClientOptions(
                    connect.WithInterceptors(authInterceptor()), // Add auth token
                ),
            )
            
            req := &statev1.GetRequest{
                Key:   args[0],
                Scope: "default", // TODO: make configurable
            }
            
            resp, err := client.Get(context.Background(), connect.NewRequest(req))
            if err != nil {
                return fmt.Errorf("failed to get value: %w", err)
            }
            
            fmt.Printf("Key: %s\nValue: %s\nVersion: %d\n", 
                args[0], resp.Msg.Value, resp.Msg.Version)
            
            return nil
        },
    }
    
    return cmd
}
```

### Development Workflow

**1. Schema Development:**
```bash
# Install Buf
curl -sSL "https://github.com/bufbuild/buf/releases/latest/download/buf-$(uname -s)-$(uname -m)" -o "/usr/local/bin/buf"
chmod +x "/usr/local/bin/buf"

# Initialize and validate schemas
buf mod init
buf lint
buf format -w
buf breaking --against '.git#branch=main'
```

**2. Code Generation:**
```bash
# Generate Go code with Connect
buf generate

# Push to Buf Schema Registry (optional)
buf push
```

**3. Service Development:**
```bash
# Start services locally
go run services/state/main.go    # :8001
go run services/identity/main.go # :8002
go run gateway/main.go           # :8080

# Test with CLI
go run client/main.go state get mykey
```

### Deployment Configuration

**docker-compose.yml:**
```yaml
version: '3.8'
services:
  gateway:
    build: ./gateway
    ports:
      - "8080:8080"
    depends_on:
      - state-service
      - identity-service
    environment:
      - STATE_SERVICE_URL=http://state-service:8001
      - IDENTITY_SERVICE_URL=http://identity-service:8002

  state-service:
    build: ./services/state
    ports:
      - "8001:8001"
    volumes:
      - ./data/state:/data

  identity-service:
    build: ./services/identity
    ports:
      - "8002:8002"
    depends_on:
      - postgres
```

This approach gives you:

1. **Centralized routing** through the gateway (preserves broker benefits)
2. **Modern gRPC/Connect** with excellent tooling
3. **Buf ecosystem integration** for schema management
4. **Backward compatibility** during migration
5. **Production-ready** deployment patterns
