[![Go Report Card](https://goreportcard.com/badge/github.com/geoffjay/plantd/core)](https://goreportcard.com/report/github.com/geoffjay/plantd/core)

---

# 🧱 Core Components

The core module provides shared libraries, utilities, and common functionality used across all plantd services. It serves as the foundation for building distributed control system components with consistent patterns and interfaces.

## Features

- **Service Framework**: Base service patterns and lifecycle management
- **Configuration Management**: Hierarchical configuration with environment variable support
- **Logging**: Structured logging with multiple output formats and levels
- **Message Bus**: ZeroMQ-based messaging abstractions and patterns
- **HTTP Utilities**: Common HTTP middleware and server utilities
- **MDP Protocol**: Majordomo Protocol implementation for reliable messaging
- **Utilities**: Common helper functions and data structures

## Components

### Service Framework (`service/`)

Base patterns for building plantd services:

```go
import "github.com/geoffjay/plantd/core/service"

// Implement the Service interface
type MyService struct {
    service.BaseService
}

func (s *MyService) Run(ctx context.Context, wg *sync.WaitGroup) {
    defer wg.Done()
    // Service implementation
}
```

### Configuration (`config/`)

Hierarchical configuration management:

```go
import "github.com/geoffjay/plantd/core/config"

type Config struct {
    Server struct {
        Port int    `yaml:"port" env:"PORT"`
        Host string `yaml:"host" env:"HOST"`
    } `yaml:"server"`
}

var cfg Config
err := config.LoadConfig("myservice", &cfg)
```

### Logging (`log/`)

Structured logging with multiple backends:

```go
import plog "github.com/geoffjay/plantd/core/log"

// Initialize logging
plog.Initialize(config.Log)

// Use structured logging
log.WithFields(log.Fields{
    "service": "myservice",
    "context": "operation",
}).Info("Operation completed")
```

### Message Bus (`bus/`)

ZeroMQ message bus abstractions:

```go
import "github.com/geoffjay/plantd/core/bus"

// Create a message bus
bus := bus.NewBus(bus.Config{
    Name:     "control",
    Backend:  "tcp://*:11000",
    Frontend: "tcp://*:11001",
})

// Start the bus
err := bus.Start(ctx, wg)
```

### MDP Protocol (`mdp/`)

Majordomo Protocol implementation:

```go
import "github.com/geoffjay/plantd/core/mdp"

// Create a worker
worker, err := mdp.NewWorker("tcp://localhost:7201", "org.plantd.MyService")

// Create a client
client, err := mdp.NewClient("tcp://localhost:7200")
```

### HTTP Utilities (`http/`)

Common HTTP middleware and utilities:

```go
import "github.com/geoffjay/plantd/core/http"

// Add common middleware
app.Use(http.LoggingMiddleware())
app.Use(http.CORSMiddleware())
app.Use(http.SecurityMiddleware())
```

### Utilities (`util/`)

Helper functions and common patterns:

```go
import "github.com/geoffjay/plantd/core/util"

// Environment variable with default
port := util.Getenv("PORT", "8080")

// String utilities
result := util.StringInSlice("item", []string{"item1", "item", "item2"})
```

## Installation

### As a Dependency

Add to your Go module:

```bash
go get github.com/geoffjay/plantd/core
```

### Development

```bash
# Clone the repository
git clone https://github.com/geoffjay/plantd.git
cd plantd/core

# Install dependencies
go mod download

# Run tests
go test ./...
```

## Usage Examples

### Basic Service

```go
package main

import (
    "context"
    "sync"
    
    "github.com/geoffjay/plantd/core/service"
    "github.com/geoffjay/plantd/core/config"
    plog "github.com/geoffjay/plantd/core/log"
)

type MyService struct {
    service.BaseService
    config MyConfig
}

type MyConfig struct {
    Port int `yaml:"port" env:"PLANTD_MYSERVICE_PORT"`
}

func NewService() *MyService {
    var cfg MyConfig
    config.LoadConfig("myservice", &cfg)
    
    return &MyService{
        config: cfg,
    }
}

func (s *MyService) Run(ctx context.Context, wg *sync.WaitGroup) {
    defer wg.Done()
    
    plog.Initialize(s.config.Log)
    
    // Service implementation
    <-ctx.Done()
}

func main() {
    service := NewService()
    
    ctx, cancel := context.WithCancel(context.Background())
    wg := &sync.WaitGroup{}
    
    wg.Add(1)
    go service.Run(ctx, wg)
    
    // Wait for termination signal
    // ...
    
    cancel()
    wg.Wait()
}
```

### Message Bus Integration

```go
package main

import (
    "context"
    "sync"
    
    "github.com/geoffjay/plantd/core/bus"
    "github.com/geoffjay/plantd/core/mdp"
)

func main() {
    // Create message buses
    controlBus := bus.NewBus(bus.Config{
        Name:     "control",
        Backend:  "tcp://*:11000",
        Frontend: "tcp://*:11001",
    })
    
    dataBus := bus.NewBus(bus.Config{
        Name:     "data", 
        Backend:  "tcp://*:12000",
        Frontend: "tcp://*:12001",
    })
    
    ctx, cancel := context.WithCancel(context.Background())
    wg := &sync.WaitGroup{}
    
    // Start buses
    wg.Add(2)
    go controlBus.Start(ctx, wg)
    go dataBus.Start(ctx, wg)
    
    // Create MDP worker
    worker, err := mdp.NewWorker("tcp://localhost:7201", "org.plantd.MyWorker")
    if err != nil {
        panic(err)
    }
    
    // Handle requests
    go func() {
        for {
            request := worker.Recv()
            // Process request
            response := processRequest(request)
            worker.Send(response)
        }
    }()
    
    // Wait for termination
    // ...
    
    cancel()
    wg.Wait()
}
```

## Configuration

### Environment Variables

Core components respect these environment variables:

```bash
# Logging
export PLANTD_LOG_LEVEL="debug"
export PLANTD_LOG_FORMAT="json"

# Message Bus
export PLANTD_BUS_BACKEND="tcp://*:11000"
export PLANTD_BUS_FRONTEND="tcp://*:11001"

# MDP Protocol
export PLANTD_MDP_ENDPOINT="tcp://localhost:7200"
export PLANTD_MDP_CLIENT_ENDPOINT="tcp://localhost:7201"
```

### Configuration Files

Core supports YAML configuration files:

```yaml
# config/core.yaml
log:
  level: "info"
  format: "text"
  output: "stdout"

bus:
  backend: "tcp://*:11000"
  frontend: "tcp://*:11001"
  
mdp:
  endpoint: "tcp://*:7200"
  client_endpoint: "tcp://*:7201"
  timeout: "30s"
```

## Testing

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test ./config
go test ./log
go test ./bus

# Run integration tests
go test -tags=integration ./...
```

## Architecture

The core module follows these design principles:

- **Modularity**: Each component can be used independently
- **Configuration**: Consistent configuration patterns across all components
- **Logging**: Structured logging with contextual information
- **Error Handling**: Comprehensive error handling and reporting
- **Testing**: Extensive test coverage with mocks and fixtures

### Dependencies

Core has minimal external dependencies:

- **ZeroMQ**: For message bus and MDP protocol
- **Logrus**: For structured logging
- **Viper**: For configuration management
- **YAML**: For configuration file parsing

## Contributing

See the main [plantd contributing guide](../README.md#contributing) for development setup and guidelines.

### Adding New Components

1. Create a new package directory under `core/`
2. Implement the component with proper interfaces
3. Add comprehensive tests
4. Update documentation
5. Add examples to this README

## License

This project is licensed under the MIT License - see the [LICENSE](../LICENSE) file for details.
