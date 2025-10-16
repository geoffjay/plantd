# PlantD API Documentation

**Version**: 1.0.0
**Protocol**: gRPC/Connect RPC over HTTP/2
**Authentication**: JWT Bearer Tokens

## Overview

PlantD provides a modern gRPC-based API for distributed control system operations. All services are accessible through a unified Traefik gateway with built-in authentication, rate limiting, and monitoring.

## Base URL

- **Production**: `https://api.plantd.com`
- **Development**: `http://localhost:8080`

## Authentication

All API requests require authentication using JWT bearer tokens obtained from the Identity Service.

### Getting an Authentication Token

```bash
# Login to get a token
curl -X POST https://auth.plantd.com/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "password"}'

# Response
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_in": 3600,
  "token_type": "Bearer"
}
```

### Using Tokens

Include the access token in the Authorization header:

```bash
curl -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
     https://api.plantd.com/plantd.state.v1.StateService/Get
```

## Services

### 1. State Service

The State Service provides distributed key-value storage with scoped access control.

**Base Path**: `/plantd.state.v1.StateService/`

#### Get State Value

**Method**: `POST /plantd.state.v1.StateService/Get`

```bash
curl -X POST https://api.plantd.com/plantd.state.v1.StateService/Get \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "key": "sensor.temperature.01",
    "service": "sensors"
  }'
```

**Response**:
```json
{
  "key": "sensor.temperature.01",
  "value": "23.5",
  "service": "sensors",
  "timestamp": "2024-11-20T10:30:00Z",
  "metadata": {
    "created_by": "user@example.com",
    "last_modified": "2024-11-20T10:30:00Z"
  }
}
```

#### Set State Value

**Method**: `POST /plantd.state.v1.StateService/Set`

```bash
curl -X POST https://api.plantd.com/plantd.state.v1.StateService/Set \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "key": "sensor.temperature.01",
    "value": "24.2",
    "service": "sensors",
    "metadata": {
      "unit": "celsius",
      "location": "building_a"
    }
  }'
```

**Response**:
```json
{
  "success": true,
  "key": "sensor.temperature.01",
  "service": "sensors",
  "timestamp": "2024-11-20T10:35:00Z"
}
```

#### List State Values

**Method**: `POST /plantd.state.v1.StateService/List`

```bash
curl -X POST https://api.plantd.com/plantd.state.v1.StateService/List \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "service": "sensors",
    "prefix": "sensor.temperature",
    "limit": 10
  }'
```

**Response**:
```json
{
  "items": [
    {
      "key": "sensor.temperature.01",
      "value": "24.2",
      "service": "sensors",
      "timestamp": "2024-11-20T10:35:00Z"
    },
    {
      "key": "sensor.temperature.02",
      "value": "23.8",
      "service": "sensors",
      "timestamp": "2024-11-20T10:34:00Z"
    }
  ],
  "total_count": 2,
  "page_token": ""
}
```

#### Delete State Value

**Method**: `POST /plantd.state.v1.StateService/Delete`

```bash
curl -X POST https://api.plantd.com/plantd.state.v1.StateService/Delete \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "key": "sensor.temperature.01",
    "service": "sensors"
  }'
```

**Response**:
```json
{
  "success": true,
  "key": "sensor.temperature.01",
  "service": "sensors"
}
```

#### List Service Scopes

**Method**: `POST /plantd.state.v1.StateService/ListScopes`

```bash
curl -X POST https://api.plantd.com/plantd.state.v1.StateService/ListScopes \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{}'
```

**Response**:
```json
{
  "scopes": [
    {
      "name": "sensors",
      "description": "Sensor data and configurations",
      "key_count": 25,
      "last_updated": "2024-11-20T10:35:00Z"
    },
    {
      "name": "controllers",
      "description": "Controller configurations",
      "key_count": 12,
      "last_updated": "2024-11-20T09:15:00Z"
    }
  ]
}
```

#### Batch Operations

**Method**: `POST /plantd.state.v1.StateService/BatchGet`

```bash
curl -X POST https://api.plantd.com/plantd.state.v1.StateService/BatchGet \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "requests": [
      {"key": "sensor.temperature.01", "service": "sensors"},
      {"key": "sensor.pressure.01", "service": "sensors"}
    ]
  }'
```

