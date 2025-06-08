package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRole_TableName(t *testing.T) {
	role := Role{}
	assert.Equal(t, "roles", role.TableName())
}

func TestRole_HasPermission(t *testing.T) {
	tests := []struct {
		name        string
		permissions string
		permission  string
		expected    bool
	}{
		{
			name:        "has permission",
			permissions: `["read", "write", "delete"]`,
			permission:  "read",
			expected:    true,
		},
		{
			name:        "does not have permission",
			permissions: `["read", "write"]`,
			permission:  "delete",
			expected:    false,
		},
		{
			name:        "empty permissions",
			permissions: `[]`,
			permission:  "read",
			expected:    false,
		},
		{
			name:        "invalid JSON",
			permissions: `invalid json`,
			permission:  "read",
			expected:    false,
		},
		{
			name:        "empty permission check",
			permissions: `["read", "write"]`,
			permission:  "",
			expected:    false,
		},
		{
			name:        "case sensitive",
			permissions: `["READ", "WRITE"]`,
			permission:  "read",
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			role := &Role{Permissions: tt.permissions}
			assert.Equal(t, tt.expected, role.HasPermission(tt.permission))
		})
	}
}

func TestRole_GetPermissions(t *testing.T) {
	tests := []struct {
		name        string
		permissions string
		expected    []string
		shouldError bool
	}{
		{
			name:        "valid permissions",
			permissions: `["read", "write", "delete"]`,
			expected:    []string{"read", "write", "delete"},
			shouldError: false,
		},
		{
			name:        "empty permissions",
			permissions: `[]`,
			expected:    []string{},
			shouldError: false,
		},
		{
			name:        "invalid JSON",
			permissions: `invalid json`,
			expected:    nil,
			shouldError: true,
		},
		{
			name:        "null JSON",
			permissions: `null`,
			expected:    nil,
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			role := &Role{Permissions: tt.permissions}
			result, err := role.GetPermissions()

			if tt.shouldError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestRole_AddPermission(t *testing.T) {
	tests := []struct {
		name            string
		initialPerms    string
		permissionToAdd string
		expectedPerms   []string
		shouldError     bool
	}{
		{
			name:            "add to empty permissions",
			initialPerms:    `[]`,
			permissionToAdd: "read",
			expectedPerms:   []string{"read"},
			shouldError:     false,
		},
		{
			name:            "add to existing permissions",
			initialPerms:    `["read"]`,
			permissionToAdd: "write",
			expectedPerms:   []string{"read", "write"},
			shouldError:     false,
		},
		{
			name:            "add duplicate permission",
			initialPerms:    `["read", "write"]`,
			permissionToAdd: "read",
			expectedPerms:   []string{"read", "write"},
			shouldError:     false,
		},
		{
			name:            "invalid initial JSON",
			initialPerms:    `invalid json`,
			permissionToAdd: "read",
			expectedPerms:   nil,
			shouldError:     true,
		},
		{
			name:            "empty permission",
			initialPerms:    `["read"]`,
			permissionToAdd: "",
			expectedPerms:   []string{"read"},
			shouldError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			role := &Role{Permissions: tt.initialPerms}
			err := role.AddPermission(tt.permissionToAdd)

			if tt.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.expectedPerms != nil {
					perms, err := role.GetPermissions()
					assert.NoError(t, err)
					assert.ElementsMatch(t, tt.expectedPerms, perms)
				}
			}
		})
	}
}

func TestRole_RemovePermission(t *testing.T) {
	tests := []struct {
		name               string
		initialPerms       string
		permissionToRemove string
		expectedPerms      []string
		shouldError        bool
	}{
		{
			name:               "remove existing permission",
			initialPerms:       `["read", "write", "delete"]`,
			permissionToRemove: "write",
			expectedPerms:      []string{"read", "delete"},
			shouldError:        false,
		},
		{
			name:               "remove non-existing permission",
			initialPerms:       `["read", "write"]`,
			permissionToRemove: "delete",
			expectedPerms:      []string{"read", "write"},
			shouldError:        false,
		},
		{
			name:               "remove from empty permissions",
			initialPerms:       `[]`,
			permissionToRemove: "read",
			expectedPerms:      []string{},
			shouldError:        false,
		},
		{
			name:               "invalid initial JSON",
			initialPerms:       `invalid json`,
			permissionToRemove: "read",
			expectedPerms:      nil,
			shouldError:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			role := &Role{Permissions: tt.initialPerms}
			err := role.RemovePermission(tt.permissionToRemove)

			if tt.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.expectedPerms != nil {
					perms, err := role.GetPermissions()
					assert.NoError(t, err)
					assert.ElementsMatch(t, tt.expectedPerms, perms)
				}
			}
		})
	}
}

