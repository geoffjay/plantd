# API Reference

## Overview

This document provides comprehensive API reference for the PlantD Identity Service, including all service interfaces, request/response formats, and usage examples.

## Service Interfaces

### Authentication Service

The `AuthService` provides core authentication functionality including login, logout, token management, and password operations.

#### Interface Definition

```go
type AuthService interface {
    Login(ctx context.Context, req *AuthRequest) (*AuthResponse, error)
    Logout(ctx context.Context, accessToken string) error
    ValidateToken(ctx context.Context, tokenString string, tokenType TokenType) (*CustomClaims, error)
    RefreshToken(ctx context.Context, req *RefreshRequest) (*TokenPair, error)
    ChangePassword(ctx context.Context, userID uint, currentPassword, newPassword string) error
    InitiatePasswordReset(ctx context.Context, req *PasswordResetInitiateRequest) error
    CompletePasswordReset(ctx context.Context, req *PasswordResetCompleteRequest) error
    GetSecurityStats(ctx context.Context) (map[string]interface{}, error)
}
```

#### Methods

##### Login

Authenticates a user with email/username and password.

```go
func (a *AuthService) Login(ctx context.Context, req *AuthRequest) (*AuthResponse, error)
```

**Request:**
```go
type AuthRequest struct {
    Identifier string `json:"identifier" validate:"required"`        // Email or username
    Password   string `json:"password" validate:"required"`          // User password
    IPAddress  string `json:"ip_address,omitempty"`                  // Client IP address
    UserAgent  string `json:"user_agent,omitempty"`                  // Client user agent
    RememberMe bool   `json:"remember_me,omitempty"`                 // Extended session
}
```

**Response:**
```go
type AuthResponse struct {
    Success    bool       `json:"success"`
    Message    string     `json:"message"`
    User       *models.User `json:"user,omitempty"`
    TokenPair  *TokenPair `json:"token_pair,omitempty"`
    ExpiresAt  time.Time  `json:"expires_at,omitempty"`
}

type TokenPair struct {
    AccessToken  string    `json:"access_token"`
    RefreshToken string    `json:"refresh_token"`
    TokenType    string    `json:"token_type"`
    ExpiresIn    int64     `json:"expires_in"`
}
```

**Example:**
```go
loginReq := &auth.AuthRequest{
    Identifier: "user@example.com",
    Password:   "MySecurePass123!",
    IPAddress:  "192.168.1.100",
    UserAgent:  "MyApp/1.0",
}

response, err := authService.Login(ctx, loginReq)
if err != nil {
    // Handle authentication error
}

accessToken := response.TokenPair.AccessToken
```

##### Token Validation

Validates and parses JWT tokens.

```go
func (a *AuthService) ValidateToken(ctx context.Context, tokenString string, tokenType TokenType) (*CustomClaims, error)
```

**Parameters:**
- `tokenString`: JWT token to validate
- `tokenType`: Type of token (Access, Refresh, Reset)

**Response:**
```go
type CustomClaims struct {
    UserID        uint     `json:"user_id"`
    Email         string   `json:"email"`
    Username      string   `json:"username"`
    Organizations []uint   `json:"organizations"`
    Roles         []string `json:"roles"`
    Permissions   []string `json:"permissions"`
    TokenType     string   `json:"token_type"`
    EmailVerified bool     `json:"email_verified"`
    IsActive      bool     `json:"is_active"`
    LastLoginAt   int64    `json:"last_login_at,omitempty"`
    jwt.RegisteredClaims
}
```

**Example:**
```go
claims, err := authService.ValidateToken(ctx, accessToken, auth.TokenTypeAccess)
if err != nil {
    // Token is invalid
}

userID := claims.UserID
permissions := claims.Permissions
```

##### Token Refresh

Generates new access token using refresh token.

```go
func (a *AuthService) RefreshToken(ctx context.Context, req *RefreshRequest) (*TokenPair, error)
```

