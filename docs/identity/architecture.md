# Architecture Overview

## System Architecture

The PlantD Identity Service follows a layered architecture pattern with clear separation of concerns, dependency injection, and clean interfaces between components.

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    PlantD Identity Service                   │
├─────────────────────────────────────────────────────────────┤
│                     HTTP/API Layer                          │
│  ┌─────────────────┐  ┌─────────────────┐  ┌──────────────┐ │
│  │   Auth Routes   │  │   User Routes   │  │ Admin Routes │ │
│  │   /auth/*       │  │   /users/*      │  │  /admin/*    │ │
│  └─────────────────┘  └─────────────────┘  └──────────────┘ │
├─────────────────────────────────────────────────────────────┤
│                  Authorization Middleware                   │
│  ┌─────────────────┐  ┌─────────────────┐  ┌──────────────┐ │
│  │ Authentication  │  │   Permission    │  │     RBAC     │ │
│  │   Validation    │  │   Checking      │  │  Enforcement │ │
│  └─────────────────┘  └─────────────────┘  └──────────────┘ │
├─────────────────────────────────────────────────────────────┤
│                      Service Layer                          │
│  ┌─────────────────┐  ┌─────────────────┐  ┌──────────────┐ │
│  │  Auth Service   │  │  User Service   │  │ RBAC Service │ │
│  │  - Login        │  │  - CRUD         │  │ - Permissions│ │
│  │  - Registration │  │  - Validation   │  │ - Roles      │ │
│  │  - JWT Tokens   │  │  - Profile Mgmt │  │ - Org Access │ │
│  └─────────────────┘  └─────────────────┘  └──────────────┘ │
├─────────────────────────────────────────────────────────────┤
│                    Repository Layer                         │
│  ┌─────────────────┐  ┌─────────────────┐  ┌──────────────┐ │
│  │ User Repository │  │  Org Repository │  │Role Repository│ │
│  │ - GORM Impl     │  │  - GORM Impl    │  │ - GORM Impl  │ │
│  │ - Interface     │  │  - Interface    │  │ - Interface  │ │
│  └─────────────────┘  └─────────────────┘  └──────────────┘ │
├─────────────────────────────────────────────────────────────┤
│                      Data Layer                             │
│  ┌─────────────────┐  ┌─────────────────┐  ┌──────────────┐ │
│  │     Database    │  │      Cache      │  │    Audit     │ │
│  │ PostgreSQL/     │  │   In-Memory/    │  │    Logs      │ │
│  │   SQLite        │  │     Redis       │  │   (Logrus)   │ │
│  └─────────────────┘  └─────────────────┘  └──────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

## Core Components

### 1. Authentication Layer

#### JWT Token Manager
```go
type JWTManager struct {
    config     *JWTConfig
    blacklist  TokenBlacklist
}

// Features:
// - Multiple token types (Access, Refresh, Reset)
// - Custom claims with user context
// - Token blacklisting for secure logout
// - Automatic token cleanup
```

#### Password Security
```go
type PasswordValidator struct {
    config *PasswordConfig
}

// Features:
// - bcrypt hashing with configurable cost
// - Password strength scoring (0-100)
// - Pattern detection (weak passwords)
// - Policy enforcement
```

#### Rate Limiter
```go
type RateLimiter struct {
    limiters map[string]*rate.Limiter
    config   *RateLimiterConfig
}

// Features:
// - Token bucket algorithm
// - Per-IP rate limiting
// - Configurable burst allowance
// - Account lockout protection
```

### 2. Authorization Layer

#### Permission System
```go
type Permission string

const (
    PermissionUserRead   Permission = "user:read"
    PermissionUserWrite  Permission = "user:write"
    // ... 43 total permissions across 5 categories
)

// Features:
// - 43 granular permissions
// - 5 permission categories
// - Permission validation
// - Category-based organization
```

#### RBAC Service
```go
type RBACService struct {
    userRepo    repositories.UserRepository
    roleRepo    repositories.RoleRepository
    cache       *PermissionCache
    logger      *logrus.Logger
}

// Features:
// - Permission checking with caching
// - Role assignment and management
// - Organization-scoped permissions
// - Audit logging
```

#### Authorization Context
```go
type AuthorizationContext struct {
    UserID         uint
    OrganizationID *uint
    ResourceID     *uint
    ResourceType   string
    IPAddress      string
    UserAgent      string
}

// Features:
// - Context-aware authorization
// - Organization scoping
// - Resource-level permissions
// - Audit trail context
```

### 3. Service Layer

#### Authentication Service
```go
type AuthService struct {
    config          *AuthConfig
    userRepo        repositories.UserRepository
    userService     services.UserService
    jwtManager      *JWTManager
    passwordValidator *PasswordValidator
    rateLimiter     *RateLimiter
    logger          *logrus.Logger
}

// Responsibilities:
// - User authentication (login/logout)
// - Token generation and validation
// - Account lockout management
// - Security event logging
```

#### User Service
```go
type UserService struct {
    userRepo repositories.UserRepository
    orgRepo  repositories.OrganizationRepository
    roleRepo repositories.RoleRepository
    logger   *logrus.Logger
}

// Responsibilities:
// - User CRUD operations
// - Profile management
// - Input validation
// - Business logic enforcement
```

#### Organization Membership Service
```go
type OrganizationMembershipService struct {
    userRepo repositories.UserRepository
    orgRepo  repositories.OrganizationRepository
    roleRepo repositories.RoleRepository
    rbac     *RBACService
    logger   *logrus.Logger
}

// Responsibilities:
// - Organization membership management
// - Multi-tenant user access
// - Organization-scoped role assignments
// - Membership validation
```

### 4. Repository Layer

#### Interface Pattern
```go
type UserRepository interface {
    Create(ctx context.Context, user *models.User) error
    GetByID(ctx context.Context, id uint) (*models.User, error)
    GetByEmail(ctx context.Context, email string) (*models.User, error)
    GetByUsername(ctx context.Context, username string) (*models.User, error)
    Update(ctx context.Context, user *models.User) error
    Delete(ctx context.Context, id uint) error
    List(ctx context.Context, offset, limit int) ([]*models.User, error)
    GetWithRoles(ctx context.Context, id uint, orgID *uint) (*models.User, error)
}

// Benefits:
// - Testable with mock implementations
// - Database-agnostic interface
// - Clear data access patterns
// - Dependency injection support
```

### 5. Data Models

#### User Model
```go
type User struct {
    ID              uint           `gorm:"primaryKey"`
    Email           string         `gorm:"uniqueIndex;not null"`
    Username        string         `gorm:"uniqueIndex;not null"`
    HashedPassword  string         `gorm:"not null"`
    FirstName       string         `gorm:"not null"`
    LastName        string         `gorm:"not null"`
    IsActive        bool           `gorm:"default:true"`
    EmailVerified   bool           `gorm:"default:false"`
    LastLoginAt     *time.Time
    FailedAttempts  int            `gorm:"default:0"`
    LockedUntil     *time.Time
    CreatedAt       time.Time
    UpdatedAt       time.Time
    DeletedAt       gorm.DeletedAt `gorm:"index"`
    
    // Relationships
    Organizations []Organization `gorm:"many2many:user_organizations;"`
    Roles         []Role         `gorm:"many2many:user_roles;"`
}
```

#### Organization Model
```go
type Organization struct {
    ID          uint           `gorm:"primaryKey"`
    Name        string         `gorm:"not null"`
    Slug        string         `gorm:"uniqueIndex;not null"`
    Description string
    IsActive    bool           `gorm:"default:true"`
    CreatedAt   time.Time
    UpdatedAt   time.Time
    DeletedAt   gorm.DeletedAt `gorm:"index"`
    
    // Relationships
    Users []User `gorm:"many2many:user_organizations;"`
}
```

#### Role Model
```go
type Role struct {
    ID          uint           `gorm:"primaryKey"`
    Name        string         `gorm:"not null"`
    Description string
    Permissions pq.StringArray `gorm:"type:text[]"`
    Scope       string         `gorm:"default:'global'"`
    CreatedAt   time.Time
    UpdatedAt   time.Time
    DeletedAt   gorm.DeletedAt `gorm:"index"`
    
    // Relationships
    Users []User `gorm:"many2many:user_roles;"`
}
```

## Data Flow

### Authentication Flow

```
Client Request
      ↓
[1] HTTP Handler
      ↓
[2] Input Validation
      ↓
[3] Rate Limiting Check
      ↓
[4] User Lookup (Repository)
      ↓
[5] Password Verification
      ↓
[6] Account Status Check
      ↓
[7] JWT Token Generation
      ↓
[8] Security Event Logging
      ↓
[9] Response with Tokens
```

### Authorization Flow

```
Authenticated Request
      ↓
[1] JWT Token Validation
      ↓
[2] User Context Extraction
      ↓
[3] Permission Cache Check
      ↓
[4] RBAC Permission Lookup
      ↓
[5] Organization Scope Check
      ↓
[6] Authorization Decision
      ↓
[7] Audit Event Logging
      ↓
[8] Allow/Deny Response
```

### User Registration Flow

```
Registration Request
      ↓
[1] Input Validation
      ↓
[2] Password Strength Check
      ↓
[3] Duplicate Check (Email/Username)
      ↓
[4] Password Hashing
      ↓
[5] User Creation (Repository)
      ↓
[6] Email Verification Token
      ↓
[7] Welcome Email (Optional)
      ↓
[8] Registration Confirmation
```

## Security Architecture

### Defense in Depth

1. **Input Validation**: All inputs validated at handler level
2. **Rate Limiting**: Per-IP and per-user rate limits
3. **Authentication**: JWT tokens with secure signing
4. **Authorization**: Granular permission checking
5. **Audit Logging**: Comprehensive security event tracking
6. **Data Protection**: Secure password hashing, data encryption

### Token Security

1. **Access Tokens**: Short-lived (15 minutes), contain user context
2. **Refresh Tokens**: Long-lived (7 days), enable token renewal
3. **Reset Tokens**: Very short-lived (2 hours), single-use password reset
4. **Token Blacklisting**: Immediate token revocation for logout
5. **Token Rotation**: Refresh tokens rotate on use

### Permission Model

```
User → Roles → Permissions
          ↓
   Organization Scope
          ↓
    Permission Check
```

## Scalability Considerations

### Horizontal Scaling

1. **Stateless Design**: No server-side session state
2. **Database Separation**: Read/write splitting capability
3. **Caching Strategy**: Permission and token caching
4. **Load Balancing**: Multiple service instances

### Performance Optimizations

1. **Permission Caching**: 5-minute TTL reduces database queries
2. **Database Indexing**: Proper indexes on lookup fields
3. **Connection Pooling**: Configurable database connections
4. **Lazy Loading**: Efficient relationship loading

### Monitoring and Observability

1. **Structured Logging**: JSON format with correlation IDs
2. **Metrics Collection**: Prometheus-compatible metrics
3. **Health Checks**: Liveness and readiness probes
4. **Distributed Tracing**: Request tracking across services

## Integration Patterns

### Service Integration

```go
// Example service initialization
func NewIdentityService(config *Config) (*IdentityService, error) {
    // Database connection
    db, err := setupDatabase(config.Database)
    if err != nil {
        return nil, err
    }
    
    // Repository layer
    userRepo := repositories.NewUserRepository(db)
    orgRepo := repositories.NewOrganizationRepository(db)
    roleRepo := repositories.NewRoleRepository(db)
    
    // Service layer
    userService := services.NewUserService(userRepo, orgRepo, roleRepo)
    authService := auth.NewAuthService(config.Auth, userRepo, userService, logger)
    rbacService := auth.NewRBACService(userRepo, roleRepo, logger)
    
    // Middleware
    authMiddleware := auth.NewAuthorizationMiddleware(authService, rbacService, logger)
    
    return &IdentityService{
        AuthService:    authService,
        UserService:    userService,
        RBACService:    rbacService,
        Middleware:     authMiddleware,
    }, nil
}
```

### Client Integration

```go
// Client usage pattern
client := identity.NewClient(identityServiceURL)

// Authenticate
tokens, err := client.Authenticate("user@example.com", "password")
if err != nil {
    return err
}

// Use access token for requests
client.SetAccessToken(tokens.AccessToken)

// Check permissions
hasPermission, err := client.HasPermission("user:read", organizationID)
if err != nil {
    return err
}
```

## Error Handling Strategy

### Error Types

1. **Authentication Errors**: Invalid credentials, account locked
2. **Authorization Errors**: Insufficient permissions, access denied
3. **Validation Errors**: Invalid input, policy violations
4. **System Errors**: Database failures, external service issues

### Error Response Format

```go
type ErrorResponse struct {
    Error      string            `json:"error"`
    Message    string            `json:"message"`
    Code       string            `json:"code,omitempty"`
    Details    map[string]string `json:"details,omitempty"`
    RequestID  string            `json:"request_id,omitempty"`
    Timestamp  time.Time         `json:"timestamp"`
}
```

## Testing Strategy

### Unit Testing

1. **Repository Testing**: Database operations with test containers
2. **Service Testing**: Business logic with mocked dependencies
3. **Handler Testing**: HTTP endpoints with test servers
4. **Authentication Testing**: Token validation and security

### Integration Testing

1. **End-to-End Flows**: Complete authentication/authorization workflows
2. **Database Integration**: Real database operations
3. **Security Testing**: Permission enforcement, rate limiting
4. **Performance Testing**: Load testing and benchmarks

## Deployment Architecture

### Container Architecture

```dockerfile
FROM golang:1.19-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o identity-service ./cmd/

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/identity-service .
EXPOSE 8080 9090
CMD ["./identity-service"]
```

### Kubernetes Deployment

```yaml
# Service, ConfigMap, Secret, Deployment
apiVersion: v1
kind: Service
metadata:
  name: identity-service
spec:
  selector:
    app: identity-service
  ports:
  - name: http
    port: 8080
    targetPort: 8080
  - name: metrics
    port: 9090
    targetPort: 9090
```

## Future Architecture Considerations

### Microservices Evolution

1. **Service Decomposition**: Split into auth, user, and admin services
2. **Event-Driven Architecture**: Domain events for service communication
3. **API Gateway**: Centralized routing and authentication
4. **Service Mesh**: Advanced networking and security

### Security Enhancements

1. **OAuth2/OpenID Connect**: Standard protocol support
2. **Multi-Factor Authentication**: Additional security layers
3. **Certificate-Based Auth**: PKI integration
4. **Hardware Security Modules**: Key management security

### Scalability Improvements

1. **Distributed Caching**: Redis cluster for permissions
2. **Database Sharding**: Horizontal database scaling
3. **Event Sourcing**: Audit trail and state reconstruction
4. **CQRS Pattern**: Separate read/write models