func TestRole_DatabaseConstraints(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	t.Run("name must be unique within scope", func(t *testing.T) {
		// Create first role
		role1 := createTestRole(t, db, func(r *Role) {
			r.Name = "Admin"
			r.Scope = RoleScopeGlobal
		})
		assert.NotZero(t, role1.ID)

		// Try to create second role with same name and scope
		role2 := &Role{
			Name:        "Admin",
			Description: "Different description",
			Permissions: `["read"]`,
			Scope:       RoleScopeGlobal,
		}

		err := db.Create(role2).Error
		assert.Error(t, err)
	})

	t.Run("same name allowed for different scopes", func(t *testing.T) {
		// Create role with global scope
		role1 := createTestRole(t, db, func(r *Role) {
			r.Name = "Manager"
			r.Scope = RoleScopeGlobal
		})
		assert.NotZero(t, role1.ID)

		// Create role with organization scope (should be allowed)
		role2 := &Role{
			Name:        "Manager",
			Description: "Organization manager",
			Permissions: `["read", "write"]`,
			Scope:       RoleScopeOrganization,
		}

		err := db.Create(role2).Error
		assert.NoError(t, err)
		assert.NotZero(t, role2.ID)
	})

	t.Run("name cannot be empty", func(t *testing.T) {
		role := &Role{
			Name:        "",
			Description: "Empty name role",
			Permissions: `["read"]`,
			Scope:       RoleScopeGlobal,
		}

		err := db.Create(role).Error
		assert.Error(t, err)
	})

	t.Run("scope must be valid", func(t *testing.T) {
		role := &Role{
			Name:        "Test Role",
			Description: "Test role with invalid scope",
			Permissions: `["read"]`,
			Scope:       RoleScope("invalid"),
		}

		err := db.Create(role).Error
		assert.Error(t, err)
	})
}

func TestRole_Relationships(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	t.Run("role can belong to multiple organizations", func(t *testing.T) {
		role := createTestRole(t, db, func(r *Role) {
			r.Name = "Org Role"
			r.Scope = RoleScopeOrganization
		})
		org1 := createTestOrganization(t, db, withOrgSlug("org1-role"))
		org2 := createTestOrganization(t, db, withOrgSlug("org2-role"))

		// Associate organizations with role
		err := db.Model(role).Association("Organizations").Append([]*Organization{org1, org2})
		require.NoError(t, err)

		// Load role with organizations
		var loadedRole Role
		err = db.Preload("Organizations").First(&loadedRole, role.ID).Error
		require.NoError(t, err)

		assert.Len(t, loadedRole.Organizations, 2)
		orgSlugs := []string{loadedRole.Organizations[0].Slug, loadedRole.Organizations[1].Slug}
		assert.Contains(t, orgSlugs, "org1-role")
		assert.Contains(t, orgSlugs, "org2-role")
	})
}

func TestRole_CRUD_Operations(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	t.Run("create role", func(t *testing.T) {
		role := &Role{
			Name:        "Test CRUD Role",
			Description: "A test role for CRUD operations",
			Permissions: `["read", "write"]`,
			Scope:       RoleScopeGlobal,
		}

		err := db.Create(role).Error
		require.NoError(t, err)
		assert.NotZero(t, role.ID)
		assert.NotZero(t, role.CreatedAt)
		assert.NotZero(t, role.UpdatedAt)
	})

	t.Run("update role", func(t *testing.T) {
		role := createTestRole(t, db, func(r *Role) {
			r.Name = "Update Test Role"
		})

		// Update the role
		role.Name = "Updated Role Name"
		role.Description = "Updated description"
		role.Permissions = `["read", "write", "delete"]`

		err := db.Save(role).Error
		require.NoError(t, err)

		// Verify the update
		var updatedRole Role
		err = db.First(&updatedRole, role.ID).Error
		require.NoError(t, err)

		assert.Equal(t, "Updated Role Name", updatedRole.Name)
		assert.Equal(t, "Updated description", updatedRole.Description)
		assert.Equal(t, `["read", "write", "delete"]`, updatedRole.Permissions)
		assert.True(t, updatedRole.UpdatedAt.After(updatedRole.CreatedAt))
	})
}