**Request:**
```go
type RefreshRequest struct {
    RefreshToken string `json:"refresh_token" validate:"required"`
    IPAddress    string `json:"ip_address,omitempty"`
    UserAgent    string `json:"user_agent,omitempty"`
}
```

**Example:**
```go
refreshReq := &auth.RefreshRequest{
    RefreshToken: existingRefreshToken,
    IPAddress:    "192.168.1.100",
}

newTokens, err := authService.RefreshToken(ctx, refreshReq)
if err != nil {
    // Refresh failed, user needs to re-authenticate
}
```

##### Logout

Invalidates access token and adds it to blacklist.

```go
func (a *AuthService) Logout(ctx context.Context, accessToken string) error
```

**Example:**
```go
err := authService.Logout(ctx, accessToken)
if err != nil {
    // Handle logout error
}
```

##### Change Password

Changes user password with current password verification.

```go
func (a *AuthService) ChangePassword(ctx context.Context, userID uint, currentPassword, newPassword string) error
```

**Example:**
```go
err := authService.ChangePassword(ctx, userID, "currentPass", "newSecurePass123!")
if err != nil {
    // Handle password change error
}
```

### User Registration Service

Handles user registration, email verification, and profile management.

#### Interface Definition

```go
type RegistrationService interface {
    Register(ctx context.Context, req *RegistrationRequest) (*RegistrationResponse, error)
    VerifyEmail(ctx context.Context, req *EmailVerificationRequest) error
    ResendVerification(ctx context.Context, userID uint) error
    UpdateProfile(ctx context.Context, userID uint, req *ProfileUpdateRequest) error
}
```

#### Methods

##### Register

Registers a new user account.

```go
func (r *RegistrationService) Register(ctx context.Context, req *RegistrationRequest) (*RegistrationResponse, error)
```

**Request:**
```go
type RegistrationRequest struct {
    Email     string `json:"email" validate:"required,email"`
    Username  string `json:"username" validate:"required,min=3,max=50"`
    Password  string `json:"password" validate:"required"`
    FirstName string `json:"first_name" validate:"required,min=1,max=100"`
    LastName  string `json:"last_name" validate:"required,min=1,max=100"`
    IPAddress string `json:"ip_address,omitempty"`
    UserAgent string `json:"user_agent,omitempty"`
}
```

**Response:**
```go
type RegistrationResponse struct {
    Success             bool         `json:"success"`
    Message             string       `json:"message"`
    User                *models.User `json:"user,omitempty"`
    RequiresVerification bool         `json:"requires_verification"`
    VerificationSent    bool         `json:"verification_sent,omitempty"`
}
```

**Example:**
```go
regReq := &auth.RegistrationRequest{
    Email:     "newuser@example.com",
    Username:  "newuser",
    Password:  "SecurePassword123!",
    FirstName: "John",
    LastName:  "Doe",
    IPAddress: "192.168.1.100",
    UserAgent: "MyApp/1.0",
}

response, err := registrationService.Register(ctx, regReq)
if err != nil {
    // Handle registration error
}
```

##### Email Verification

Verifies user email with verification token.

```go
func (r *RegistrationService) VerifyEmail(ctx context.Context, req *EmailVerificationRequest) error
```

**Request:**
```go
type EmailVerificationRequest struct {
    Token     string `json:"token" validate:"required"`
    IPAddress string `json:"ip_address,omitempty"`
}
```

### RBAC Service

Provides role-based access control functionality.

#### Interface Definition

```go
type PermissionChecker interface {
    HasPermission(ctx context.Context, userID uint, permission Permission, orgID *uint) (bool, error)
    HasAnyPermission(ctx context.Context, userID uint, permissions []Permission, orgID *uint) (bool, error)
    HasAllPermissions(ctx context.Context, userID uint, permissions []Permission, orgID *uint) (bool, error)
    GetUserPermissions(ctx context.Context, userID uint, orgID *uint) ([]Permission, error)
    AssignRoleToUser(ctx context.Context, roleID, userID uint, orgID *uint) error
    RemoveRoleFromUser(ctx context.Context, roleID, userID uint, orgID *uint) error
    GetUserRoles(ctx context.Context, userID uint, orgID *uint) ([]models.Role, error)
}
```

