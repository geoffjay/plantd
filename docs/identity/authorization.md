# Authorization & RBAC System

## Overview

The PlantD Identity Service implements a comprehensive Role-Based Access Control (RBAC) system with organization-scoped permissions, granular permission management, and HTTP middleware for request authorization.

## Core Components

### 1. Permission System

#### Permission Structure
The system defines 43 granular permissions across 5 categories:

```go
// User Management Permissions (8)
const (
    PermissionUserRead   Permission = "user:read"
    PermissionUserWrite  Permission = "user:write" 
    PermissionUserDelete Permission = "user:delete"
    PermissionUserList   Permission = "user:list"
    PermissionUserSearch Permission = "user:search"
    PermissionUserAdmin  Permission = "user:admin"
    PermissionUserImpersonate Permission = "user:impersonate"
    PermissionUserExport Permission = "user:export"
)

// Organization Management Permissions (10)
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

// Role Management Permissions (10)
const (
    PermissionRoleRead      Permission = "role:read"
    PermissionRoleWrite     Permission = "role:write"
    PermissionRoleDelete    Permission = "role:delete"
    PermissionRoleList      Permission = "role:list"
    PermissionRoleAssign    Permission = "role:assign"
    PermissionRoleRevoke    Permission = "role:revoke"
    PermissionRoleAdmin     Permission = "role:admin"
    PermissionRoleCreate    Permission = "role:create"
    PermissionRoleUpdate    Permission = "role:update"
    PermissionRoleAudit     Permission = "role:audit"
)

// Authentication & Session Permissions (7)
const (
    PermissionAuthLogin          Permission = "auth:login"
    PermissionAuthLogout         Permission = "auth:logout"
    PermissionAuthPasswordChange Permission = "auth:password:change"
    PermissionAuthPasswordReset  Permission = "auth:password:reset"
    PermissionAuthTokenRefresh   Permission = "auth:token:refresh"
    PermissionAuthSessionList    Permission = "auth:session:list"
    PermissionAuthSessionRevoke  Permission = "auth:session:revoke"
)

// System Administration Permissions (8)
const (
    PermissionSystemAdmin    Permission = "system:admin"
    PermissionSystemRead     Permission = "system:read"
    PermissionSystemWrite    Permission = "system:write"
    PermissionSystemMonitor  Permission = "system:monitor"
    PermissionSystemAudit    Permission = "system:audit"
    PermissionSystemConfig   Permission = "system:config"
    PermissionSystemBackup   Permission = "system:backup"
    PermissionSystemMaintenance Permission = "system:maintenance"
)
```

#### Permission Categories
Each permission belongs to a category for organization and management:

```go
type PermissionCategory string

const (
    CategoryUser         PermissionCategory = "user"
    CategoryOrganization PermissionCategory = "organization" 
    CategoryRole         PermissionCategory = "role"
    CategoryAuth         PermissionCategory = "auth"
    CategorySystem       PermissionCategory = "system"
)
```

#### Permission Validation
```go
func IsValidPermission(permission Permission) bool
func GetPermissionCategory(permission Permission) PermissionCategory
func GetAllPermissions() []Permission
func GetPermissionsByCategory(category PermissionCategory) []Permission
```

### 2. Authorization Context

The system uses an authorization context to determine permission scope:

```go
type AuthorizationContext struct {
    UserID         uint   `json:"user_id"`
    OrganizationID *uint  `json:"organization_id,omitempty"`
    ResourceID     *uint  `json:"resource_id,omitempty"`
    ResourceType   string `json:"resource_type,omitempty"`
    IPAddress      string `json:"ip_address,omitempty"`
    UserAgent      string `json:"user_agent,omitempty"`
}
```

### 3. RBAC Service

