# State Service Authentication Integration

This document describes the authentication and authorization integration implemented in the plantd State Service as part of Phase 1 of the broader authentication rollout.

## Overview

The State Service now integrates with the plantd Identity Service to provide secure, permission-based access to key-value storage operations. This implementation serves as the authentication template for all other plantd services.

## Authentication Features

### ✅ Implemented Features

- **JWT Token Validation**: All operations require valid authentication tokens
- **Permission-Based Access Control**: Granular permissions for different operations
- **Scope Isolation**: Service-specific data isolation with cross-service access control
- **Graceful Degradation**: Service continues operating when Identity Service is unavailable
- **Audit Logging**: Comprehensive logging of all authenticated operations
- **Health Monitoring**: Detailed health checks including authentication status

### 🔄 Authentication Flow

```
Client Request → Extract Token → Validate with Identity Service → 
Check Permissions → Execute Operation → Return Response
```

## Permission Model

### Scope Management (Global Permissions)
- `state:scope:create` - Create new service scopes
- `state:scope:delete` - Delete entire service scopes  
- `state:scope:list` - List available scopes

### Data Operations (Per-Scope Permissions)
- `state:data:read` - Read key-value pairs from specific scope
- `state:data:write` - Create/update key-value pairs in specific scope
- `state:data:delete` - Delete key-value pairs from specific scope

### Administrative Permissions
- `state:admin:full` - Full administrative access to all operations
- `state:health:read` - Access to health check endpoints

### Permission Hierarchy

1. **Global Permissions**: Apply to all scopes (e.g., admin operations)
2. **Scoped Permissions**: Apply to specific service namespaces  
3. **Operation-Specific**: Granular control over individual operations

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PLANTD_STATE_IDENTITY_ENDPOINT` | `tcp://127.0.0.1:9797` | Identity service endpoint |
| `PLANTD_STATE_IDENTITY_TIMEOUT` | `30s` | Timeout for identity service calls |
| `PLANTD_STATE_IDENTITY_RETRIES` | `3` | Number of retry attempts |

### Configuration File (config.yaml)

```yaml
identity:
  endpoint: tcp://localhost:9797
  timeout: 30s
  retries: 3
```

## Message Format

### Authenticated Requests

All state service operations now require an authentication token:

```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "service": "org.plantd.MyService",
  "key": "mykey", 
  "value": "myvalue"
}
```

### Response Format

Enhanced response format with consistent success/error structure:

**Success Response:**
```json
{
  "success": true,
  "data": {
    "scope": "org.plantd.MyService",
    "key": "mykey",
    "value": "myvalue",
    "status": "set"
  }
}
```

**Error Response:**
```json
{
  "success": false,
  "error": "Permission denied: insufficient permissions for state:data:write on org.plantd.MyService"
}
```

## Operation-Permission Mapping

| Operation | Required Permission | Scope |
|-----------|-------------------|-------|
| `create-scope` | `state:scope:create` | Global |
| `delete-scope` | `state:scope:delete` | Global |
| `set` | `state:data:write` | Per-scope |
| `get` | `state:data:read` | Per-scope |
| `delete` | `state:data:delete` | Per-scope |
| `health` | `state:health:read` | Optional |

## Graceful Degradation

### Authentication Enabled Mode (Normal Operation)
- All operations require valid JWT tokens
- Permission checks enforce access control
- Full audit logging of operations
- Enhanced error messages with permission details

### Authentication Disabled Mode (Fallback)
- Service operates without authentication when Identity Service unavailable
- All operations allowed (maintains backward compatibility)
- Warning logs indicate authentication is disabled
- Health checks show `auth_mode: disabled`

### Triggering Conditions

Authentication is disabled when:
1. Identity Service is unreachable during startup
2. Identity Service configuration is invalid
3. Identity Service health checks fail consistently

## Health Monitoring

### Enhanced Health Endpoint

Access: `GET http://localhost:8081/health`

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

### Status Indicators

- **status**: Overall service health (`healthy`/`unhealthy`)
- **store**: Database connectivity status
- **identity.status**: Identity service status (`healthy`/`unhealthy`/`disabled`)
- **identity.connected**: Connection status to Identity Service
- **identity.message**: Detailed status message
- **auth_mode**: Authentication status (`enabled`/`disabled`)