#### Methods

##### HasPermission

Checks if user has specific permission.

```go
func (r *RBACService) HasPermission(ctx context.Context, userID uint, permission Permission, orgID *uint) (bool, error)
```

**Parameters:**
- `userID`: User ID to check
- `permission`: Permission to verify (e.g., "user:read")
- `orgID`: Optional organization ID for scoped permissions

**Example:**
```go
// Check global permission
hasPermission, err := rbacService.HasPermission(ctx, userID, auth.PermissionUserRead, nil)

// Check organization-scoped permission
orgID := uint(123)
hasOrgPermission, err := rbacService.HasPermission(ctx, userID, auth.PermissionOrganizationAdmin, &orgID)
```

##### HasAnyPermission

Checks if user has any of the specified permissions.

```go
func (r *RBACService) HasAnyPermission(ctx context.Context, userID uint, permissions []Permission, orgID *uint) (bool, error)
```

**Example:**
```go
requiredPerms := []auth.Permission{
    auth.PermissionUserRead,
    auth.PermissionUserAdmin,
}
hasAny, err := rbacService.HasAnyPermission(ctx, userID, requiredPerms, nil)
```

##### AssignRoleToUser

Assigns role to user with optional organization scope.

```go
func (r *RBACService) AssignRoleToUser(ctx context.Context, roleID, userID uint, orgID *uint) error
```

**Example:**
```go
// Assign global role
err := rbacService.AssignRoleToUser(ctx, adminRoleID, userID, nil)

// Assign organization-scoped role
orgID := uint(123)
err := rbacService.AssignRoleToUser(ctx, managerRoleID, userID, &orgID)
```

### Organization Membership Service

Manages organization membership and organization-scoped access.

#### Interface Definition

```go
type OrganizationMembershipService interface {
    AddUserToOrganization(ctx context.Context, userID, orgID uint, roleIDs []uint) error
    RemoveUserFromOrganization(ctx context.Context, userID, orgID uint) error
    IsUserMember(ctx context.Context, userID, orgID uint) (bool, error)
    GetUserOrganizations(ctx context.Context, userID uint) ([]models.Organization, error)
    GetOrganizationMembers(ctx context.Context, orgID uint) ([]models.User, error)
    UpdateUserRolesInOrganization(ctx context.Context, userID, orgID uint, roleIDs []uint) error
}
```

#### Methods

##### AddUserToOrganization

Adds user to organization with specified roles.

```go
func (o *OrganizationMembershipService) AddUserToOrganization(ctx context.Context, userID, orgID uint, roleIDs []uint) error
```

**Example:**
```go
roleIDs := []uint{memberRoleID, viewerRoleID}
err := orgMembershipService.AddUserToOrganization(ctx, userID, orgID, roleIDs)
```

##### GetUserOrganizations

Gets all organizations user belongs to.

```go
func (o *OrganizationMembershipService) GetUserOrganizations(ctx context.Context, userID uint) ([]models.Organization, error)
```

## Permission System

### Permission Constants

The system defines 43 granular permissions across 5 categories:

#### User Management (8 permissions)
```go
const (
    PermissionUserRead        Permission = "user:read"
    PermissionUserWrite       Permission = "user:write"
    PermissionUserDelete      Permission = "user:delete"
    PermissionUserList        Permission = "user:list"
    PermissionUserSearch      Permission = "user:search"
    PermissionUserAdmin       Permission = "user:admin"
    PermissionUserImpersonate Permission = "user:impersonate"
    PermissionUserExport      Permission = "user:export"
)
```

