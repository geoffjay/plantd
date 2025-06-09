# PlantD Identity Service Documentation

## Overview

The PlantD Identity Service is a comprehensive authentication and authorization system designed for the PlantD distributed control system. It provides secure user management, role-based access control (RBAC), organization-based permissions, and JWT token management.

## 🚀 Current Status

**✅ Phase 2: Authentication - COMPLETED**
- Secure password handling with bcrypt
- JWT token management (Access/Refresh/Reset tokens)
- User registration and email verification
- Rate limiting and account lockout protection
- Password strength validation and security policies

**✅ Phase 3: Authorization & RBAC - COMPLETED**
- 43 granular permissions across 5 categories
- Role-based access control with organization scoping
- Authorization middleware for HTTP requests
- Permission caching and validation
- Organization membership management

## 📚 Documentation

- **[Authentication System](authentication.md)** - User authentication, passwords, JWT tokens
- **[Authorization & RBAC](authorization.md)** - Permissions, roles, organization-based access control
- **[API Reference](api-reference.md)** - Complete API documentation for all services
- **[Examples & Usage](examples.md)** - How to run examples and integrate the service
- **[Configuration](configuration.md)** - Service configuration options and setup
- **[Architecture](architecture.md)** - System design and component overview

## 🎯 Key Features

### Authentication
- **Secure Password Management**: bcrypt hashing, strength validation, policy enforcement
- **JWT Token System**: Access, refresh, and reset tokens with proper lifecycle management
- **Rate Limiting**: Protection against brute force attacks with configurable limits
- **User Registration**: Self-registration with optional email verification
- **Account Security**: Lockout protection, failed attempt tracking, security event logging

### Authorization
- **Granular Permissions**: 43 permissions across User, Organization, Role, Auth, and System categories
- **Role-Based Access Control**: Flexible role assignment with organization scoping
- **Organization Isolation**: Multi-tenant support with organization-based permission boundaries
- **Authorization Middleware**: HTTP middleware for request-level authorization
- **Permission Caching**: 5-minute cache for improved performance

### Security
- **Comprehensive Audit Logging**: All security events logged with context
- **Token Blacklisting**: Secure token revocation and invalidation
- **Input Validation**: Request validation and sanitization
- **Rate Limiting**: Configurable rate limits per IP and user
- **Password Policies**: Configurable strength requirements and pattern detection

## 🛠️ Quick Start

### Prerequisites
- Go 1.19+
- SQLite or PostgreSQL database

### Running Examples

1. **Authentication Example**:
   ```bash
   cd identity/examples
   PLANTD_IDENTITY_CONFIG=./identity.yaml go run auth_example.go
   ```

2. **RBAC Example**:
   ```bash
   cd identity/examples
   go run rbac_simple_example.go
   ```

### Integration

```go
import (
    "github.com/geoffjay/plantd/identity/internal/auth"
    "github.com/geoffjay/plantd/identity/internal/services"
)

// Initialize authentication service
authService := auth.NewAuthService(config, userRepo, userService, logger)

// Register user
user, err := registrationService.Register(ctx, registrationRequest)

// Authenticate user
tokens, err := authService.Login(ctx, loginRequest)

// Check permissions
hasPermission := rbacService.HasPermission(ctx, userID, "user:read", &orgID)
```

## 🔧 Development

### Project Structure
```
identity/
├── cmd/                    # Service entry point
├── internal/
│   ├── auth/              # Authentication & authorization logic
│   ├── config/            # Configuration management
│   ├── models/            # Data models (User, Organization, Role)
│   ├── repositories/      # Data access layer
│   └── services/          # Business logic layer
├── examples/              # Usage examples
└── docs/                  # Documentation
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/auth/...
```

## 📋 Implementation Status

### Completed Features ✅
- [x] User authentication and registration
- [x] JWT token management
- [x] Password security and validation
- [x] Rate limiting and account lockout
- [x] Role-based access control (RBAC)
- [x] Organization-based permissions
- [x] Authorization middleware
- [x] Security audit logging
- [x] Permission caching
- [x] Working examples and documentation

### Future Enhancements 🔮
- [ ] MDP protocol handlers (Phase 4)
- [ ] HTTP API endpoints
- [ ] gRPC service interface
- [ ] Email notification system
- [ ] Advanced audit dashboard
- [ ] OAuth2/OpenID Connect integration

## 🚨 Security Considerations

This is a **critical security component** that requires:
- Regular security audits
- Proper secret management in production
- Secure deployment practices
- Regular dependency updates
- Monitoring and alerting

## 📞 Support

For questions, issues, or contributions:
1. Check the documentation in this directory
2. Review the examples in `identity/examples/`
3. Examine the test files for usage patterns
4. Refer to the PlantD project documentation

---

**Note**: This service is production-ready but should undergo security review before deployment in critical environments. 
