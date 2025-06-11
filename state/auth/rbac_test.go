package auth

import (
	"context"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// Test helper functions
func createTestUserContext(permissions []string) *UserContext {
	userPerms := make([]Permission, len(permissions))
	for i, perm := range permissions {
		userPerms[i] = Permission{Name: perm}
	}

	return &UserContext{
		UserID:      1,
		UserEmail:   "test@example.com",
		Username:    "test",
		Permissions: userPerms,
		ValidUntil:  time.Now().Add(time.Hour),
	}
}

func createTestAccessChecker() *AccessChecker {
	logger := log.New()
	logger.SetLevel(log.DebugLevel)

	return NewAccessChecker(&AccessCheckerConfig{
		IdentityClient: nil, // Tests will work without identity client for basic functionality
		CacheTTL:       5 * time.Minute,
		Logger:         logger,
	})
}

func TestPermissionUtils(t *testing.T) {
	utils := NewPermissionUtils()

	t.Run("ValidatePermission", func(t *testing.T) {
		tests := []struct {
			name        string
			permission  string
			expectError bool
		}{
			{"Valid permission", "state:data:read", false},
			{"Valid admin permission", "state:admin:full", false},
			{"Empty permission", "", true},
			{"Invalid format", "invalid", true},
			{"Wrong service", "broker:data:read", true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := utils.ValidatePermission(tt.permission)
				if tt.expectError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})

	t.Run("IsWildcardPermission", func(t *testing.T) {
		assert.True(t, utils.IsWildcardPermission("state:*"))
		assert.True(t, utils.IsWildcardPermission("state:data:*"))
		assert.False(t, utils.IsWildcardPermission("state:data:read"))
	})

	t.Run("IsGlobalPermission", func(t *testing.T) {
		assert.True(t, utils.IsGlobalPermission(StateScopeCreate))
		assert.True(t, utils.IsGlobalPermission(StateAdminFull))
		assert.False(t, utils.IsGlobalPermission("state:scope:test:read"))
	})

	t.Run("CreateScopedPermission", func(t *testing.T) {
		scoped := utils.CreateScopedPermission(StateDataRead, "test-scope")
		assert.Equal(t, "state:scope:test-scope:read", scoped)

		// Test with empty scope
		global := utils.CreateScopedPermission(StateDataRead, "")
		assert.Equal(t, StateDataRead, global)
	})
}

func TestPermissionInheritance(t *testing.T) {
	inheritance := NewPermissionInheritance()

	t.Run("GetImpliedPermissions", func(t *testing.T) {
		// Test admin permission implies others
		implied := inheritance.GetImpliedPermissions(StateAdminFull)
		assert.Contains(t, implied, StateDataRead)
		assert.Contains(t, implied, StateDataWrite)
		assert.Contains(t, implied, StateScopeCreate)

		// Test system admin implies admin
		systemImplied := inheritance.GetImpliedPermissions(StateSystemAdmin)
		assert.Contains(t, systemImplied, StateAdminFull)

		// Test write implies read
		writeImplied := inheritance.GetImpliedPermissions(StateDataWrite)
		assert.Contains(t, writeImplied, StateDataRead)
	})

	t.Run("HasImpliedPermission", func(t *testing.T) {
		// Direct permission match
		assert.True(t, inheritance.HasImpliedPermission(StateDataRead, StateDataRead))

		// Admin implies other permissions
		assert.True(t, inheritance.HasImpliedPermission(StateAdminFull, StateDataRead))
		assert.True(t, inheritance.HasImpliedPermission(StateAdminFull, StateDataWrite))

		// Write implies read
		assert.True(t, inheritance.HasImpliedPermission(StateDataWrite, StateDataRead))

		// Read does not imply write
		assert.False(t, inheritance.HasImpliedPermission(StateDataRead, StateDataWrite))

		// Wildcard permissions
		assert.True(t, inheritance.HasImpliedPermission("state:*", StateDataRead))
		assert.True(t, inheritance.HasImpliedPermission("state:data:*", StateDataRead))
	})
}

func TestAccessChecker(t *testing.T) {
	checker := createTestAccessChecker()

	t.Run("AdminAccess", func(t *testing.T) {
		// User with admin permission
		adminUser := createTestUserContext([]string{StateAdminFull})
		err := checker.CheckScopeAccess(adminUser, StateDataRead, "test-scope")
		assert.NoError(t, err)

		// User with system admin permission
		systemAdminUser := createTestUserContext([]string{StateSystemAdmin})
		err = checker.CheckScopeAccess(systemAdminUser, StateDataWrite, "test-scope")
		assert.NoError(t, err)
	})

	t.Run("GlobalPermissions", func(t *testing.T) {
		// User with global read permission
		readUser := createTestUserContext([]string{StateDataRead})
		err := checker.CheckScopeAccess(readUser, StateDataRead, "any-scope")
		assert.NoError(t, err)

		// User without permission
		noPermUser := createTestUserContext([]string{})
		err = checker.CheckScopeAccess(noPermUser, StateDataRead, "any-scope")
		assert.Error(t, err)
	})

	t.Run("ScopedPermissions", func(t *testing.T) {
		// User with scoped read permission
		scopedUser := createTestUserContext([]string{"state:scope:test-scope:read"})
		err := checker.CheckScopeAccess(scopedUser, StateDataRead, "test-scope")
		assert.NoError(t, err)

		// Same user on different scope should fail
		err = checker.CheckScopeAccess(scopedUser, StateDataRead, "other-scope")
		assert.Error(t, err)
	})

	t.Run("ServiceOwnership", func(t *testing.T) {
		// Register scope ownership
		checker.RegisterScopeOwnership("owned-scope", "test-service")

		// Test ownership registration
		owner, exists := checker.GetScopeOwner("owned-scope")
		assert.True(t, exists)
		assert.Equal(t, "test-service", owner)

		// Test unregistering ownership
		checker.UnregisterScopeOwnership("owned-scope")
		_, exists = checker.GetScopeOwner("owned-scope")
		assert.False(t, exists)
	})

	t.Run("WildcardPermissions", func(t *testing.T) {
		// User with wildcard permission
		wildcardUser := createTestUserContext([]string{"state:*"})
		err := checker.CheckScopeAccess(wildcardUser, StateDataRead, "any-scope")
		assert.NoError(t, err)

		err = checker.CheckScopeAccess(wildcardUser, StateDataWrite, "any-scope")
		assert.NoError(t, err)
	})
}

func TestRoleManager(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.DebugLevel)

	roleManager := NewRoleManager(&RoleManagerConfig{
		IdentityClient: nil, // Tests will work without identity client for basic functionality
		Logger:         logger,
	})

	t.Run("ValidateRoleDefinition", func(t *testing.T) {
		validRole := &RoleDefinition{
			Name:        "state-test",
			Description: "Test role",
			Permissions: []string{StateDataRead},
			Scope:       "organization",
		}

		err := roleManager.ValidateRoleDefinition(validRole)
		assert.NoError(t, err)

		// Invalid name
		invalidRole := &RoleDefinition{
			Name:        "invalid-name",
			Description: "Test role",
			Permissions: []string{StateDataRead},
			Scope:       "organization",
		}
		err = roleManager.ValidateRoleDefinition(invalidRole)
		assert.Error(t, err)

		// Invalid scope
		invalidScope := &RoleDefinition{
			Name:        "state-test",
			Description: "Test role",
			Permissions: []string{StateDataRead},
			Scope:       "invalid-scope",
		}
		err = roleManager.ValidateRoleDefinition(invalidScope)
		assert.Error(t, err)
	})

	t.Run("GetRolePermissions", func(t *testing.T) {
		permissions, err := roleManager.GetRolePermissions(context.Background(), "state-developer")
		assert.NoError(t, err)
		assert.Contains(t, permissions, StateDataRead)
		assert.Contains(t, permissions, StateDataWrite)

		// Non-existent role
		_, err = roleManager.GetRolePermissions(context.Background(), "non-existent")
		assert.Error(t, err)
	})

	t.Run("ValidateRoleAssignment", func(t *testing.T) {
		// Valid assignment - organization role with org context
		orgID := uint(123)
		err := roleManager.ValidateRoleAssignment(context.Background(), "user@example.com", "state-developer", &orgID)
		assert.NoError(t, err)

		// Global role with org context should fail
		err = roleManager.ValidateRoleAssignment(context.Background(), "user@example.com", "state-system-admin", &orgID)
		assert.Error(t, err)

		// Global role without org context should succeed
		err = roleManager.ValidateRoleAssignment(context.Background(), "user@example.com", "state-system-admin", nil)
		assert.NoError(t, err)

		// Organization role without org context should fail
		err = roleManager.ValidateRoleAssignment(context.Background(), "user@example.com", "state-admin", nil)
		assert.Error(t, err)
	})
}

func BenchmarkPermissionCheck(b *testing.B) {
	checker := createTestAccessChecker()
	user := createTestUserContext([]string{StateDataRead, StateDataWrite, StateAdminFull})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = checker.CheckScopeAccess(user, StateDataRead, "benchmark-scope")
	}
}