#### Organization Management (10 permissions)
```go
const (
    PermissionOrganizationRead         Permission = "organization:read"
    PermissionOrganizationWrite        Permission = "organization:write"
    PermissionOrganizationDelete       Permission = "organization:delete"
    PermissionOrganizationList         Permission = "organization:list"
    PermissionOrganizationAdmin        Permission = "organization:admin"
    PermissionOrganizationMemberAdd    Permission = "organization:member:add"
    PermissionOrganizationMemberRemove Permission = "organization:member:remove"
    PermissionOrganizationMemberList   Permission = "organization:member:list"
    PermissionOrganizationSettings     Permission = "organization:settings"
    PermissionOrganizationAudit        Permission = "organization:audit"
)
```

#### Role Management (10 permissions)
```go
const (
    PermissionRoleRead   Permission = "role:read"
    PermissionRoleWrite  Permission = "role:write"
    PermissionRoleDelete Permission = "role:delete"
    PermissionRoleList   Permission = "role:list"
    PermissionRoleAssign Permission = "role:assign"
    PermissionRoleRevoke Permission = "role:revoke"
    PermissionRoleAdmin  Permission = "role:admin"
    PermissionRoleCreate Permission = "role:create"
    PermissionRoleUpdate Permission = "role:update"
    PermissionRoleAudit  Permission = "role:audit"
)
```

#### Authentication & Session (7 permissions)
```go
const (
    PermissionAuthLogin          Permission = "auth:login"
    PermissionAuthLogout         Permission = "auth:logout"
    PermissionAuthPasswordChange Permission = "auth:password:change"
    PermissionAuthPasswordReset  Permission = "auth:password:reset"
    PermissionAuthTokenRefresh   Permission = "auth:token:refresh"
    PermissionAuthSessionList    Permission = "auth:session:list"
    PermissionAuthSessionRevoke  Permission = "auth:session:revoke"
)
```

#### System Administration (8 permissions)
```go
const (
    PermissionSystemAdmin       Permission = "system:admin"
    PermissionSystemRead        Permission = "system:read"
    PermissionSystemWrite       Permission = "system:write"
    PermissionSystemMonitor     Permission = "system:monitor"
    PermissionSystemAudit       Permission = "system:audit"
    PermissionSystemConfig      Permission = "system:config"
    PermissionSystemBackup      Permission = "system:backup"
    PermissionSystemMaintenance Permission = "system:maintenance"
)
```

### Permission Utilities

```go
// Validate permission
func IsValidPermission(permission Permission) bool

// Get permission category
func GetPermissionCategory(permission Permission) PermissionCategory

// Get all permissions
func GetAllPermissions() []Permission

// Get permissions by category
func GetPermissionsByCategory(category PermissionCategory) []Permission
```

## Authorization Middleware

The authorization middleware provides HTTP request-level authorization.

### Middleware Types

#### RequireAuth

Basic authentication middleware that validates JWT tokens.

```go
func (m *AuthorizationMiddleware) RequireAuth() gin.HandlerFunc
```

**Usage:**
```go
router.Use(authMiddleware.RequireAuth())
```

#### RequirePermission

Permission-based authorization middleware.

```go
func (m *AuthorizationMiddleware) RequirePermission(permission auth.Permission) gin.HandlerFunc
```

**Usage:**
```go
router.GET("/users", authMiddleware.RequirePermission(auth.PermissionUserList), getUsersHandler)
```

#### RequireRole

Role-based authorization middleware.

```go
func (m *AuthorizationMiddleware) RequireRole(roleName string) gin.HandlerFunc
```

**Usage:**
```go
adminGroup := router.Group("/admin")
adminGroup.Use(authMiddleware.RequireRole("admin"))
```

#### RequireResourceAccess

Resource-level authorization middleware.

```go
func (m *AuthorizationMiddleware) RequireResourceAccess(resourceType string, permission auth.Permission) gin.HandlerFunc
```

**Usage:**
```go
router.GET("/users/:id", authMiddleware.RequireResourceAccess("user", auth.PermissionUserRead), getUserHandler)
```

## Error Handling

### Error Types

