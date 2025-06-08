[![Go Report Card](https://goreportcard.com/badge/github.com/geoffjay/plantd/proxy)](https://goreportcard.com/report/github.com/geoffjay/plantd/proxy)

---

# 📨 Proxy Service

The proxy service acts as a protocol gateway and API gateway for the plantd distributed control system. It translates between different communication protocols, making it easier for external clients to interact with plantd services without needing to understand ZeroMQ or the Majordomo Protocol.

## Features

- **Protocol Translation**: Convert between HTTP/REST, GraphQL, gRPC, and ZeroMQ
- **API Gateway**: Centralized entry point for all external API requests
- **Service Discovery**: Automatic discovery and routing to available services
- **Load Balancing**: Distribute requests across multiple service instances
- **Authentication**: Integrate with identity service for request authentication
- **Rate Limiting**: Protect backend services from excessive requests
- **Request/Response Transformation**: Modify requests and responses as needed
- **Caching**: Cache frequently requested data to improve performance

## Quick Start

### Prerequisites

- Go 1.24 or later
- ZeroMQ library (for backend communication)
- Access to plantd broker service

### Installation

```bash
# Build the proxy service
make build-proxy

# Or build directly
cd proxy
go build -o proxy main.go
```

### Basic Usage

```bash
# Start with default configuration
./build/plantd-proxy

# Start with custom port
PLANTD_PROXY_PORT=8080 ./build/plantd-proxy

# Start with debug logging
PLANTD_PROXY_LOG_LEVEL=debug ./build/plantd-proxy

# Start with custom broker endpoint
PLANTD_PROXY_BROKER_ENDPOINT="tcp://localhost:7200" ./build/plantd-proxy
```

## Configuration

### Environment Variables

```bash
# Server configuration
export PLANTD_PROXY_PORT="5000"
export PLANTD_PROXY_ADDRESS="0.0.0.0"

# Broker configuration
export PLANTD_PROXY_BROKER_ENDPOINT="tcp://localhost:7200"

# Logging configuration
export PLANTD_PROXY_LOG_LEVEL="info"
export PLANTD_PROXY_LOG_FORMAT="text"

# Authentication
export PLANTD_PROXY_AUTH_ENABLED="true"
export PLANTD_PROXY_IDENTITY_URL="http://localhost:8080"

# Rate limiting
export PLANTD_PROXY_RATE_LIMIT="100"
export PLANTD_PROXY_RATE_WINDOW="1m"

# Caching
export PLANTD_PROXY_CACHE_ENABLED="true"
export PLANTD_PROXY_CACHE_TTL="5m"
```

### Configuration File

```yaml
# config/proxy.yaml
server:
  port: 5000
  address: "0.0.0.0"
  read_timeout: "30s"
  write_timeout: "30s"

broker:
  endpoint: "tcp://localhost:7200"
  timeout: "30s"
  retries: 3

authentication:
  enabled: true
  identity_url: "http://localhost:8080"
  bypass_paths:
    - "/health"
    - "/metrics"
    - "/docs"

rate_limiting:
  enabled: true
  requests_per_minute: 100
  burst_size: 20
  
caching:
  enabled: true
  default_ttl: "5m"
  max_size: "100MB"
  
cors:
  enabled: true
  allowed_origins: ["*"]
  allowed_methods: ["GET", "POST", "PUT", "DELETE"]
  allowed_headers: ["Content-Type", "Authorization"]

logging:
  level: "info"
  format: "json"
```

## API Endpoints

### Service Proxy

The proxy automatically creates REST endpoints for all discovered services:

```bash
# Generic service call
POST /api/v1/services/{service_name}/{operation}
{
  "data": "request_payload"
}

# Examples:
# Call state service
POST /api/v1/services/org.plantd.State/get
{
  "service": "org.plantd.MyService",
  "key": "temperature"
}

# Call broker service
POST /api/v1/services/org.plantd.Broker/services
{}

# Call echo service
POST /api/v1/services/org.plantd.Echo/echo
{
  "message": "hello world"
}
```

### Service Discovery

```bash
# List available services
GET /api/v1/services

# Response:
{
  "services": [
    {
      "name": "org.plantd.State",
      "status": "active",
      "workers": 2,
      "last_seen": "2024-01-01T12:00:00Z"
    },
    {
      "name": "org.plantd.Broker", 
      "status": "active",
      "workers": 1,
      "last_seen": "2024-01-01T12:00:00Z"
    }
  ]
}

# Get service details
GET /api/v1/services/org.plantd.State

# Response:
{
  "name": "org.plantd.State",
  "status": "active",
  "workers": 2,
  "operations": ["get", "set", "delete", "list"],
  "last_seen": "2024-01-01T12:00:00Z",
  "metadata": {
    "version": "1.0.0",
    "description": "Distributed state management"
  }
}
```

### Health and Status

```bash
# Health check
GET /health
# Response: {"status": "healthy", "timestamp": "2024-01-01T12:00:00Z"}

# Detailed status
GET /status
# Response: {
#   "status": "healthy",
#   "uptime": "2h30m",
#   "requests_handled": 1250,
#   "active_connections": 15,
#   "broker_status": "connected",
#   "services_discovered": 5
# }

# Metrics
GET /metrics
# Response: Prometheus-style metrics
```

## Protocol Support

### REST API

Standard HTTP/REST interface:

```bash
# GET request
curl http://localhost:5000/api/v1/services/org.plantd.State/get?service=test&key=value

# POST request with JSON
curl -X POST http://localhost:5000/api/v1/services/org.plantd.State/set \
  -H "Content-Type: application/json" \
  -d '{"service": "test", "key": "temperature", "value": "23.5"}'

# With authentication
curl -X POST http://localhost:5000/api/v1/services/org.plantd.State/get \
  -H "Authorization: Bearer your-jwt-token" \
  -H "Content-Type: application/json" \
  -d '{"service": "test", "key": "temperature"}'
```

### GraphQL (Planned)

GraphQL interface for complex queries:

```graphql
# Query multiple services
query {
  state(service: "org.plantd.TempSensor") {
    get(key: "temperature")
    get(key: "humidity")
  }
  
  broker {
    services {
      name
      status
      workers
    }
  }
}

# Mutation for state changes
mutation {
  setState(service: "org.plantd.TempSensor", key: "setpoint", value: "25.0") {
    success
    message
  }
}
```

### gRPC (Planned)

Protocol buffer definitions for type-safe communication:

```protobuf
// plantd.proto
syntax = "proto3";

package plantd;

service PlantdProxy {
  rpc CallService(ServiceRequest) returns (ServiceResponse);
  rpc ListServices(Empty) returns (ServiceList);
  rpc GetServiceStatus(ServiceName) returns (ServiceStatus);
}

message ServiceRequest {
  string service_name = 1;
  string operation = 2;
  bytes payload = 3;
}

message ServiceResponse {
  bool success = 1;
  bytes data = 2;
  string error = 3;
}
```

## Authentication Integration

### JWT Token Validation

```go
// Example authentication middleware
func authMiddleware(identityURL string) fiber.Handler {
    return func(c *fiber.Ctx) error {
        // Skip authentication for public endpoints
        if isPublicEndpoint(c.Path()) {
            return c.Next()
        }
        
        token := extractToken(c)
        if token == "" {
            return c.Status(401).JSON(fiber.Map{
                "error": "Missing authentication token"
            })
        }
        
        // Validate token with identity service
        user, err := validateToken(identityURL, token)
        if err != nil {
            return c.Status(401).JSON(fiber.Map{
                "error": "Invalid authentication token"
            })
        }
        
        // Store user context
        c.Locals("user", user)
        return c.Next()
    }
}
```

### Permission Checking

```go
// Example permission middleware
func requirePermission(permission string) fiber.Handler {
    return func(c *fiber.Ctx) error {
        user := c.Locals("user").(User)
        
        if !user.HasPermission(permission) {
            return c.Status(403).JSON(fiber.Map{
                "error": "Insufficient permissions"
            })
        }
        
        return c.Next()
    }
}

// Usage in routes
app.Post("/api/v1/services/org.plantd.State/set", 
    requirePermission("state:write"), 
    handleStateSet)
```

## Load Balancing

### Service Instance Selection

```go
// Example load balancing strategies
type LoadBalancer interface {
    SelectWorker(service string, workers []Worker) Worker
}

// Round-robin load balancing
type RoundRobinBalancer struct {
    counters map[string]int
}

func (lb *RoundRobinBalancer) SelectWorker(service string, workers []Worker) Worker {
    if len(workers) == 0 {
        return nil
    }
    
    counter := lb.counters[service]
    worker := workers[counter%len(workers)]
    lb.counters[service] = counter + 1
    
    return worker
}

// Least connections load balancing
type LeastConnectionsBalancer struct{}

func (lb *LeastConnectionsBalancer) SelectWorker(service string, workers []Worker) Worker {
    if len(workers) == 0 {
        return nil
    }
    
    minConnections := workers[0].Connections()
    selectedWorker := workers[0]
    
    for _, worker := range workers[1:] {
        if worker.Connections() < minConnections {
            minConnections = worker.Connections()
            selectedWorker = worker
        }
    }
    
    return selectedWorker
}
```

## Caching

### Response Caching

```go
// Example caching middleware
func cacheMiddleware(cache Cache, ttl time.Duration) fiber.Handler {
    return func(c *fiber.Ctx) error {
        // Generate cache key
        key := generateCacheKey(c.Method(), c.Path(), c.Body())
        
        // Check cache
        if cached, found := cache.Get(key); found {
            return c.JSON(cached)
        }
        
        // Continue to handler
        if err := c.Next(); err != nil {
            return err
        }
        
        // Cache response if successful
        if c.Response().StatusCode() == 200 {
            cache.Set(key, c.Response().Body(), ttl)
        }
        
        return nil
    }
}
```

### Cache Configuration

```yaml
caching:
  enabled: true
  default_ttl: "5m"
  max_size: "100MB"
  
  # Per-service cache settings
  services:
    "org.plantd.State":
      ttl: "1m"
      operations:
        "get": "5m"
        "list": "30s"
    "org.plantd.Metrics":
      ttl: "10s"
```

## Monitoring

### Metrics

The proxy service exposes comprehensive metrics:

- **Request Count**: Total requests processed
- **Response Time**: Request processing latency
- **Error Rate**: Failed requests per second
- **Service Availability**: Backend service health
- **Cache Hit Rate**: Cache effectiveness
- **Connection Pool**: Backend connection statistics

### Logging

Structured request/response logging:

```json
{
  "timestamp": "2024-01-01T12:00:00Z",
  "level": "info",
  "message": "Request processed",
  "request_id": "req-123",
  "method": "POST",
  "path": "/api/v1/services/org.plantd.State/get",
  "user_id": "user-456",
  "service": "org.plantd.State",
  "operation": "get",
  "duration_ms": 15.3,
  "status_code": 200,
  "cache_hit": false
}
```

## Development

### Hot Reload

```bash
# Install Air for hot reload
go install github.com/cosmtrek/air@latest

# Start with hot reload
air
```

### Testing

```bash
# Run unit tests
go test ./...

# Run integration tests (requires broker)
go test -tags=integration ./...

# Load testing
go test -bench=. ./...
```

### Adding New Protocol Support

1. Create protocol handler in `handlers/`
2. Implement request/response translation
3. Register routes in `routes.go`
4. Add configuration options
5. Update documentation

## Deployment

### Docker

```bash
# Build Docker image
docker build -t plantd-proxy .

# Run with Docker
docker run -p 5000:5000 \
  -e PLANTD_PROXY_BROKER_ENDPOINT="tcp://broker:7200" \
  plantd-proxy
```

### Kubernetes

```yaml
# k8s-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: plantd-proxy
spec:
  replicas: 3
  selector:
    matchLabels:
      app: plantd-proxy
  template:
    metadata:
      labels:
        app: plantd-proxy
    spec:
      containers:
      - name: proxy
        image: plantd-proxy:latest
        ports:
        - containerPort: 5000
        env:
        - name: PLANTD_PROXY_BROKER_ENDPOINT
          value: "tcp://plantd-broker:7200"
        - name: PLANTD_PROXY_IDENTITY_URL
          value: "http://plantd-identity:8080"
        livenessProbe:
          httpGet:
            path: /health
            port: 5000
          initialDelaySeconds: 30
          periodSeconds: 10
---
apiVersion: v1
kind: Service
metadata:
  name: plantd-proxy-service
spec:
  selector:
    app: plantd-proxy
  ports:
  - port: 80
    targetPort: 5000
  type: LoadBalancer
```

## Troubleshooting

### Common Issues

1. **Broker Connection Failed**:
   ```bash
   # Check broker connectivity
   telnet localhost 7200
   
   # Verify broker endpoint
   PLANTD_PROXY_LOG_LEVEL=debug ./build/plantd-proxy
   ```

2. **Service Not Found**:
   ```bash
   # List available services
   curl http://localhost:5000/api/v1/services
   
   # Check service registration
   curl http://localhost:5000/api/v1/services/org.plantd.YourService
   ```

3. **Authentication Errors**:
   ```bash
   # Test without authentication
   export PLANTD_PROXY_AUTH_ENABLED=false
   
   # Verify identity service
   curl http://localhost:8080/health
   ```

### Debug Mode

Enable comprehensive debugging:

```bash
export PLANTD_PROXY_LOG_LEVEL=trace
export PLANTD_PROXY_LOG_FORMAT=text
./build/plantd-proxy
```

## Contributing

See the main [plantd contributing guide](../README.md#contributing) for development setup and guidelines.

## License

This project is licensed under the MIT License - see the [LICENSE](../LICENSE) file for details.
