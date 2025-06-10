# Examples & Usage Guide

## Overview

The PlantD Identity Service includes comprehensive examples demonstrating authentication and authorization features. This guide explains how to run the examples and integrate the service into your applications.

## Available Examples

### 1. Authentication Example (`auth_example.go`)

**Purpose**: Demonstrates the complete authentication system including user registration, login, token management, and password operations.

**Location**: `identity/examples/auth_example/auth_example.go`

**Features Demonstrated**:
- User registration with validation
- Email verification workflow (optional)
- User authentication and JWT token generation
- Token validation and parsing
- Token refresh mechanism
- Password change functionality
- Profile update operations
- Password reset flow
- Password strength evaluation
- Security statistics
- Logout and token blacklisting
- Rate limiting protection

### 2. RBAC Example (`rbac_simple_example.go`)

**Purpose**: Demonstrates the Role-Based Access Control system including permissions, roles, organization management, and authorization middleware.

**Location**: `identity/examples/rbac_example/rbac_simple_example.go`

**Features Demonstrated**:
- Permission system with 43 granular permissions
- Role creation and management
- User-role assignments
- Organization-based permissions
- Permission checking and validation
- Organization membership management
- Authorization context and scoping
- Security audit logging

## Running the Examples

### Prerequisites

Before running the examples, ensure you have:

1. **Go 1.19+** installed
2. **Project dependencies** installed (`go mod tidy`)
3. **Database access** (SQLite works out of the box)

### Authentication Example

#### Quick Start

```bash
# Navigate to authentication example directory
cd identity/examples/auth_example

# Run the authentication example
PLANTD_IDENTITY_CONFIG=./identity.yaml go run auth_example.go
```

#### Expected Output

The authentication example will show:

```
=== User Registration Example ===
Registration successful: Registration successful. You can now log in.
User ID: 1
Requires verification: false

=== User Login Example ===
Login successful for user: john.doe@example.com
Access token expires at: 2025-06-09 14:50:26 -0700 PDT
Token type: Bearer

=== Token Validation Example ===
Token is valid for user: john.doe@example.com (ID: 1)
User roles: []
User permissions: []

=== Token Refresh Example ===
Token refresh successful
New access token expires at: 2025-06-09 14:50:26 -0700 PDT

=== Password Change Example ===
Password change successful

=== Profile Update Example ===
Profile updated successfully
New username: jonathan.doe
Full name: Jonathan Doe

=== Password Reset Example ===
Password reset initiated successfully
In a real application, a reset email would be sent

=== Password Strength Example ===
Password 'weak' strength: 10/100
Password 'StrongerPassword123' strength: 70/100
Password 'VeryStrongPassword123!@#' strength: 90/100
Password 'password123' strength: 25/100

=== Security Statistics Example ===
Security stats: map[blocked_clients:0 total_accounts:0 total_clients:1]

=== Logout Example ===
Logout successful

=== Rate Limiting Example ===
Failed login attempt 1: invalid credentials
Failed login attempt 2: invalid credentials
Failed login attempt 3: invalid credentials

=== Authentication Example Complete ===
```

### RBAC Example

#### Quick Start

```bash
# Navigate to RBAC example directory
cd identity/examples/rbac_example

# Run the RBAC example
PLANTD_IDENTITY_CONFIG=./identity.yaml go run rbac_simple_example.go
```

#### Expected Output

The RBAC example will demonstrate:

```
=== PlantD Identity Service RBAC Example ===

=== 1. Permission System Overview ===
Total permissions: 43
Permission categories: 5

User Management Permissions (8):
- user:read
- user:write
- user:delete
- user:list
- user:search
- user:admin
- user:impersonate
- user:export

Organization Management Permissions (10):
- organization:read
- organization:write
...

=== 2. Permission Validation ===
✓ Valid permission 'user:read': true
✓ Invalid permission 'invalid:permission': false

=== 3. Role Management ===
✓ Created admin role with ID: 1
✓ Created user role with ID: 2

=== 4. User and Organization Setup ===
✓ Created test user with ID: 1
✓ Created test organization with ID: 1

=== 5. RBAC Permission Testing ===
✓ System admin role grants system:admin permission
✗ Regular user denied system:admin permission
✓ Permission system working correctly

=== 6. Organization Membership ===
✓ Organization membership service operational

=== 7. Authorization Middleware ===
✓ Authorization middleware framework operational

=== 8. Security Audit Logging ===
✓ Security events logged with structured data

=== Example completed successfully! ===
```

## Configuration

### Example Configuration File

The examples use a configuration file located at `identity/examples/identity.yaml`:

```yaml
env: development

database:
  driver: sqlite
  dsn: ":memory:"

server:
  port: 8080
  read_timeout: 30
  write_timeout: 30
  idle_timeout: 120

security:
  jwt_secret: "example-jwt-secret-change-in-production"
  jwt_refresh_secret: "example-refresh-secret-change-in-production"
  jwt_expiration: 900  # 15 minutes
  refresh_expiration: 604800  # 7 days
  jwt_issuer: "plantd-identity-example"
  bcrypt_cost: 12
  password_min_length: 8
  password_max_length: 128
  require_uppercase: true
  require_lowercase: true
  require_numbers: true
  require_special_chars: true
  rate_limit_rps: 10
  rate_limit_burst: 5
  max_failed_attempts: 5
  lockout_duration_minutes: 15
  allow_self_registration: true
  require_email_verification: false  # Disabled for example
  email_verification_expiry_hours: 24
  password_reset_expiry_hours: 2

log:
  formatter: text
  level: info

service:
  id: org.plantd.Identity.Example
```