#### Authentication Errors
```go
var (
    ErrInvalidCredentials = errors.New("invalid credentials")
    ErrAccountLocked      = errors.New("account is locked")
    ErrAccountInactive    = errors.New("account is inactive")
    ErrTokenExpired       = errors.New("token has expired")
    ErrTokenInvalid       = errors.New("token is invalid")
    ErrTokenBlacklisted   = errors.New("token has been revoked")
)
```

#### Authorization Errors
```go
var (
    ErrPermissionDenied    = errors.New("permission denied")
    ErrInsufficientRights = errors.New("insufficient rights")
    ErrInvalidPermission   = errors.New("invalid permission")
    ErrRoleNotFound        = errors.New("role not found")
)
```

#### Validation Errors
```go
var (
    ErrInvalidEmail       = errors.New("invalid email format")
    ErrWeakPassword      = errors.New("password does not meet policy requirements")
    ErrUserExists        = errors.New("user already exists")
    ErrInvalidInput      = errors.New("invalid input provided")
)
```

### Error Response Format

```go
type ErrorResponse struct {
    Error     string            `json:"error"`
    Message   string            `json:"message"`
    Code      string            `json:"code,omitempty"`
    Details   map[string]string `json:"details,omitempty"`
    RequestID string            `json:"request_id,omitempty"`
    Timestamp time.Time         `json:"timestamp"`
}
```

## Security Audit Events

### Event Structure

```go
type SecurityEvent struct {
    UserID        *uint             `json:"user_id,omitempty"`
    Email         string            `json:"email,omitempty"`
    EventType     string            `json:"event_type"`
    Success       bool              `json:"success"`
    FailureReason string            `json:"failure_reason,omitempty"`
    IPAddress     string            `json:"ip_address,omitempty"`
    UserAgent     string            `json:"user_agent,omitempty"`
    Timestamp     time.Time         `json:"timestamp"`
    Metadata      map[string]string `json:"metadata,omitempty"`
}
```

### Event Types

#### Authentication Events
- `login_success`
- `login_failure`
- `login_invalid_password`
- `login_account_locked`
- `logout`
- `token_refresh_success`
- `token_refresh_failure`

#### Authorization Events
- `permission_granted`
- `permission_denied`
- `role_assigned`
- `role_removed`
- `organization_access_granted`
- `organization_access_denied`

#### User Management Events
- `registration_success`
- `registration_failure`
- `password_change_success`
- `password_change_failure`
- `password_reset_initiated`
- `email_verification_success`
- `profile_update_success`

## Configuration Reference

### Service Configuration

```go
type Config struct {
    Env      string           `json:"env"`
    Database DatabaseConfig   `json:"database"`
    Server   ServerConfig     `json:"server"`
    Security SecurityConfig   `json:"security"`
    Log      LogConfig        `json:"log"`
    Service  ServiceConfig    `json:"service"`
}
```

### Authentication Configuration

```go
type AuthConfig struct {
    JWTSecret           string        `json:"jwt_secret"`
    JWTRefreshSecret    string        `json:"jwt_refresh_secret"`
    JWTExpiration       time.Duration `json:"jwt_expiration"`
    RefreshExpiration   time.Duration `json:"refresh_expiration"`
    JWTIssuer          string        `json:"jwt_issuer"`
    BcryptCost         int           `json:"bcrypt_cost"`
    PasswordMinLength  int           `json:"password_min_length"`
    PasswordMaxLength  int           `json:"password_max_length"`
    RequireUppercase   bool          `json:"require_uppercase"`
    RequireLowercase   bool          `json:"require_lowercase"`
    RequireNumbers     bool          `json:"require_numbers"`
    RequireSpecialChars bool         `json:"require_special_chars"`
    RateLimitRPS       int           `json:"rate_limit_rps"`
    RateLimitBurst     int           `json:"rate_limit_burst"`
    MaxFailedAttempts  int           `json:"max_failed_attempts"`
    LockoutDuration    time.Duration `json:"lockout_duration"`
}
```

## Usage Examples

### Complete Integration Example