## Development Workflow

### 1. Start Services

```bash
# Start all services with Docker Compose
docker-compose -f docker-compose.plantd.yml up -d

# Or start manually (ensure Identity Service is running)
make run-state
```

### 2. Verify Health

```bash
# Check state service health
curl http://localhost:8081/health

# Verify authentication is enabled
curl http://localhost:8081/health | jq '.auth_mode'
```

### 3. Test Operations

```bash
# Using plant CLI (requires authentication setup)
plant auth login --email=user@example.com --password=secret
plant state set mykey myvalue --service=org.plantd.Test
plant state get mykey --service=org.plantd.Test
```

## Testing

### Unit Tests

```bash
# Run authentication middleware tests
go test ./auth/...

# Run service integration tests
go test ./...
```

### Integration Tests

```bash
# Test with Identity Service running
make test-integration

# Test graceful degradation (Identity Service stopped)
make test-degradation
```

## Troubleshooting

### Common Issues

#### 1. Authentication Disabled

**Symptoms:**
- Health check shows `auth_mode: disabled`
- Warning logs: "Failed to setup identity client"

**Solutions:**
- Verify Identity Service is running: `curl http://localhost:8080/health`
- Check identity endpoint configuration
- Review Identity Service logs

#### 2. Permission Denied Errors

**Symptoms:**
- Error: "Permission denied: insufficient permissions"
- Operations fail with 403 status

**Solutions:**
- Verify user has required permissions in Identity Service
- Check permission mapping for specific operation
- Review audit logs for permission details

#### 3. Token Validation Failures

**Symptoms:**
- Error: "Authentication failed: invalid token"
- Token appears valid but requests fail

**Solutions:**
- Verify token hasn't expired
- Check Identity Service connectivity
- Refresh authentication token

### Debug Commands

```bash
# View service logs
docker logs plantd_state_1

# Check identity connectivity
docker exec plantd_state_1 nc -z plantd_identity_1 8080

# Test health endpoints
curl -v http://localhost:8081/health
curl -v http://localhost:8080/health
```

## Performance Considerations

### Optimization Features

- **Permission Caching**: 5-minute TTL cache for permission checks
- **Connection Pooling**: Reuse Identity Service connections
- **Async Validation**: Non-blocking token validation where possible

### Performance Metrics

- **Authentication Overhead**: ~5-10ms per request
- **Permission Check Time**: ~1-3ms (cached) / ~10-20ms (uncached)
- **Memory Usage**: ~20MB additional for auth middleware

## Migration Guide

### From Unauthenticated State Service

1. **Deploy Identity Service** alongside existing state service
2. **Update Configuration** to include identity endpoint
3. **Restart State Service** to enable authentication
4. **Test Operations** with authenticated clients
5. **Update Clients** to include authentication tokens

### Backward Compatibility

The authentication integration maintains compatibility:
- Existing message formats work with added token field
- Service continues operating if Identity Service unavailable
- Configuration is additive (no breaking changes)
- Health checks remain available

## Future Enhancements

### Phase 2 Planned Features

- **API Key Authentication**: Service-to-service authentication
- **Role Templates**: Predefined permission sets
- **Advanced Auditing**: Detailed operation logging
- **Rate Limiting**: Protect against abuse
- **Token Refresh**: Automatic token renewal

### Template for Other Services

This implementation provides the template for integrating authentication into:
- Broker Service (worker registration)
- Logger Service (log access control)  
- Proxy Service (REST/GraphQL endpoints)
- App Service (web session management)

## Contributing

When extending the authentication system:

1. Follow the established permission naming convention
2. Implement graceful degradation for new features
3. Add comprehensive tests for auth workflows
4. Update documentation and examples
5. Consider performance impact of new checks

## References

- [Identity Service Documentation](../identity/README.md)
- [Deployment Guide](../docs/deployment-auth.md)
- [Authentication Plan](../docs/plan/execution/state/v0.md)
- [Permission Management Guide](../docs/identity/permissions.md) 