#### Core Interface
```go
type PermissionChecker interface {
    HasPermission(ctx context.Context, userID uint, permission Permission, orgID *uint) (bool, error)
    HasAnyPermission(ctx context.Context, userID uint, permissions []Permission, orgID *uint) (bool, error)
    HasAllPermissions(ctx context.Context, userID uint, permissions []Permission, orgID *uint) (bool, error)
    GetUserPermissions(ctx context.Context, userID uint, orgID *uint) ([]Permission, error)
}
```

#### Implementation Features
- **Permission Caching**: 5-minute cache for performance optimization
- **Organization Scoping**: Permissions can be global or organization-specific
- **Role Inheritance**: Users inherit permissions from assigned roles
- **Audit Logging**: All permission checks are logged for security auditing

#### Core Methods

**Permission Checking**
```go
func (r *RBACService) HasPermission(ctx context.Context, userID uint, permission Permission, orgID *uint) (bool, error)
```
- Checks if user has specific permission
- Supports organization scoping
- Uses cached permissions for performance
- Logs permission checks for auditing

**Multiple Permission Checks**
```go
func (r *RBACService) HasAnyPermission(ctx context.Context, userID uint, permissions []Permission, orgID *uint) (bool, error)
func (r *RBACService) HasAllPermissions(ctx context.Context, userID uint, permissions []Permission, orgID *uint) (bool, error)
```

**Role Management**
```go
func (r *RBACService) AssignRoleToUser(ctx context.Context, roleID, userID uint, orgID *uint) error
func (r *RBACService) RemoveRoleFromUser(ctx context.Context, roleID, userID uint, orgID *uint) error
func (r *RBACService) GetUserRoles(ctx context.Context, userID uint, orgID *uint) ([]models.Role, error)
```

### 4. Organization-Based Permissions

#### Organization Membership Service
```go
type OrganizationMembershipService struct {
    userRepo repositories.UserRepository
    orgRepo  repositories.OrganizationRepository
    roleRepo repositories.RoleRepository
    rbac     *RBACService
    logger   *logrus.Logger
}
```

#### Key Features
- **Organization Isolation**: Permissions are scoped to specific organizations
- **Multi-tenancy Support**: Users can belong to multiple organizations
- **Role Inheritance**: Organization-specific role assignments
- **Member Management**: Add/remove users from organizations

#### Core Methods

**Membership Management**
```go
func (o *OrganizationMembershipService) AddUserToOrganization(ctx context.Context, userID, orgID uint, roleIDs []uint) error
func (o *OrganizationMembershipService) RemoveUserFromOrganization(ctx context.Context, userID, orgID uint) error
func (o *OrganizationMembershipService) IsUserMember(ctx context.Context, userID, orgID uint) (bool, error)
```

**Organization Context**
```go
func (o *OrganizationMembershipService) GetUserOrganizations(ctx context.Context, userID uint) ([]models.Organization, error)
func (o *OrganizationMembershipService) GetOrganizationMembers(ctx context.Context, orgID uint) ([]models.User, error)
```

### 5. Authorization Middleware

The system provides HTTP middleware for request-level authorization:

#### Middleware Types

**Basic Authentication**
```go
func (m *AuthorizationMiddleware) RequireAuth() gin.HandlerFunc
```
- Validates JWT tokens
- Extracts user context
- Sets user information in request context

**Permission-Based Authorization**
```go
func (m *AuthorizationMiddleware) RequirePermission(permission auth.Permission) gin.HandlerFunc
```
- Checks specific permission
- Supports organization context from URL parameters
- Returns 403 Forbidden if permission denied

**Role-Based Authorization**
```go
func (m *AuthorizationMiddleware) RequireRole(roleName string) gin.HandlerFunc
```
- Checks if user has specific role
- Supports organization scoping
- Useful for simpler authorization scenarios

**Resource-Based Authorization**
```go
func (m *AuthorizationMiddleware) RequireResourceAccess(resourceType string, permission auth.Permission) gin.HandlerFunc
```
- Advanced resource-level authorization
- Extracts resource ID from URL parameters
- Checks permission in context of specific resource

