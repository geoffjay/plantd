[![Go Report Card](https://goreportcard.com/badge/github.com/geoffjay/plantd/client)](https://goreportcard.com/report/github.com/geoffjay/plantd/client)

---

# 🖥️ Plantd Command Line Client

The plantd client is a powerful command-line interface (CLI) tool for interacting with and managing plantd distributed control system services. It provides administrators and developers with direct access to system functionality through an intuitive command structure.

## Features

- **Service Management**: Start, stop, and monitor plantd services
- **State Operations**: Query and manipulate distributed state across the system
- **Echo Testing**: Test connectivity and message routing between services
- **Configuration**: View and modify system configuration settings
- **Health Monitoring**: Check service health and system status
- **Batch Operations**: Execute multiple commands via scripts or configuration files

## Installation

### From Source

```bash
# Clone and build
cd client
go build -o plant main.go

# Install globally (optional)
sudo cp plant /usr/local/bin/
```

### Using Make

```bash
# Build client from project root
make build-client

# The binary will be available at ./build/plant
```

## Quick Start

### Basic Usage

```bash
# Show help and available commands
plant --help

# Check version
plant version

# Enable verbose output for debugging
plant --verbose <command>
```

### Configuration

The client uses a configuration file to connect to plantd services:

```bash
# Default configuration locations:
# $HOME/.config/plantd/client.yaml
# ./config.yaml

# Example configuration
cat > ~/.config/plantd/client.yaml << EOF
server:
  endpoint: "tcp://localhost:7200"
EOF
```

## Commands

### State Management

Interact with the distributed state service:

```bash
# Set a key-value pair for a service
plant state set --service="org.plantd.MyService" mykey myvalue

# Get a value by key
plant state get --service="org.plantd.MyService" mykey

# List all keys for a service
plant state list --service="org.plantd.MyService"

# Delete a key
plant state delete --service="org.plantd.MyService" mykey

# Create a new service scope
plant state create-scope --service="org.plantd.NewService"

# Delete an entire service scope
plant state delete-scope --service="org.plantd.OldService"
```

### Echo Testing

Test connectivity and message routing:

```bash
# Send a simple echo message
plant echo "Hello, plantd!"

# Test with specific service endpoint
plant echo --endpoint="tcp://localhost:5000" "Test message"

# Send multiple echo messages for load testing
plant echo --count=10 "Load test message"
```

### Service Operations

Manage plantd services:

```bash
# List all available services
plant services list

# Get detailed service information
plant services info --service="org.plantd.Broker"

# Check service health
plant services health --service="org.plantd.State"
```

## Configuration

### Configuration File Format

The client supports YAML configuration files:

```yaml
# ~/.config/plantd/client.yaml
server:
  endpoint: "tcp://localhost:7200"
  timeout: 30s
  retries: 3

logging:
  level: "info"
  format: "text"

defaults:
  service: "org.plantd.Default"
```

### Environment Variables

Override configuration with environment variables:

```bash
# Server endpoint
export PLANTD_CLIENT_ENDPOINT="tcp://production-broker:7200"

# Default service name
export PLANTD_CLIENT_DEFAULT_SERVICE="org.plantd.Production"

# Enable debug logging
export PLANTD_CLIENT_LOG_LEVEL="debug"
```

### Command-Line Flags

Global flags available for all commands:

- `--config`: Specify custom configuration file path
- `--verbose, -v`: Enable verbose output
- `--endpoint`: Override server endpoint
- `--timeout`: Set request timeout
- `--help, -h`: Show help information

## Examples

### Basic State Operations

```bash
# Store configuration for a temperature sensor
plant state set --service="org.plantd.TempSensor01" \
  config '{"interval": 1000, "unit": "celsius"}'

# Retrieve the configuration
plant state get --service="org.plantd.TempSensor01" config

# Store current reading
plant state set --service="org.plantd.TempSensor01" \
  current_temp "23.5"
```

### Batch Operations

Create a script for multiple operations:

```bash
#!/bin/bash
# setup-sensors.sh

# Create service scopes for multiple sensors
for i in {1..5}; do
  plant state create-scope --service="org.plantd.TempSensor$(printf "%02d" $i)"
  plant state set --service="org.plantd.TempSensor$(printf "%02d" $i)" \
    config '{"interval": 1000, "unit": "celsius"}'
done
```

### Health Monitoring

```bash
# Check if all critical services are running
services=("org.plantd.Broker" "org.plantd.State" "org.plantd.Logger")
for service in "${services[@]}"; do
  echo "Checking $service..."
  plant services health --service="$service"
done
```

## Integration

### Shell Completion

Generate shell completion scripts:

```bash
# Bash
plant completion bash > /etc/bash_completion.d/plant

# Zsh
plant completion zsh > "${fpath[1]}/_plant"

# Fish
plant completion fish > ~/.config/fish/completions/plant.fish
```

### Scripting

The client is designed for automation and scripting:

```bash
# Exit codes indicate success/failure
if plant state get --service="org.plantd.Service" key > /dev/null 2>&1; then
  echo "Key exists"
else
  echo "Key not found"
fi

# JSON output for parsing
plant services list --output=json | jq '.services[].name'
```

## Development

### Building from Source

```bash
# Install dependencies
go mod download

# Build for current platform
go build -o plant main.go

# Cross-compile for different platforms
GOOS=linux GOARCH=amd64 go build -o plant-linux-amd64 main.go
GOOS=windows GOARCH=amd64 go build -o plant-windows-amd64.exe main.go
```

### Adding New Commands

1. Create a new command file in `cmd/`
2. Implement the command using Cobra
3. Register the command in `cmd/cli.go`

Example:

```go
// cmd/newcommand.go
package cmd

import (
    "github.com/spf13/cobra"
)

var newCmd = &cobra.Command{
    Use:   "new",
    Short: "Description of new command",
    Run: func(cmd *cobra.Command, args []string) {
        // Implementation
    },
}

func init() {
    // Add flags and configuration
}
```

### Testing

```bash
# Run unit tests
go test ./...

# Run integration tests (requires running plantd services)
go test -tags=integration ./...

# Test with different configurations
PLANTD_CLIENT_ENDPOINT="tcp://localhost:7200" go test ./...
```

## Troubleshooting

### Common Issues

1. **Connection Refused**:
   ```bash
   # Check if broker service is running
   plant services health --service="org.plantd.Broker"
   
   # Verify endpoint configuration
   plant --verbose echo "test"
   ```

2. **Permission Denied**:
   ```bash
   # Check service permissions
   plant state get --service="org.plantd.Service" --verbose
   ```

3. **Timeout Errors**:
   ```bash
   # Increase timeout
   plant --timeout=60s state get --service="org.plantd.Service" key
   ```

### Debug Mode

Enable verbose logging for troubleshooting:

```bash
# Global verbose flag
plant --verbose <command>

# Environment variable
export PLANTD_CLIENT_LOG_LEVEL="debug"
plant <command>
```

## Contributing

See the main [plantd contributing guide](../README.md#contributing) for development setup and guidelines.

## License

This project is licensed under the MIT License - see the [LICENSE](../LICENSE) file for details. 