```go
package main

import (
    "context"
    "log"
    
    "github.com/gin-gonic/gin"
    "github.com/geoffjay/plantd/identity/internal/auth"
    "github.com/geoffjay/plantd/identity/internal/config"
    "github.com/geoffjay/plantd/identity/internal/models"
    "github.com/geoffjay/plantd/identity/internal/repositories"
    "github.com/geoffjay/plantd/identity/internal/services"
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
    orgRepo := repositories.NewOrganizationRepository(db)
    roleRepo := repositories.NewRoleRepository(db)
    
    // Initialize services
    userService := services.NewUserService(userRepo, orgRepo, roleRepo)
    authService := auth.NewAuthService(cfg.ToAuthConfig(), userRepo, userService, logger)
    rbacService := auth.NewRBACService(userRepo, roleRepo, logger)
    orgMembershipService := auth.NewOrganizationMembershipService(
        userRepo, orgRepo, roleRepo, rbacService, logger)
    
    // Initialize middleware
    authMiddleware := auth.NewAuthorizationMiddleware(authService, rbacService, logger)
    
    // Setup routes
    router := setupRoutes(authService, rbacService, authMiddleware)
    
    // Start server
    log.Println("Starting identity service on :8080")
    router.Run(":8080")
}

func setupRoutes(
    authService *auth.AuthService,
    rbacService *auth.RBACService,
    authMiddleware *auth.AuthorizationMiddleware,
) *gin.Engine {
    router := gin.Default()
    
    // Public routes
    router.POST("/auth/login", loginHandler(authService))
    router.POST("/auth/register", registerHandler(registrationService))
    
    // Protected routes
    api := router.Group("/api")
    api.Use(authMiddleware.RequireAuth())
    {
        // User profile
        api.GET("/profile", getProfileHandler)
        api.PUT("/profile", updateProfileHandler)
        
        // User management
        users := api.Group("/users")
        users.GET("", authMiddleware.RequirePermission(auth.PermissionUserList), listUsersHandler)
        users.POST("", authMiddleware.RequirePermission(auth.PermissionUserWrite), createUserHandler)
        users.GET("/:id", authMiddleware.RequireResourceAccess("user", auth.PermissionUserRead), getUserHandler)
        users.PUT("/:id", authMiddleware.RequireResourceAccess("user", auth.PermissionUserWrite), updateUserHandler)
        users.DELETE("/:id", authMiddleware.RequireResourceAccess("user", auth.PermissionUserDelete), deleteUserHandler)
        
        // Organization management
        orgs := api.Group("/organizations")
        orgs.GET("", authMiddleware.RequirePermission(auth.PermissionOrganizationList), listOrgsHandler)
        orgs.POST("", authMiddleware.RequirePermission(auth.PermissionOrganizationWrite), createOrgHandler)
        
        // Organization-scoped routes
        org := orgs.Group("/:orgId")
        {
            org.GET("", authMiddleware.RequirePermission(auth.PermissionOrganizationRead), getOrgHandler)
            org.PUT("", authMiddleware.RequirePermission(auth.PermissionOrganizationWrite), updateOrgHandler)
            org.GET("/members", authMiddleware.RequirePermission(auth.PermissionOrganizationMemberList), getOrgMembersHandler)
            org.POST("/members", authMiddleware.RequirePermission(auth.PermissionOrganizationMemberAdd), addOrgMemberHandler)
            org.DELETE("/members/:userId", authMiddleware.RequirePermission(auth.PermissionOrganizationMemberRemove), removeOrgMemberHandler)
        }
        
        // Admin routes
        admin := api.Group("/admin")
        admin.Use(authMiddleware.RequireRole("admin"))
        {
            admin.GET("/stats", getStatsHandler)
            admin.GET("/audit", authMiddleware.RequirePermission(auth.PermissionSystemAudit), getAuditHandler)
            admin.GET("/health", getHealthHandler)
        }
    }
    
    return router
}
```

This API reference provides comprehensive documentation for integrating and using the PlantD Identity Service. For more detailed examples and usage patterns, refer to the [Examples & Usage Guide](examples.md).