#### Usage Examples

**Basic Route Protection**
```go
// Require authentication
router.Use(authMiddleware.RequireAuth())

// Require specific permission
router.GET("/users", authMiddleware.RequirePermission(auth.PermissionUserList), getUsersHandler)

// Require role
router.GET("/admin", authMiddleware.RequireRole("admin"), adminHandler)

// Resource-based authorization
router.GET("/users/:id", authMiddleware.RequireResourceAccess("user", auth.PermissionUserRead), getUserHandler)
```

**Organization-Scoped Routes**
```go
orgGroup := router.Group("/org/:orgId")
orgGroup.Use(authMiddleware.RequireAuth())
{
    // Organization member list - requires org-scoped permission
    orgGroup.GET("/members", 
        authMiddleware.RequirePermission(auth.PermissionOrganizationMemberList), 
        getOrgMembersHandler)
    
    // Organization admin functions
    orgGroup.POST("/members", 
        authMiddleware.RequirePermission(auth.PermissionOrganizationMemberAdd),
        addOrgMemberHandler)
}
```

### 6. Security Audit Logging

All authorization events are logged with detailed context:

```go
type AuditEvent struct {
    UserID       uint      `json:"user_id"`
    Action       string    `json:"action"`
    Resource     string    `json:"resource,omitempty"`
    ResourceID   *uint     `json:"resource_id,omitempty"`
    Permission   string    `json:"permission,omitempty"`
    Authorized   bool      `json:"authorized"`
    Reason       string    `json:"reason,omitempty"`
    IPAddress    string    `json:"ip_address,omitempty"`
    UserAgent    string    `json:"user_agent,omitempty"`
    Timestamp    time.Time `json:"timestamp"`
    Metadata     map[string]interface{} `json:"metadata,omitempty"`
}
```

#### Logged Events
- Permission checks (granted/denied)
- Role assignments/removals
- Organization membership changes
- Authentication middleware decisions
- Resource access attempts

## API Usage Examples

### Basic Permission Checking
```go
// Check if user can read other users
canRead, err := rbacService.HasPermission(ctx, userID, auth.PermissionUserRead, nil)
if err != nil {
    return err
}
if !canRead {
    return errors.New("permission denied")
}

// Check organization-scoped permission
orgID := uint(123)
canManageOrg, err := rbacService.HasPermission(ctx, userID, auth.PermissionOrganizationAdmin, &orgID)
```

### Multiple Permission Checks
```go
// User needs any of these permissions
requiredPerms := []auth.Permission{
    auth.PermissionUserRead,
    auth.PermissionUserAdmin,
}
hasAny, err := rbacService.HasAnyPermission(ctx, userID, requiredPerms, nil)

// User needs all of these permissions
requiredPerms := []auth.Permission{
    auth.PermissionUserWrite,
    auth.PermissionUserDelete,
}
hasAll, err := rbacService.HasAllPermissions(ctx, userID, requiredPerms, nil)
```

### Role Management
```go
// Assign role to user globally
err := rbacService.AssignRoleToUser(ctx, adminRoleID, userID, nil)

// Assign role to user within organization
orgID := uint(123)
err := rbacService.AssignRoleToUser(ctx, managerRoleID, userID, &orgID)

// Get user's roles
roles, err := rbacService.GetUserRoles(ctx, userID, &orgID)
```

### Organization Membership
```go
// Add user to organization with roles
roleIDs := []uint{memberRoleID, viewerRoleID}
err := orgMembershipService.AddUserToOrganization(ctx, userID, orgID, roleIDs)

// Check if user is member
isMember, err := orgMembershipService.IsUserMember(ctx, userID, orgID)

// Get user's organizations
orgs, err := orgMembershipService.GetUserOrganizations(ctx, userID)
```

