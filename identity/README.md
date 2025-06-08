[![Go Report Card](https://goreportcard.com/badge/github.com/geoffjay/plantd/identity)](https://goreportcard.com/report/github.com/geoffjay/plantd/identity)

---

# 🔐 Identity Service

The identity service provides authentication, authorization, and user management for the plantd distributed control system. It serves as the central authority for user credentials, permissions, and access control across all plantd services.

## Features

- **User Authentication**: Secure login with multiple authentication methods
- **Authorization**: Role-based access control (RBAC) for service permissions
- **Token Management**: JWT token generation and validation
- **User Management**: User registration, profile management, and password policies
- **Service Authentication**: Inter-service authentication and authorization
- **Audit Logging**: Comprehensive audit trails for security events
- **Multi-Factor Authentication**: Support for 2FA/MFA (planned)

## Status

🚧 **Under Development** - This service is currently in early development phase.

## Planned Architecture

### Authentication Methods

- **Username/Password**: Traditional credential-based authentication
- **API Keys**: Service-to-service authentication
- **JWT Tokens**: Stateless token-based authentication
- **OAuth2/OIDC**: Integration with external identity providers (planned)

### Authorization Model

```yaml
# Example RBAC configuration
roles:
  - name: "admin"
    permissions:
      - "service:*:*"
      - "user:*:*"
      - "config:*:*"
  
  - name: "operator"
    permissions:
      - "service:read:*"
      - "service:control:production"
      - "state:read:*"
      - "state:write:production"
  
  - name: "viewer"
    permissions:
      - "service:read:*"
      - "state:read:*"
      - "metrics:read:*"

users:
  - username: "admin"
    roles: ["admin"]
  - username: "operator1"
    roles: ["operator"]
```

## Planned API

### Authentication Endpoints

```bash
# User login
POST /auth/login
{
  "username": "user@example.com",
  "password": "password"
}

# Token refresh
POST /auth/refresh
{
  "refresh_token": "..."
}

# User logout
POST /auth/logout
{
  "token": "..."
}
```

### User Management Endpoints

```bash
# Create user
POST /users
{
  "username": "newuser",
  "email": "user@example.com",
  "password": "password",
  "roles": ["operator"]
}

# Get user profile
GET /users/{id}

# Update user
PUT /users/{id}
{
  "email": "newemail@example.com",
  "roles": ["admin", "operator"]
}

# Delete user
DELETE /users/{id}
```

### Permission Validation

```bash
# Check permissions
POST /auth/authorize
{
  "token": "jwt_token",
  "resource": "service:control:production",
  "action": "write"
}
```

## Development Roadmap

### Phase 1: Basic Authentication
- [ ] User registration and login
- [ ] Password hashing and validation
- [ ] JWT token generation and validation
- [ ] Basic user management API

### Phase 2: Authorization
- [ ] Role-based access control (RBAC)
- [ ] Permission management
- [ ] Service-to-service authentication
- [ ] Authorization middleware for other services

### Phase 3: Advanced Features
- [ ] Multi-factor authentication (MFA)
- [ ] OAuth2/OIDC integration
- [ ] Audit logging and security events
- [ ] Password policies and complexity requirements

### Phase 4: Enterprise Features
- [ ] LDAP/Active Directory integration
- [ ] Single Sign-On (SSO)
- [ ] Session management
- [ ] Advanced security policies

## Configuration (Planned)

### Environment Variables

```bash
# Database connection
export PLANTD_IDENTITY_DB_URL="postgres://user:pass@localhost/plantd_identity"

# JWT configuration
export PLANTD_IDENTITY_JWT_SECRET="your-secret-key"
export PLANTD_IDENTITY_JWT_EXPIRY="24h"

# Server configuration
export PLANTD_IDENTITY_PORT="8080"
export PLANTD_IDENTITY_HOST="0.0.0.0"

# Security settings
export PLANTD_IDENTITY_PASSWORD_MIN_LENGTH="8"
export PLANTD_IDENTITY_REQUIRE_MFA="false"
```

### Configuration File

```yaml
# config/identity.yaml
server:
  port: 8080
  host: "0.0.0.0"

database:
  url: "postgres://user:pass@localhost/plantd_identity"
  max_connections: 10
  ssl_mode: "require"

jwt:
  secret: "your-secret-key"
  expiry: "24h"
  refresh_expiry: "7d"

security:
  password_min_length: 8
  password_require_special: true
  password_require_numbers: true
  max_login_attempts: 5
  lockout_duration: "15m"

oauth:
  providers:
    - name: "google"
      client_id: "your-client-id"
      client_secret: "your-client-secret"
      redirect_url: "http://localhost:8080/auth/callback/google"
```

## Integration with Other Services

### Service Authentication

Other plantd services will integrate with the identity service:

```go
// Example middleware for service authentication
func AuthMiddleware(identityURL string) fiber.Handler {
    return func(c *fiber.Ctx) error {
        token := c.Get("Authorization")
        if token == "" {
            return c.Status(401).JSON(fiber.Map{"error": "Missing token"})
        }
        
        // Validate token with identity service
        valid, err := validateToken(identityURL, token)
        if err != nil || !valid {
            return c.Status(401).JSON(fiber.Map{"error": "Invalid token"})
        }
        
        return c.Next()
    }
}
```

### Permission Checking

```go
// Example permission checking
func RequirePermission(permission string) fiber.Handler {
    return func(c *fiber.Ctx) error {
        token := c.Get("Authorization")
        
        hasPermission, err := checkPermission(identityURL, token, permission)
        if err != nil || !hasPermission {
            return c.Status(403).JSON(fiber.Map{"error": "Insufficient permissions"})
        }
        
        return c.Next()
    }
}

// Usage in routes
app.Get("/admin/users", RequirePermission("user:read:*"), getUsersHandler)
app.Post("/services/restart", RequirePermission("service:control:*"), restartServiceHandler)
```

## Database Schema (Planned)

```sql
-- Users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(255) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    last_login TIMESTAMP,
    is_active BOOLEAN DEFAULT true
);

-- Roles table
CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) UNIQUE NOT NULL,
    description TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Permissions table
CREATE TABLE permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resource VARCHAR(255) NOT NULL,
    action VARCHAR(255) NOT NULL,
    scope VARCHAR(255) DEFAULT '*',
    created_at TIMESTAMP DEFAULT NOW()
);

-- User roles junction table
CREATE TABLE user_roles (
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);

-- Role permissions junction table
CREATE TABLE role_permissions (
    role_id UUID REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

-- Audit log table
CREATE TABLE audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id),
    action VARCHAR(255) NOT NULL,
    resource VARCHAR(255),
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);
```

## Security Considerations

### Password Security
- Bcrypt hashing with configurable cost
- Password complexity requirements
- Password history to prevent reuse
- Account lockout after failed attempts

### Token Security
- Short-lived access tokens (15-30 minutes)
- Longer-lived refresh tokens (7 days)
- Token blacklisting for logout
- Secure token storage recommendations

### Communication Security
- HTTPS/TLS for all communications
- Certificate-based service authentication
- Request signing for critical operations
- Rate limiting and DDoS protection

## Contributing

This service is in early development. Contributions are welcome:

1. Review the planned architecture and provide feedback
2. Implement core authentication features
3. Add comprehensive tests
4. Improve security measures
5. Add documentation and examples

See the main [plantd contributing guide](../README.md#contributing) for development setup and guidelines.

## License

This project is licensed under the MIT License - see the [LICENSE](../LICENSE) file for details.
