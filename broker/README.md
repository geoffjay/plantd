[![Go Report Card](https://goreportcard.com/badge/github.com/geoffjay/plantd/broker)](https://goreportcard.com/report/github.com/geoffjay/plantd/broker)

---

# 🚌 Message Broker

The broker service is the central message routing hub for the plantd distributed control system. It implements the Majordomo Protocol (MDP) to provide reliable request-reply messaging between services and supports publish-subscribe patterns for real-time data distribution.

## Features

- **Majordomo Protocol**: Reliable request-reply messaging with automatic worker discovery
- **Publish-Subscribe**: Real-time data distribution to multiple subscribers
- **Service Discovery**: Automatic registration and health monitoring of workers
- **Load Balancing**: Distributes requests across multiple worker instances
- **Message Persistence**: Optional message queuing and persistence for reliability
- **Health Monitoring**: Built-in health checks and metrics endpoints
- **ZeroMQ Backend**: High-performance messaging using ZeroMQ transport

## Quick Start

### Prerequisites

- Go 1.24 or later
- ZeroMQ library (libzmq)

### Installation

```bash
# Install ZeroMQ (Ubuntu/Debian)
sudo apt-get install libzmq3-dev

# Install ZeroMQ (macOS)
brew install zeromq

# Build the broker
make build-broker
```

### Basic Usage

```bash
# Start with default configuration
./build/plantd-broker

# Start with debug logging
PLANTD_BROKER_LOG_LEVEL=debug ./build/plantd-broker

# Start with custom configuration
PLANTD_BROKER_CONFIG=configs/production.yaml ./build/plantd-broker
```

## Configuration

### Environment Variables

```bash
# Broker endpoint for client connections
export PLANTD_BROKER_ENDPOINT="tcp://*:7200"

# Client endpoint for worker connections  
export PLANTD_BROKER_CLIENT_ENDPOINT="tcp://*:7201"

# Health check port
export PLANTD_BROKER_HEALTH_PORT="8081"

# Log level (trace, debug, info, warn, error)
export PLANTD_BROKER_LOG_LEVEL="info"

# Configuration file path
export PLANTD_BROKER_CONFIG="configs/broker.yaml"
```

### Configuration File

Create a YAML configuration file:

```yaml
# configs/broker.yaml
endpoint: "tcp://*:7200"
client_endpoint: "tcp://*:7201"

buses:
  - name: "control"
    backend: "tcp://*:11000"
    frontend: "tcp://*:11001"
    capture: "tcp://*:11002"
  - name: "data"
    backend: "tcp://*:12000"
    frontend: "tcp://*:12001"
    capture: "tcp://*:12002"

log:
  level: "info"
  format: "json"
```

## Architecture

### Message Flow

1. **Clients** connect to the broker endpoint (`tcp://localhost:7200`)
2. **Workers** connect to the client endpoint (`tcp://localhost:7201`)
3. **Broker** routes messages between clients and workers
4. **Publish-Subscribe** buses handle real-time data distribution

### Service Registration

Workers automatically register with the broker:

```go
// Example worker registration
worker, err := mdp.NewWorker("tcp://localhost:7201", "org.plantd.MyService")
if err != nil {
    log.Fatal(err)
}

// Handle requests
for {
    request := worker.Recv()
    // Process request
    worker.Send(response)
}
```

### Client Communication

Clients send requests to services:

```go
// Example client request
client, err := mdp.NewClient("tcp://localhost:7200")
if err != nil {
    log.Fatal(err)
}

response := client.Send("org.plantd.MyService", "operation", data)
```

## Message Buses

The broker manages multiple message buses for different data types:

### Control Bus
- **Purpose**: Command and control messages
- **Pattern**: Request-reply with acknowledgment
- **Use Cases**: Service commands, configuration updates

### Data Bus  
- **Purpose**: Real-time sensor data and telemetry
- **Pattern**: Publish-subscribe
- **Use Cases**: Sensor readings, status updates, alerts

### Configuration

Each bus can be configured independently:

```yaml
buses:
  - name: "control"
    backend: "tcp://*:11000"   # Publishers connect here
    frontend: "tcp://*:11001"  # Subscribers connect here
    capture: "tcp://*:11002"   # Optional message capture
```

## Monitoring

### Health Checks

The broker provides health endpoints:

```bash
# Check broker health
curl http://localhost:8081/healthz

# Response
{
  "status": "pass",
  "version": "1",
  "releaseId": "1.0.0-SNAPSHOT",
  "checks": {
    "broker": {
      "status": "pass",
      "time": "2024-01-01T12:00:00Z"
    }
  }
}
```

### Metrics

Monitor broker performance:

- **Active Workers**: Number of registered workers per service
- **Message Throughput**: Messages per second
- **Queue Depth**: Pending messages in queues
- **Response Times**: Average request-reply latency

### Logging

Structured logging with configurable levels:

```bash
# Enable debug logging
export PLANTD_BROKER_LOG_LEVEL=debug

# JSON format for log aggregation
export PLANTD_BROKER_LOG_FORMAT=json
```

## Development

### Hot Reload

Use Air for development with hot reload:

```bash
# Install Air
go install github.com/cosmtrek/air@latest

# Start with hot reload
air
```

### Testing

```bash
# Run unit tests
go test ./...

# Run integration tests
go test -tags=integration ./...

# Test with race detection
go test -race ./...
```

### Docker

```bash
# Build Docker image
docker build -t plantd-broker .

# Run with Docker
docker run -p 7200:7200 -p 7201:7201 -p 8081:8081 plantd-broker
```

## Deployment

### Production Configuration

```yaml
# production.yaml
endpoint: "tcp://*:7200"
client_endpoint: "tcp://*:7201"

buses:
  - name: "control"
    backend: "tcp://*:11000"
    frontend: "tcp://*:11001"
  - name: "data"
    backend: "tcp://*:12000"
    frontend: "tcp://*:12001"

log:
  level: "warn"
  format: "json"
```

### Systemd Service

```ini
# /etc/systemd/system/plantd-broker.service
[Unit]
Description=Plantd Message Broker
After=network.target

[Service]
Type=simple
User=plantd
WorkingDirectory=/opt/plantd
ExecStart=/opt/plantd/bin/plantd-broker
Environment=PLANTD_BROKER_CONFIG=/opt/plantd/configs/production.yaml
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

### High Availability

For production deployments:

1. **Multiple Brokers**: Run multiple broker instances with load balancing
2. **Message Persistence**: Enable message persistence for critical services
3. **Monitoring**: Integrate with monitoring systems (Prometheus, Grafana)
4. **Backup**: Regular configuration and state backups

## Troubleshooting

### Common Issues

1. **Port Conflicts**:
   ```bash
   # Check if ports are in use
   netstat -tulpn | grep :7200
   
   # Change ports in configuration
   export PLANTD_BROKER_ENDPOINT="tcp://*:7300"
   ```

2. **ZeroMQ Errors**:
   ```bash
   # Ensure ZeroMQ is installed
   pkg-config --modversion libzmq
   
   # Install missing dependencies
   sudo apt-get install libzmq3-dev
   ```

3. **Worker Connection Issues**:
   ```bash
   # Check broker logs
   PLANTD_BROKER_LOG_LEVEL=debug ./build/plantd-broker
   
   # Verify endpoints are accessible
   telnet localhost 7201
   ```

### Debug Mode

Enable comprehensive debugging:

```bash
export PLANTD_BROKER_LOG_LEVEL=trace
export PLANTD_BROKER_LOG_FORMAT=text
./build/plantd-broker
```

## Contributing

See the main [plantd contributing guide](../README.md#contributing) for development setup and guidelines.

## License

This project is licensed under the MIT License - see the [LICENSE](../LICENSE) file for details.
