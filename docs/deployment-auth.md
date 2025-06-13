# PlantD Services Deployment with Authentication

This document describes how to deploy the plantd services with authentication enabled, specifically covering the State Service integration with the Identity Service.

## Overview

The plantd ecosystem now supports secure authentication and authorization through the Identity Service. The State Service is the first to implement this authentication pattern, which will serve as a template for other services.

## Service Dependencies

The services have the following dependency chain:

```
Broker Service (Base)
    ↓
Identity Service (Authentication)
    ↓
State Service (First Authenticated Service)
```

### Service Descriptions

- **Broker Service**: Central message broker for all plantd service communication
- **Identity Service**: Handles authentication, authorization, and user management
- **State Service**: Key-value storage with authentication and permission-based access control

## Deployment Options

### Option 1: Docker Compose (Recommended for Development)

Use the provided `docker-compose.plantd.yml` configuration:

```bash
# Start all services with authentication
docker-compose -f docker-compose.plantd.yml up -d

# Check service health
docker-compose -f docker-compose.plantd.yml ps

# View logs for specific service
docker-compose -f docker-compose.plantd.yml logs state
docker-compose -f docker-compose.plantd.yml logs identity
```

### Option 2: Manual Service Deployment

#### 1. Start Broker Service

```bash
# Build and run broker
cd broker
docker build -t plantd-broker .
docker run -d --name plantd-broker \
  -p 9797:9797 \
  -e PLANTD_BROKER_LOG_LEVEL=debug \
  plantd-broker
```

#### 2. Start Identity Service

```bash
# Build and run identity service
cd identity
docker build -t plantd-identity .
docker run -d --name plantd-identity \
  -p 8080:8080 \
  -e PLANTD_IDENTITY_BROKER_ENDPOINT=tcp://plantd-broker:9797 \
  -e PLANTD_IDENTITY_LOG_LEVEL=debug \
  --link plantd-broker \
  plantd-identity
```

#### 3. Start State Service

```bash
# Build and run state service with authentication
cd state
docker build -t plantd-state .
docker run -d --name plantd-state \
  -p 8081:8081 \
  -e PLANTD_STATE_BROKER_ENDPOINT=tcp://plantd-broker:9797 \
  -e PLANTD_STATE_IDENTITY_ENDPOINT=tcp://plantd-broker:9797 \
  -e PLANTD_STATE_LOG_LEVEL=debug \
  --link plantd-broker \
  --link plantd-identity \
  plantd-state
```

## Configuration

### Environment Variables

#### State Service Authentication Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `PLANTD_STATE_IDENTITY_ENDPOINT` | `tcp://127.0.0.1:9797` | Identity service endpoint |
| `PLANTD_STATE_IDENTITY_TIMEOUT` | `30s` | Timeout for identity service calls |
| `PLANTD_STATE_IDENTITY_RETRIES` | `3` | Number of retry attempts |

#### Identity Service Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `PLANTD_IDENTITY_BROKER_ENDPOINT` | `tcp://127.0.0.1:9797` | Broker service endpoint |
| `PLANTD_IDENTITY_DB_PATH` | `identity.db` | Database file path |
| `PLANTD_IDENTITY_HEALTH_PORT` | `8080` | Health check port |

### Configuration Files

#### State Service Configuration (state/config.yaml)

```yaml
env: development
broker-endpoint: tcp://localhost:9797
identity:
  endpoint: tcp://localhost:9797
  timeout: 30s
  retries: 3
database:
  adapter: bbolt
  uri: plantd-state.db
log:
  level: info
  formatter: text
```

## Health Checks

### Service Health Endpoints

- **Broker**: Check port 9797 connectivity
- **Identity**: `http://localhost:8080/health`
- **State**: `http://localhost:8081/health`

### Enhanced State Service Health Check

The state service health endpoint now provides detailed status including authentication status:

```bash
curl http://localhost:8081/health
```

