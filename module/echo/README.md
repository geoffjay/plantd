[![Go Report Card](https://goreportcard.com/badge/github.com/geoffjay/plantd/module/echo)](https://goreportcard.com/report/github.com/geoffjay/plantd/module/echo)

---

# 📡 Echo Module

The echo module is a simple testing and connectivity verification service for the plantd distributed control system. It provides echo functionality to test message routing, network connectivity, and service availability across the system.

## Features

- **Message Echo**: Echoes back received messages for connectivity testing
- **HTTP Endpoints**: RESTful API for testing HTTP connectivity
- **ZeroMQ Integration**: Tests message bus connectivity and routing
- **Health Monitoring**: Built-in health checks and status reporting
- **Load Testing**: Support for high-volume echo testing
- **Latency Measurement**: Measures round-trip message latency
- **Configurable Responses**: Customizable echo responses and delays

## Quick Start

### Prerequisites

- Go 1.24 or later
- ZeroMQ library (for message bus integration)

### Installation

```bash
# Build the echo module
cd module/echo
go build -o echo main.go

# Or build from project root
make build-echo
```

### Basic Usage

```bash
# Start with default configuration
./echo

# Start with custom port
PLANTD_MODULE_ECHO_PORT=5001 ./echo

# Start with debug logging
PLANTD_MODULE_ECHO_LOG_LEVEL=debug ./echo

# Start with custom bind address
PLANTD_MODULE_ECHO_ADDRESS=0.0.0.0 PLANTD_MODULE_ECHO_PORT=5000 ./echo
```

## Configuration

### Environment Variables

```bash
# Server configuration
export PLANTD_MODULE_ECHO_PORT="5000"
export PLANTD_MODULE_ECHO_ADDRESS="0.0.0.0"

# Logging configuration
export PLANTD_MODULE_ECHO_LOG_LEVEL="info"
export PLANTD_MODULE_ECHO_LOG_FORMAT="text"

# Message bus configuration
export PLANTD_MODULE_ECHO_BUS_ENDPOINT="tcp://localhost:11001"

# Response configuration
export PLANTD_MODULE_ECHO_DELAY="0ms"
export PLANTD_MODULE_ECHO_MAX_MESSAGE_SIZE="1MB"
```

### Configuration File

```yaml
# config/echo.yaml
server:
  port: 5000
  address: "0.0.0.0"
  read_timeout: "30s"
  write_timeout: "30s"

logging:
  level: "info"
  format: "text"

bus:
  endpoint: "tcp://localhost:11001"
  topics:
    - "echo.*"
    - "test.*"

response:
  delay: "0ms"
  max_message_size: "1MB"
  include_timestamp: true
  include_metadata: true
```

## API Endpoints

### HTTP Echo

```bash
# Simple echo
GET /echo?message=hello
# Response: {"message": "hello", "timestamp": "2024-01-01T12:00:00Z"}

# POST echo with JSON
POST /echo
{
  "message": "test message",
  "metadata": {"source": "client"}
}
# Response: {
#   "message": "test message",
#   "metadata": {"source": "client"},
#   "timestamp": "2024-01-01T12:00:00Z",
#   "echo_metadata": {
#     "service": "echo",
#     "latency_ms": 1.2
#   }
# }

# Echo with delay
GET /echo?message=delayed&delay=1s
# Response delayed by 1 second

# Bulk echo for load testing
POST /echo/bulk
{
  "messages": ["msg1", "msg2", "msg3"],
  "count": 100
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
#   "uptime": "1h30m",
#   "requests_handled": 1250,
#   "average_latency_ms": 2.1,
#   "memory_usage": "15MB"
# }

# Metrics
GET /metrics
# Response: Prometheus-style metrics
```

## Message Bus Integration

### ZeroMQ Echo

The echo module can participate in the plantd message bus:

```bash
# Subscribe to echo topics
# The module automatically subscribes to:
# - echo.*
# - test.*

# Example message structure
{
  "topic": "echo.test",
  "payload": {
    "message": "test message",
    "correlation_id": "uuid-1234",
    "timestamp": "2024-01-01T12:00:00Z"
  }
}
```

### MDP Integration

Integration with Majordomo Protocol for service discovery:

```go
// Example MDP worker registration
worker, err := mdp.NewWorker("tcp://localhost:7201", "org.plantd.Echo")
if err != nil {
    log.Fatal(err)
}

// Handle echo requests
for {
    request := worker.Recv()
    response := processEchoRequest(request)
    worker.Send(response)
}
```

## Testing and Validation

### Connectivity Testing

Use the echo module to test various connectivity scenarios:

```bash
# Test HTTP connectivity
curl http://localhost:5000/echo?message=connectivity_test

# Test with timeout
curl --max-time 5 http://localhost:5000/echo?message=timeout_test

# Test large messages
curl -X POST http://localhost:5000/echo \
  -H "Content-Type: application/json" \
  -d '{"message": "'$(head -c 1000 /dev/urandom | base64)'"}'
```

### Load Testing

```bash
# Simple load test with curl
for i in {1..100}; do
  curl -s http://localhost:5000/echo?message=load_test_$i &
done
wait

# Using Apache Bench
ab -n 1000 -c 10 http://localhost:5000/echo?message=benchmark

# Using wrk
wrk -t12 -c400 -d30s http://localhost:5000/echo?message=stress_test
```

### Latency Testing

```bash
# Measure round-trip latency
time curl -s http://localhost:5000/echo?message=latency_test

# Batch latency measurement
for i in {1..10}; do
  time curl -s http://localhost:5000/echo?message=latency_$i
done
```

## Integration Examples

### Client Integration

```go
package main

import (
    "encoding/json"
    "fmt"
    "net/http"
    "time"
)

type EchoResponse struct {
    Message   string    `json:"message"`
    Timestamp time.Time `json:"timestamp"`
    Latency   float64   `json:"latency_ms"`
}

func testEcho(message string) (*EchoResponse, error) {
    url := fmt.Sprintf("http://localhost:5000/echo?message=%s", message)
    
    start := time.Now()
    resp, err := http.Get(url)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var echoResp EchoResponse
    if err := json.NewDecoder(resp.Body).Decode(&echoResp); err != nil {
        return nil, err
    }
    
    echoResp.Latency = float64(time.Since(start).Nanoseconds()) / 1e6
    return &echoResp, nil
}

func main() {
    response, err := testEcho("hello_world")
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Echo response: %+v\n", response)
}
```

### Service Discovery Testing

```go
// Test service discovery through broker
func testServiceDiscovery() error {
    client, err := mdp.NewClient("tcp://localhost:7200")
    if err != nil {
        return err
    }
    
    // Send echo request through broker
    response := client.Send("org.plantd.Echo", "echo", []byte("service_discovery_test"))
    
    fmt.Printf("Service discovery response: %s\n", string(response))
    return nil
}
```

## Monitoring

### Metrics

The echo module exposes metrics for monitoring:

- **Request Count**: Total number of echo requests processed
- **Response Time**: Average, min, max response times
- **Error Rate**: Failed requests per second
- **Throughput**: Requests per second
- **Memory Usage**: Current memory consumption
- **Uptime**: Service uptime

### Logging

Structured logging with configurable levels:

```bash
# Enable debug logging
export PLANTD_MODULE_ECHO_LOG_LEVEL=debug

# JSON format for log aggregation
export PLANTD_MODULE_ECHO_LOG_FORMAT=json
```

### Health Checks

```bash
# Basic health check
curl http://localhost:5000/health

# Detailed health with dependencies
curl http://localhost:5000/health/detailed
```

## Development

### Hot Reload

```bash
# Install Air for hot reload
go install github.com/cosmtrek/air@latest

# Start with hot reload (if .air.toml exists)
air

# Or use go run with file watching
find . -name "*.go" | entr -r go run main.go
```

### Testing

```bash
# Run unit tests
go test ./...

# Run with coverage
go test -cover ./...

# Run integration tests
go test -tags=integration ./...

# Benchmark tests
go test -bench=. ./...
```

### Docker

```bash
# Build Docker image
docker build -t plantd-echo .

# Run with Docker
docker run -p 5000:5000 plantd-echo

# Run with environment variables
docker run -p 5000:5000 \
  -e PLANTD_MODULE_ECHO_LOG_LEVEL=debug \
  plantd-echo
```

## Deployment

### Standalone Deployment

```bash
# Build for production
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o echo main.go

# Create systemd service
sudo tee /etc/systemd/system/plantd-echo.service << EOF
[Unit]
Description=Plantd Echo Module
After=network.target

[Service]
Type=simple
User=plantd
WorkingDirectory=/opt/plantd
ExecStart=/opt/plantd/bin/echo
Environment=PLANTD_MODULE_ECHO_PORT=5000
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

# Enable and start service
sudo systemctl enable plantd-echo
sudo systemctl start plantd-echo
```

### Container Deployment

```yaml
# docker-compose.yml
version: '3.8'
services:
  echo:
    build: .
    ports:
      - "5000:5000"
    environment:
      - PLANTD_MODULE_ECHO_LOG_LEVEL=info
      - PLANTD_MODULE_ECHO_ADDRESS=0.0.0.0
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:5000/health"]
      interval: 30s
      timeout: 10s
      retries: 3
```

### Kubernetes Deployment

```yaml
# k8s-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: plantd-echo
spec:
  replicas: 3
  selector:
    matchLabels:
      app: plantd-echo
  template:
    metadata:
      labels:
        app: plantd-echo
    spec:
      containers:
      - name: echo
        image: plantd-echo:latest
        ports:
        - containerPort: 5000
        env:
        - name: PLANTD_MODULE_ECHO_ADDRESS
          value: "0.0.0.0"
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
  name: plantd-echo-service
spec:
  selector:
    app: plantd-echo
  ports:
  - port: 80
    targetPort: 5000
  type: LoadBalancer
```

## Troubleshooting

### Common Issues

1. **Port Already in Use**:
   ```bash
   # Check what's using the port
   lsof -i :5000
   
   # Use a different port
   export PLANTD_MODULE_ECHO_PORT=5001
   ```

2. **High Memory Usage**:
   ```bash
   # Monitor memory usage
   curl http://localhost:5000/metrics | grep memory
   
   # Reduce max message size
   export PLANTD_MODULE_ECHO_MAX_MESSAGE_SIZE="512KB"
   ```

3. **Connection Timeouts**:
   ```bash
   # Increase timeout values
   export PLANTD_MODULE_ECHO_READ_TIMEOUT="60s"
   export PLANTD_MODULE_ECHO_WRITE_TIMEOUT="60s"
   ```

### Debug Mode

Enable comprehensive debugging:

```bash
export PLANTD_MODULE_ECHO_LOG_LEVEL=trace
export PLANTD_MODULE_ECHO_LOG_FORMAT=text
./echo
```

## Use Cases

### Network Diagnostics

- Test connectivity between services
- Measure network latency and throughput
- Validate message routing paths
- Debug network configuration issues

### Load Testing

- Stress test the message broker
- Validate system performance under load
- Test service discovery mechanisms
- Benchmark message processing rates

### Development Testing

- Verify service integration
- Test API endpoints during development
- Validate configuration changes
- Debug message flow issues

## Contributing

See the main [plantd contributing guide](../../README.md#contributing) for development setup and guidelines.

## License

This project is licensed under the MIT License - see the [LICENSE](../../LICENSE) file for details. 