#### Watch State Changes (Streaming)

**Method**: `POST /plantd.state.v1.StateService/Watch`

```bash
curl -X POST https://api.plantd.com/plantd.state.v1.StateService/Watch \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "service": "sensors",
    "key_pattern": "sensor.temperature.*"
  }'
```

### 2. Identity Service

The Identity Service provides authentication, authorization, and user management.

**Base Path**: `/plantd.identity.v1.IdentityService/`

#### User Authentication

**Method**: `POST /plantd.identity.v1.IdentityService/Login`

```bash
curl -X POST https://api.plantd.com/plantd.identity.v1.IdentityService/Login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password",
    "organization_id": "org_123"
  }'
```

**Response**:
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_in": 3600,
  "token_type": "Bearer",
  "user": {
    "id": "user_456",
    "email": "user@example.com",
    "name": "John Doe",
    "roles": ["state-developer", "sensor-admin"]
  }
}
```

#### Token Validation

**Method**: `POST /plantd.identity.v1.IdentityService/ValidateToken`

```bash
curl -X POST https://api.plantd.com/plantd.identity.v1.IdentityService/ValidateToken \
  -H "Content-Type: application/json" \
  -d '{
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
  }'
```

**Response**:
```json
{
  "valid": true,
  "user": {
    "id": "user_456",
    "email": "user@example.com",
    "name": "John Doe",
    "organization_id": "org_123",
    "roles": ["state-developer", "sensor-admin"],
    "permissions": [
      "state:data:read",
      "state:data:write",
      "sensors:config:read"
    ]
  },
  "expires_at": "2024-11-20T11:30:00Z"
}
```

#### Token Refresh

**Method**: `POST /plantd.identity.v1.IdentityService/RefreshToken`

```bash
curl -X POST https://api.plantd.com/plantd.identity.v1.IdentityService/RefreshToken \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
  }'
```

#### User Management

**Method**: `POST /plantd.identity.v1.IdentityService/CreateUser`

```bash
curl -X POST https://api.plantd.com/plantd.identity.v1.IdentityService/CreateUser \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "newuser@example.com",
    "name": "Jane Smith",
    "password": "secure_password",
    "organization_id": "org_123",
    "roles": ["state-developer"]
  }'
```

#### Logout

**Method**: `POST /plantd.identity.v1.IdentityService/Logout`

```bash
curl -X POST https://api.plantd.com/plantd.identity.v1.IdentityService/Logout \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{}'
```

### 3. Health Service

The Health Service provides system health monitoring and status checks.

**Base Path**: `/plantd.health.v1.HealthService/`

#### Check Service Health

**Method**: `POST /plantd.health.v1.HealthService/Check`

```bash
curl -X POST https://api.plantd.com/plantd.health.v1.HealthService/Check \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "service": "plantd.state.v1.StateService"
  }'
```

**Response**:
```json
{
  "status": "SERVING",
  "service": "plantd.state.v1.StateService",
  "timestamp": "2024-11-20T10:30:00Z",
  "details": {
    "uptime": "72h15m30s",
    "version": "1.0.0",
    "build": "abc123def"
  }
}
```

#### Watch Health Status (Streaming)

**Method**: `POST /plantd.health.v1.HealthService/Watch`

```bash
curl -X POST https://api.plantd.com/plantd.health.v1.HealthService/Watch \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "service": "plantd.state.v1.StateService"
  }'