Example response:
```json
{
  "status": "healthy",
  "store": true,
  "identity": {
    "status": "healthy",
    "connected": true,
    "message": "Identity service is responsive"
  },
  "auth_mode": "enabled",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

#### Health Status Meanings

- **auth_mode**: `enabled` or `disabled` - Whether authentication is active
- **identity.status**: `healthy`, `unhealthy`, or `disabled`
- **identity.connected**: Boolean indicating connection status
- **identity.message**: Detailed status message

## Graceful Degradation

The State Service implements graceful degradation when the Identity Service is unavailable:

### Authentication Enabled Mode
- All operations require valid JWT tokens
- Permission checks enforce access control
- Audit logging captures all operations

### Authentication Disabled Mode (Fallback)
- Service continues to operate without authentication
- All operations are allowed (backward compatibility)
- Warning logs indicate authentication is disabled

### Triggering Degradation

Authentication will be disabled if:
1. Identity service is unreachable during startup
2. Identity service configuration is invalid
3. Identity service health checks consistently fail

## Troubleshooting

### Common Issues

#### 1. State Service Cannot Connect to Identity Service

**Symptoms:**
- State service logs: "Failed to setup identity client - authentication will be disabled"
- Health check shows `auth_mode: disabled`

**Solutions:**
```bash
# Check identity service status
curl http://localhost:8080/health

# Check identity service logs
docker logs plantd-identity

# Verify network connectivity
docker exec plantd-state nc -z plantd-identity 8080
```

#### 2. Authentication Failures

**Symptoms:**
- Client receives "Authentication failed" errors
- State service logs show token validation failures

**Solutions:**
```bash
# Check identity service health
curl http://localhost:8080/health

# Verify token is valid
# (Use plant CLI auth commands)
plant auth status

# Check identity service logs for errors
docker logs plantd-identity
```

#### 3. Service Startup Dependencies

**Symptoms:**
- Services fail to start in correct order
- Connection errors during startup

**Solutions:**
```bash
# Use proper dependency management
docker-compose -f docker-compose.plantd.yml up --wait

# Manual startup with delays
docker-compose -f docker-compose.plantd.yml up broker
sleep 10
docker-compose -f docker-compose.plantd.yml up identity
sleep 10
docker-compose -f docker-compose.plantd.yml up state
```

### Debugging Commands

```bash
# Check all service status
docker-compose -f docker-compose.plantd.yml ps

# View real-time logs
docker-compose -f docker-compose.plantd.yml logs -f

# Test service health
curl http://localhost:8080/health  # Identity
curl http://localhost:8081/health  # State

# Connect to service containers
docker exec -it plantd_state_1 /bin/sh
docker exec -it plantd_identity_1 /bin/sh
```

## Security Considerations

### Network Security
- Services communicate internally via Docker network
- Only health check ports are exposed to host
- Message broker uses TCP with potential for TLS encryption

### Authentication Security
- JWT tokens have configurable expiration
- Permission-based access control
- Audit logging for all authenticated operations

### Data Security
- Database files stored in Docker volumes
- File system permissions protect sensitive data
- Configuration secrets via environment variables

## Monitoring

### Log Aggregation

All services support structured JSON logging:

```bash
# View logs in JSON format
docker-compose -f docker-compose.plantd.yml logs --json

# Integration with log aggregation systems
# (Loki, ELK stack, etc.)
```

### Metrics

Services expose health metrics:
- Service uptime
- Authentication success/failure rates
- Identity service connectivity
- Database health status

## Next Steps

This deployment pattern establishes the foundation for:

1. **Other Service Integration**: Apply same authentication pattern to other plantd services
2. **Enhanced Security**: Add TLS encryption, API keys, rate limiting
3. **Monitoring**: Implement comprehensive metrics and alerting
4. **Scalability**: Multi-instance deployments with load balancing

## Migration from Unauthenticated Deployment

### Existing Deployments

To migrate existing state service deployments:

1. **Deploy Identity Service** alongside existing services
2. **Update State Service** with authentication configuration
3. **Test Authentication** in development environment
4. **Migrate Production** during maintenance window

### Backward Compatibility

The authentication integration maintains backward compatibility:
- Existing plant CLI commands continue to work (after authentication)
- API contracts remain unchanged
- Configuration is additive (no breaking changes) 