## Integration Examples

### Basic Authentication Integration

```go
package main

import (
    "context"
    "log"
    
    "github.com/geoffjay/plantd/identity/internal/auth"
    "github.com/geoffjay/plantd/identity/internal/config"
    "github.com/geoffjay/plantd/identity/internal/repositories"
    "github.com/geoffjay/plantd/identity/internal/services"
    "github.com/geoffjay/plantd/identity/internal/models"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
)

func main() {
    // Load configuration
    cfg := config.GetConfig()
    
    // Setup database
    db, err := gorm.Open(sqlite.Open("identity.db"), &gorm.Config{})
    if err != nil {
        log.Fatal("Failed to connect to database:", err)
    }
    
    // Auto-migrate
    db.AutoMigrate(&models.User{}, &models.Organization{}, &models.Role{})
    
    // Initialize repositories
    userRepo := repositories.NewUserRepository(db)
    
    // Initialize services
    userService := services.NewUserService(userRepo, nil, nil)
    authService := auth.NewAuthService(
        cfg.ToAuthConfig(),
        userRepo,
        userService,
        logger,
    )
    
    // Your application logic here
    ctx := context.Background()
    
    // Authenticate user
    loginReq := &auth.AuthRequest{
        Identifier: "user@example.com",
        Password:   "userpassword",
        IPAddress:  "192.168.1.100",
        UserAgent:  "MyApp/1.0",
    }
    
    authResp, err := authService.Login(ctx, loginReq)
    if err != nil {
        log.Printf("Authentication failed: %v", err)
        return
    }
    
    log.Printf("User authenticated: %s", authResp.User.Email)
    log.Printf("Access token: %s", authResp.TokenPair.AccessToken)
}
```

### HTTP API Integration with Gin

```go
package main

import (
    "net/http"
    
    "github.com/gin-gonic/gin"
    "github.com/geoffjay/plantd/identity/internal/auth"
)

func setupRoutes(authService *auth.AuthService, rbacService *auth.RBACService) *gin.Engine {
    router := gin.Default()
    
    // Initialize authorization middleware
    authMiddleware := auth.NewAuthorizationMiddleware(
        authService,
        rbacService,
        logger,
    )
    
    // Public routes
    router.POST("/auth/login", loginHandler)
    router.POST("/auth/register", registerHandler)
    
    // Protected routes
    api := router.Group("/api")
    api.Use(authMiddleware.RequireAuth())
    {
        // User profile (requires authentication)
        api.GET("/profile", getProfileHandler)
        
        // User management (requires specific permissions)
        users := api.Group("/users")
        users.GET("", authMiddleware.RequirePermission(auth.PermissionUserList), listUsersHandler)
        users.POST("", authMiddleware.RequirePermission(auth.PermissionUserWrite), createUserHandler)
        users.GET("/:id", authMiddleware.RequireResourceAccess("user", auth.PermissionUserRead), getUserHandler)
        
        // Admin routes (requires admin role)
        admin := api.Group("/admin")
        admin.Use(authMiddleware.RequireRole("admin"))
        {
            admin.GET("/stats", getStatsHandler)
            admin.GET("/audit", getAuditHandler)
        }
    }
    
    return router
}

func loginHandler(c *gin.Context) {
    var req auth.AuthRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    // Set IP address from request
    req.IPAddress = c.ClientIP()
    req.UserAgent = c.GetHeader("User-Agent")
    
    resp, err := authService.Login(c.Request.Context(), &req)
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication failed"})
        return
    }
    
    c.JSON(http.StatusOK, resp)
}
```

## Troubleshooting

### Common Issues

#### "Config file not found"
**Problem**: `FATA[0000] error reading config file: open ./identity/identity.yaml: no such file or directory`

**Solution**:
```bash
# Set the correct config file path
export PLANTD_IDENTITY_CONFIG=./identity.yaml

# Or run from the correct directory
cd identity/examples
go run auth_example.go
```

#### "Multiple main functions" Error
**Problem**: `main redeclared in this block`

**Solution**: Run examples individually, not together:
```bash
# Run one example at a time
go run auth_example.go
# Not: go run *.go
```

#### "Password validation failed"
**Problem**: Password doesn't meet policy requirements

**Solution**: Use a strong password that meets all requirements:
- Minimum 8 characters
- Contains uppercase, lowercase, numbers, and special characters
- Avoids common patterns like "password", "123", "abc"

#### "Rate limit exceeded"
**Problem**: Too many requests in short time

**Solution**: Wait for rate limit to reset or adjust configuration:
```yaml
security:
  rate_limit_rps: 10      # Requests per second
  rate_limit_burst: 5     # Burst allowance
```

#### "Token validation failed"
**Problem**: JWT token is invalid or expired

**Solution**:
- Check token expiration time
- Verify JWT secret configuration
- Ensure token hasn't been blacklisted
- Use token refresh mechanism

### Getting Help

1. **Check the logs**: Examples include detailed logging for troubleshooting
2. **Review configuration**: Ensure all required settings are properly configured
3. **Test incrementally**: Start with basic authentication before adding authorization
4. **Examine test files**: Look at unit tests for usage patterns
5. **Check documentation**: Review other documentation files in this directory

### Performance Tuning

For production use, consider:

1. **Enable permission caching**: Reduces database queries
2. **Optimize database**: Add proper indexes for user-role-organization relationships
3. **Configure rate limiting**: Adjust limits based on your traffic patterns
4. **Monitor metrics**: Track authentication success rates and performance
5. **Use connection pooling**: For database connections in high-traffic scenarios