```

## Error Handling

### HTTP Status Codes

- `200 OK` - Successful request
- `400 Bad Request` - Invalid request parameters
- `401 Unauthorized` - Missing or invalid authentication
- `403 Forbidden` - Insufficient permissions
- `404 Not Found` - Resource not found
- `429 Too Many Requests` - Rate limit exceeded
- `500 Internal Server Error` - Server error
- `503 Service Unavailable` - Service temporarily unavailable

### Error Response Format

```json
{
  "error": {
    "code": "PERMISSION_DENIED",
    "message": "User does not have permission to access this resource",
    "details": [
      {
        "type": "type.googleapis.com/plantd.common.v1.ErrorInfo",
        "reason": "INSUFFICIENT_PERMISSIONS",
        "domain": "plantd.com",
        "metadata": {
          "required_permission": "state:data:write",
          "user_permissions": "state:data:read"
        }
      }
    ]
  }
}
```

## Rate Limiting

API requests are subject to rate limiting based on user authentication:

- **General API**: 100 requests/minute, 200 burst
- **Authentication endpoints**: 10 requests/minute, 20 burst
- **Streaming endpoints**: 5 concurrent connections per user

Rate limit headers are included in responses:

```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 87
X-RateLimit-Reset: 1700485200
```

## CLI Client Examples

### Installation

```bash
# Download the latest client
curl -L https://github.com/plantd/plantd/releases/latest/download/plant-grpc > plant-grpc
chmod +x plant-grpc
```

### Configuration

```bash
# Configure the client
plant-grpc config set grpc-endpoint https://api.plantd.com
plant-grpc config set identity-endpoint https://auth.plantd.com
```

### Authentication

```bash
# Login
plant-grpc auth auth-grpc login

# Check status
plant-grpc auth auth-grpc status

# Get user info
plant-grpc auth auth-grpc whoami
```

### State Operations

```bash
# Set a value
plant-grpc state state-grpc set sensor.temp.01 "23.5" --service=sensors

# Get a value
plant-grpc state state-grpc get sensor.temp.01 --service=sensors

# List values
plant-grpc state state-grpc list --service=sensors --prefix=sensor.temp

# List service scopes
plant-grpc state state-grpc list-scopes

# Delete a value
plant-grpc state state-grpc delete sensor.temp.01 --service=sensors
```

## SDK Examples

### Go SDK

```go
package main

import (
    "context"
    "fmt"
    "net/http"
    
    "connectrpc.com/connect"
    statev1 "github.com/plantd/plantd/gen/proto/go/plantd/state/v1"
    "github.com/plantd/plantd/gen/proto/go/plantd/state/v1/statev1connect"
)

func main() {
    client := statev1connect.NewStateServiceClient(
        http.DefaultClient,
        "https://api.plantd.com",
    )
    
    // Add authentication header
    req := connect.NewRequest(&statev1.GetRequest{
        Key:     "sensor.temperature.01",
        Service: "sensors",
    })
    req.Header().Set("Authorization", "Bearer "+token)
    
    res, err := client.Get(context.Background(), req)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Value: %s\n", res.Msg.Value)
}
```

### JavaScript/TypeScript SDK

```typescript
import { createPromiseClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { StateService } from "./gen/plantd/state/v1/state_connect";

const transport = createConnectTransport({
  baseUrl: "https://api.plantd.com",
  httpVersion: "2",
});

const client = createPromiseClient(StateService, transport);

// Get state value
const response = await client.get({
  key: "sensor.temperature.01",
  service: "sensors",
}, {
  headers: {
    "Authorization": `Bearer ${token}`,
  },
});

console.log("Value:", response.value);
```

### Python SDK

```python
import grpc
from plantd.state.v1 import state_pb2, state_pb2_grpc

# Create channel with authentication
channel = grpc.secure_channel('api.plantd.com:443', grpc.ssl_channel_credentials())
client = state_pb2_grpc.StateServiceStub(channel)

# Create request with metadata
metadata = [('authorization', f'Bearer {token}')]
request = state_pb2.GetRequest(
    key='sensor.temperature.01',
    service='sensors'
)

response = client.Get(request, metadata=metadata)
print(f"Value: {response.value}")
```

## Monitoring and Observability

### Metrics

All API endpoints expose Prometheus metrics:

- `http_requests_total` - Total HTTP requests
- `http_request_duration_seconds` - Request duration histogram
- `grpc_server_handled_total` - Total gRPC requests
- `grpc_server_handling_seconds` - gRPC request duration

### Tracing

Distributed tracing is available via Jaeger at `https://tracing.plantd.com`

### Health Endpoints

- **Gateway Health**: `GET /ping`
- **Service Health**: `GET /health`
- **Readiness**: `GET /ready`

## Support

- **Documentation**: https://docs.plantd.com
- **Issues**: https://github.com/plantd/plantd/issues
- **Community**: https://discord.gg/plantd
- **Support**: support@plantd.com 
