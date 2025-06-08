[![Go Report Card](https://goreportcard.com/badge/github.com/geoffjay/plantd/app)](https://goreportcard.com/report/github.com/geoffjay/plantd/app)

---

# 🌐 Plantd Web Application

The web application provides a modern, browser-based interface for monitoring and controlling plantd distributed control system services. Built with Go Fiber and featuring a responsive UI, it serves as the primary dashboard for system administrators and operators.

## Features

- **Service Monitoring**: Real-time status and health monitoring of all plantd services
- **Configuration Management**: Web-based configuration interface for system settings
- **Secure Access**: HTTPS with automatic self-signed certificate generation for development
- **Session Management**: Secure user sessions with configurable storage
- **API Documentation**: Built-in Swagger documentation for REST endpoints
- **Responsive Design**: Modern UI that works on desktop and mobile devices

## Quick Start

### Prerequisites

- Go 1.24 or later
- Node.js (for frontend asset compilation)

### Development

1. **Build and run the application**:
   ```bash
   cd app
   go run main.go
   ```

2. **Access the web interface**:
   - Open your browser to `https://localhost:8443`
   - Accept the self-signed certificate warning (development only)

3. **Environment Configuration**:
   ```bash
   # Bind address (default: 127.0.0.1)
   export PLANTD_APP_BIND_ADDRESS="0.0.0.0"
   
   # Bind port (default: 8443)
   export PLANTD_APP_BIND_PORT="8443"
   
   # TLS certificate files (auto-generated in development)
   export PLANTD_APP_TLS_CERT="cert/app-cert.pem"
   export PLANTD_APP_TLS_KEY="cert/app-key.pem"
   ```

### Production Deployment

1. **Build the application**:
   ```bash
   make build-app
   ```

2. **Provide production certificates**:
   - Place your SSL certificate at the path specified by `PLANTD_APP_TLS_CERT`
   - Place your private key at the path specified by `PLANTD_APP_TLS_KEY`

3. **Configure environment**:
   ```bash
   export PLANTD_ENV="production"
   export PLANTD_APP_BIND_ADDRESS="0.0.0.0"
   export PLANTD_APP_BIND_PORT="443"
   ```

## Configuration

The application uses a hierarchical configuration system that supports:

- **Environment variables**: Override any setting with `PLANTD_APP_*` prefixed variables
- **Configuration files**: YAML configuration in `app/config/`
- **Runtime flags**: Command-line arguments for common settings

### Key Configuration Sections

- **Server**: Bind address, port, and TLS settings
- **Session**: Session storage, timeout, and security settings
- **CORS**: Cross-origin resource sharing policies
- **Logging**: Log levels, formats, and output destinations

## API Documentation

The application includes built-in Swagger documentation accessible at:
- `https://localhost:8443/swagger/` (when running)

### Key Endpoints

- `GET /api/v1/services` - List all plantd services and their status
- `GET /api/v1/health` - Application health check
- `POST /api/v1/config` - Update system configuration
- `GET /api/v1/metrics` - System metrics and performance data

## Development

### Hot Reload

The application supports hot reload during development using Air:

```bash
# Install Air if not already installed
go install github.com/cosmtrek/air@latest

# Start with hot reload
air
```

### Frontend Development

The application uses Tailwind CSS for styling:

```bash
# Install dependencies
npm install

# Watch for CSS changes
npm run watch-css

# Build production CSS
npm run build-css
```

### Testing

```bash
# Run unit tests
go test ./...

# Run with coverage
go test -cover ./...

# Run integration tests
go test -tags=integration ./...
```

## Architecture

The web application follows a clean architecture pattern:

- **Handlers**: HTTP request handlers and middleware
- **Services**: Business logic and service orchestration
- **Repository**: Data access layer and state management
- **Models**: Data structures and validation
- **Views**: HTML templates and frontend assets

### Security Features

- **HTTPS Only**: All traffic encrypted with TLS
- **Session Security**: Secure session cookies with CSRF protection
- **Rate Limiting**: Request rate limiting to prevent abuse
- **Input Validation**: Comprehensive input sanitization and validation
- **Security Headers**: Helmet middleware for security headers

## Integration

The web application integrates with other plantd services:

- **Broker**: Receives real-time updates via message bus
- **State**: Queries and updates distributed state
- **Logger**: Centralized logging and audit trails
- **Proxy**: API gateway and service discovery

## Troubleshooting

### Common Issues

1. **Certificate Errors**: 
   - In development, accept the self-signed certificate
   - In production, ensure valid SSL certificates are provided

2. **Port Conflicts**:
   - Change the bind port using `PLANTD_APP_BIND_PORT`
   - Ensure no other services are using the same port

3. **Permission Errors**:
   - Ensure the application has permission to bind to the specified port
   - Use `sudo` for ports below 1024 or configure capabilities

### Logging

Enable debug logging for troubleshooting:

```bash
export PLANTD_APP_LOG_LEVEL="debug"
```

## Contributing

See the main [plantd contributing guide](../README.md#contributing) for development setup and guidelines.

## License

This project is licensed under the MIT License - see the [LICENSE](../LICENSE) file for details. 
