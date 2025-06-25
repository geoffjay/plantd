# MDP to Traefik + gRPC Migration Plan

**Timeline**: 8 weeks | **Status**: Ready for execution

## Phase 1: Foundation (Week 1)

### 1.1 Buf Tooling Setup
- Install Buf CLI and configure workspace
- Create proto directory structure
- Setup buf.yaml, buf.gen.yaml, buf.work.yaml
- Configure CI/CD integration

### 1.2 Common Protocols
- Define shared types (common.proto)
- Health check protocol (health.proto)
- Error handling patterns
- Generate initial Go code

**Deliverables**: Working Buf setup, common protocols

## Phase 2: Service Protocols (Week 2)

### 2.1 State Service Protocol
```protobuf
service StateService {
  rpc Get(GetRequest) returns (GetResponse);
  rpc Set(SetRequest) returns (SetResponse);
  rpc Delete(DeleteRequest) returns (DeleteResponse);
  rpc List(ListRequest) returns (stream ListResponse);
  rpc Watch(WatchRequest) returns (stream WatchResponse);
  rpc Health(Empty) returns (HealthResponse);
}
```

### 2.2 Identity Service Protocol
```protobuf
service IdentityService {
  rpc Login(LoginRequest) returns (LoginResponse);
  rpc ValidateToken(ValidateTokenRequest) returns (ValidateTokenResponse);
  rpc CreateUser(CreateUserRequest) returns (CreateUserResponse);
  // ... other methods
}
```

**Deliverables**: Complete protobuf definitions, generated Go code

## Phase 3: Service Implementation (Week 3)

### 3.1 gRPC Server Implementation
- State service Connect server
- Identity service Connect server  
- Health check implementation
- MDP compatibility layer for gradual migration

### 3.2 Testing
- Unit tests for gRPC services
- Integration tests
- MDP compatibility tests

**Deliverables**: Working gRPC services with MDP compatibility

## Phase 4: Traefik Gateway (Week 4)

### 4.1 Traefik Configuration
```yaml
# traefik.yml
entryPoints:
  grpc:
    address: ":8080"

providers:
  docker:
    exposedByDefault: false

http:
  routers:
    state-service:
      rule: "PathPrefix(`/plantd.state.v1`)"
      service: state-service
      middlewares: [auth, retry]
```

### 4.2 Docker Setup
- Service Dockerfiles
- docker-compose.yml with Traefik
- Development scripts
- Health check integration

**Deliverables**: Working Traefik gateway routing to gRPC services

## Phase 5: Client Migration (Week 5)

### 5.1 CLI Client Update
```go
// Before (MDP)
client, err := mdp.NewClient("tcp://127.0.0.1:9797")
err = client.Send("org.plantd.State", "get", "key1")

// After (Connect)
client := statev1connect.NewStateServiceClient(
    http.DefaultClient, "http://localhost:8080")
resp, err := client.Get(ctx, &statev1.GetRequest{Key: "key1"})
```

### 5.2 Web App Integration
- Update service calls to use gRPC gateway
- Authentication header handling
- Error handling updates

**Deliverables**: Updated CLI and web app using gRPC

## Phase 6: Integration Testing (Week 6)

### 6.1 End-to-End Testing
- Full system tests with Traefik
- Performance benchmarking
- Load testing
- Failure scenario testing

### 6.2 Migration Testing
- Parallel MDP/gRPC operation
- Data consistency validation
- Rollback procedures

**Deliverables**: Comprehensive test suite, performance validation

## Phase 7: Production Deployment (Week 7)

### 7.1 Production Configuration
```yaml
# traefik.prod.yml
certificatesResolvers:
  letsencrypt:
    acme:
      email: admin@plantd.com
      storage: /letsencrypt/acme.json

http:
  middlewares:
    secure-headers:
      headers:
        sslRedirect: true
        forceSTSHeader: true
```

### 7.2 Deployment Strategy
- Blue/green deployment setup
- Monitoring and alerting
- Backup and rollback procedures
- Security configuration

**Deliverables**: Production-ready deployment

## Phase 8: Migration & Documentation (Week 8)

### 8.1 Live Migration
- Gradual traffic migration from MDP to gRPC
- Monitor performance and errors
- Complete MDP shutdown
- Cleanup old code

### 8.2 Documentation
- API documentation
- Deployment guides
- Troubleshooting guides
- Architecture decision records

**Deliverables**: Complete migration, comprehensive documentation

## Key Implementation Files

### Protocol Definitions
```
proto/plantd/
├── common/v1/common.proto      # Shared types
├── health/v1/health.proto      # Health checks
├── state/v1/state.proto        # State service
└── identity/v1/identity.proto  # Identity service
```

### Service Implementation
```
state/grpc_server.go           # State gRPC server
identity/grpc_server.go        # Identity gRPC server
gateway/main.go                # Optional custom gateway
```

### Configuration
```
traefik/traefik.yml           # Traefik config
docker-compose.plantd.yml     # Docker setup
buf.gen.yaml                  # Code generation
```

### Client Updates
```
client/cmd/state.go           # CLI state commands
app/internal/services/        # Web app service clients
```

## Migration Commands

```bash
# Week 1: Setup
make proto-setup
make proto-gen

# Week 2-3: Development
make build-grpc-services
make test-grpc

# Week 4: Gateway
make traefik-dev-start
make test-gateway

# Week 5: Clients
make build-client
make test-e2e

# Week 6-7: Production
make deploy-staging
make deploy-prod

# Week 8: Migration
make migrate-traffic
make cleanup-mdp
```

## Success Metrics

- **Code Reduction**: 85% reduction in messaging code
- **Reliability**: Zero CGO-related crashes
- **Performance**: <20ms p95 latency for local requests
- **Deployment**: <5 minute deployment time
- **Monitoring**: Full observability with Prometheus/Jaeger

## Risk Mitigation

- **Backward Compatibility**: MDP compatibility layer during transition
- **Rollback Plan**: Keep MDP broker running until migration complete
- **Testing**: Comprehensive integration and load testing
- **Monitoring**: Detailed metrics and alerting throughout migration

This plan provides a clear, actionable roadmap for migrating from MDP to Traefik + gRPC while maintaining system reliability and minimizing downtime. 