### HTTP Middleware Usage
```go
func setupRoutes(router *gin.Engine, authMiddleware *auth.AuthorizationMiddleware) {
    // Public routes (no auth required)
    router.POST("/auth/login", loginHandler)
    router.POST("/auth/register", registerHandler)
    
    // Protected routes
    api := router.Group("/api")
    api.Use(authMiddleware.RequireAuth())
    {
        // User management routes
        users := api.Group("/users")
        users.GET("", authMiddleware.RequirePermission(auth.PermissionUserList), listUsersHandler)
        users.POST("", authMiddleware.RequirePermission(auth.PermissionUserWrite), createUserHandler)
        users.GET("/:id", authMiddleware.RequireResourceAccess("user", auth.PermissionUserRead), getUserHandler)
        users.PUT("/:id", authMiddleware.RequireResourceAccess("user", auth.PermissionUserWrite), updateUserHandler)
        users.DELETE("/:id", authMiddleware.RequireResourceAccess("user", auth.PermissionUserDelete), deleteUserHandler)
        
        // Organization routes
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
        }
        
        // Admin-only routes
        admin := api.Group("/admin")
        admin.Use(authMiddleware.RequireRole("admin"))
        {
            admin.GET("/stats", getStatsHandler)
            admin.GET("/audit", authMiddleware.RequirePermission(auth.PermissionSystemAudit), getAuditHandler)
        }
    }
}
```

## Configuration

### Role Definitions
Roles can be defined in configuration or created via API:

```yaml
roles:
  - name: "admin"
    description: "System administrator"
    permissions:
      - "system:admin"
      - "user:admin"
      - "organization:admin"
      - "role:admin"
      
  - name: "manager"
    description: "Organization manager"
    permissions:
      - "organization:admin"
      - "organization:member:add"
      - "organization:member:remove"
      - "user:read"
      - "user:list"
      
  - name: "user"
    description: "Regular user"
    permissions:
      - "auth:login"
      - "auth:logout"
      - "auth:password:change"
      - "user:read"  # Own profile only
```

### Permission Caching
```yaml
rbac:
  cache_ttl: 300  # 5 minutes
  enable_audit_logging: true
  organization_isolation: true
```

## Best Practices

### Permission Design
1. **Principle of Least Privilege**: Grant minimum necessary permissions
2. **Granular Permissions**: Use specific permissions rather than broad admin rights
3. **Organization Scoping**: Implement proper multi-tenancy boundaries
4. **Resource-Level Authorization**: Check permissions for specific resources when possible

### Role Management
1. **Role Hierarchy**: Design clear role hierarchies within organizations
2. **Default Roles**: Assign appropriate default roles to new users
3. **Role Auditing**: Regularly review role assignments
4. **Temporary Permissions**: Use time-limited role assignments when needed

### Performance Optimization
1. **Permission Caching**: Enable caching for frequently checked permissions
2. **Batch Checks**: Use bulk permission checks when validating multiple permissions
3. **Database Indexing**: Ensure proper indexing on user-role-organization relationships
4. **Middleware Placement**: Place authorization middleware strategically to avoid unnecessary checks

### Security Considerations
1. **Input Validation**: Validate all permission and role identifiers
2. **Audit Logging**: Enable comprehensive audit logging for compliance
3. **Regular Reviews**: Conduct regular permission and role audits
4. **Defense in Depth**: Combine RBAC with other security measures

## Troubleshooting

### Common Issues

**"Permission denied" errors**
- Verify user has required role assignment
- Check if permission is organization-scoped correctly
- Review permission cache TTL settings
- Validate permission name spelling

**Performance issues**
- Enable permission caching
- Review database queries and indexing
- Consider bulk permission checks
- Monitor cache hit rates

**Authorization middleware not working**
- Check middleware order in router setup
- Verify JWT token validation is working
- Ensure user context is properly set
- Review route parameter extraction

**Organization scoping problems**
- Verify organization ID extraction from request
- Check user's organization membership
- Review organization-scoped role assignments
- Validate organization isolation logic 
