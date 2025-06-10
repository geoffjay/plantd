package models

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupTestDB creates an in-memory SQLite database for testing.
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	// Run auto-migration
	err = AutoMigrate(db)
	require.NoError(t, err)

	return db
}

// cleanupTestDB cleans up the test database.
func cleanupTestDB(t *testing.T, db *gorm.DB) {
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())
}

// createTestUser creates a test user for testing purposes.
func createTestUser(t *testing.T, db *gorm.DB, overrides ...func(*User)) *User {
	// Generate unique values to avoid constraint violations
	uniqueID := rand.Intn(100000)
	user := &User{
		Email:          fmt.Sprintf("test_%d@example.com", uniqueID),
		Username:       fmt.Sprintf("testuser_%d", uniqueID),
		HashedPassword: "hashedpassword123",
		FirstName:      "Test",
		LastName:       "User",
		IsActive:       true,
		EmailVerified:  false,
	}

	for _, override := range overrides {
		override(user)
	}

	err := db.Create(user).Error
	require.NoError(t, err)

	return user
}

// createTestOrganization creates a test organization for testing purposes.
func createTestOrganization(t *testing.T, db *gorm.DB, overrides ...func(*Organization)) *Organization {
	// Generate unique values to avoid constraint violations
	uniqueID := rand.Intn(100000)
	org := &Organization{
		Name:        fmt.Sprintf("Test Organization %d", uniqueID),
		Slug:        fmt.Sprintf("test-org-%d", uniqueID),
		Description: "A test organization",
		IsActive:    true,
	}

	for _, override := range overrides {
		override(org)
	}

	err := db.Create(org).Error
	require.NoError(t, err)

	return org
}

// createTestRole creates a test role for testing purposes.
func createTestRole(t *testing.T, db *gorm.DB, overrides ...func(*Role)) *Role {
	// Generate unique values to avoid constraint violations
	uniqueID := rand.Intn(100000)
	role := &Role{
		Name:        fmt.Sprintf("Test Role %d", uniqueID),
		Description: "A test role",
		Permissions: `["read", "write"]`,
		Scope:       RoleScopeOrganization,
	}

	for _, override := range overrides {
		override(role)
	}

	err := db.Create(role).Error
	require.NoError(t, err)

	return role
}

// Helper functions for test data creation.
func withEmail(email string) func(*User) {
	return func(u *User) {
		u.Email = email
	}
}

func withUsername(username string) func(*User) {
	return func(u *User) {
		u.Username = username
	}
}

func withOrgSlug(slug string) func(*Organization) {
	return func(o *Organization) {
		o.Slug = slug
	}
}

func withRoleName(name string) func(*Role) {
	return func(r *Role) {
		r.Name = name
	}
}

func TestUser_TableName(t *testing.T) {
	user := User{}
	assert.Equal(t, "users", user.TableName())
}

func TestUser_BeforeCreate(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	tests := []struct {
		name          string
		email         string
		expectedEmail string
	}{
		{
			name:          "converts email to lowercase",
			email:         "TEST@EXAMPLE.COM",
			expectedEmail: "test@example.com",
		},
		{
			name:          "leaves lowercase email unchanged",
			email:         "test@example.com",
			expectedEmail: "test@example.com",
		},
		{
			name:          "handles mixed case",
			email:         "Test.User@Example.COM",
			expectedEmail: "test.user@example.com",
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use index to ensure unique usernames and emails
			// But preserve the original email pattern for testing transformation
			baseEmail := strings.Split(tt.email, "@")[0]
			domain := "@example.com"
			if strings.Contains(tt.email, "@") {
				parts := strings.Split(tt.email, "@")
				baseEmail = parts[0]
				domain = "@" + parts[1]
			}

			uniqueEmail := fmt.Sprintf("%s_%d%s", baseEmail, i, domain)
			expectedBaseEmail := strings.Split(tt.expectedEmail, "@")[0]
			expectedDomain := "@example.com"
			if strings.Contains(tt.expectedEmail, "@") {
				parts := strings.Split(tt.expectedEmail, "@")
				expectedBaseEmail = parts[0]
				expectedDomain = "@" + parts[1]
			}
			expectedEmail := fmt.Sprintf("%s_%d%s", expectedBaseEmail, i, expectedDomain)

			user := &User{
				Email:          uniqueEmail,
				Username:       fmt.Sprintf("testuser_%d_%s", i, strings.ReplaceAll(tt.name, " ", "_")),
				HashedPassword: "password123",
			}

			err := db.Create(user).Error
			require.NoError(t, err)

			assert.Equal(t, expectedEmail, user.Email)
		})
	}
}

func TestUser_BeforeUpdate(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	user := createTestUser(t, db, withEmail("original@example.com"))

	user.Email = "UPDATED@EXAMPLE.COM"
	err := db.Save(user).Error
	require.NoError(t, err)

	assert.Equal(t, "updated@example.com", user.Email)
}

func TestUser_GetFullName(t *testing.T) {
	tests := []struct {
		name         string
		firstName    string
		lastName     string
		username     string
		expectedName string
	}{
		{
			name:         "returns full name when both first and last name are set",
			firstName:    "John",
			lastName:     "Doe",
			username:     "johndoe",
			expectedName: "John Doe",
		},
		{
			name:         "returns username when no names are set",
			firstName:    "",
			lastName:     "",
			username:     "johndoe",
			expectedName: "johndoe",
		},
		{
			name:         "returns first name only when last name is empty",
			firstName:    "John",
			lastName:     "",
			username:     "johndoe",
			expectedName: "John",
		},
		{
			name:         "returns last name only when first name is empty",
			firstName:    "",
			lastName:     "Doe",
			username:     "johndoe",
			expectedName: "Doe",
		},
		{
			name:         "handles whitespace correctly",
			firstName:    "  John  ",
			lastName:     "  Doe  ",
			username:     "johndoe",
			expectedName: "John     Doe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &User{
				FirstName: tt.firstName,
				LastName:  tt.lastName,
				Username:  tt.username,
			}
			assert.Equal(t, tt.expectedName, user.GetFullName())
		})
	}
}

func TestUser_IsEmailVerified(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name             string
		emailVerified    bool
		emailVerifiedAt  *time.Time
		expectedVerified bool
	}{
		{
			name:             "returns true when email is verified and timestamp is set",
			emailVerified:    true,
			emailVerifiedAt:  &now,
			expectedVerified: true,
		},
		{
			name:             "returns false when email is not verified",
			emailVerified:    false,
			emailVerifiedAt:  &now,
			expectedVerified: false,
		},
		{
			name:             "returns false when email is verified but timestamp is nil",
			emailVerified:    true,
			emailVerifiedAt:  nil,
			expectedVerified: false,
		},
		{
			name:             "returns false when both are false/nil",
			emailVerified:    false,
			emailVerifiedAt:  nil,
			expectedVerified: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &User{
				EmailVerified:   tt.emailVerified,
				EmailVerifiedAt: tt.emailVerifiedAt,
			}
			assert.Equal(t, tt.expectedVerified, user.IsEmailVerified())
		})
	}
}

func TestUser_MarkEmailAsVerified(t *testing.T) {
	user := &User{
		EmailVerified:   false,
		EmailVerifiedAt: nil,
	}

	before := time.Now()
	user.MarkEmailAsVerified()
	after := time.Now()

	assert.True(t, user.EmailVerified)
	assert.NotNil(t, user.EmailVerifiedAt)
	assert.True(t, user.EmailVerifiedAt.After(before) || user.EmailVerifiedAt.Equal(before))
	assert.True(t, user.EmailVerifiedAt.Before(after) || user.EmailVerifiedAt.Equal(after))
}

func TestUser_UpdateLastLogin(t *testing.T) {
	user := &User{
		LastLoginAt: nil,
	}

	before := time.Now()
	user.UpdateLastLogin()
	after := time.Now()

	assert.NotNil(t, user.LastLoginAt)
	assert.True(t, user.LastLoginAt.After(before) || user.LastLoginAt.Equal(before))
	assert.True(t, user.LastLoginAt.Before(after) || user.LastLoginAt.Equal(after))
}

func TestUser_DatabaseConstraints(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	t.Run("email must be unique", func(t *testing.T) {
		// Create first user
		user1 := createTestUser(t, db, withEmail("unique@example.com"))
		assert.NotZero(t, user1.ID)

		// Try to create second user with same email
		user2 := &User{
			Email:          "unique@example.com",
			Username:       "differentuser",
			HashedPassword: "password123",
		}

		err := db.Create(user2).Error
		assert.Error(t, err)
	})

	t.Run("username must be unique", func(t *testing.T) {
		// Create first user
		user1 := createTestUser(t, db, withUsername("uniqueuser"))
		assert.NotZero(t, user1.ID)

		// Try to create second user with same username
		user2 := &User{
			Email:          "different@example.com",
			Username:       "uniqueuser",
			HashedPassword: "password123",
		}

		err := db.Create(user2).Error
		assert.Error(t, err)
	})

	t.Run("soft delete works correctly", func(t *testing.T) {
		user := createTestUser(t, db, withEmail("softdelete@example.com"))
		userID := user.ID

		// Delete the user (soft delete)
		err := db.Delete(user).Error
		require.NoError(t, err)

		// User should not be found in normal queries
		var foundUser User
		err = db.First(&foundUser, userID).Error
		assert.Error(t, err)
		assert.Equal(t, gorm.ErrRecordNotFound, err)

		// User should be found when including deleted records
		err = db.Unscoped().First(&foundUser, userID).Error
		require.NoError(t, err)
		assert.NotNil(t, foundUser.DeletedAt)
	})
}

func TestUser_Relationships(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	t.Run("user can have multiple roles", func(t *testing.T) {
		user := createTestUser(t, db)
		role1 := createTestRole(t, db, withRoleName("Role1"))
		role2 := createTestRole(t, db, withRoleName("Role2"))

		// Associate roles with user
		err := db.Model(user).Association("Roles").Append([]*Role{role1, role2})
		require.NoError(t, err)

		// Load user with roles
		var loadedUser User
		err = db.Preload("Roles").First(&loadedUser, user.ID).Error
		require.NoError(t, err)

		assert.Len(t, loadedUser.Roles, 2)
		roleNames := []string{loadedUser.Roles[0].Name, loadedUser.Roles[1].Name}
		assert.Contains(t, roleNames, "Role1")
		assert.Contains(t, roleNames, "Role2")
	})

	t.Run("user can belong to multiple organizations", func(t *testing.T) {
		user := createTestUser(t, db, withEmail("user@orgs.com"))
		org1 := createTestOrganization(t, db, withOrgSlug("org1"))
		org2 := createTestOrganization(t, db, withOrgSlug("org2"))

		// Associate organizations with user
		err := db.Model(user).Association("Organizations").Append([]*Organization{org1, org2})
		require.NoError(t, err)

		// Load user with organizations
		var loadedUser User
		err = db.Preload("Organizations").First(&loadedUser, user.ID).Error
		require.NoError(t, err)

		assert.Len(t, loadedUser.Organizations, 2)
		orgSlugs := []string{loadedUser.Organizations[0].Slug, loadedUser.Organizations[1].Slug}
		assert.Contains(t, orgSlugs, "org1")
		assert.Contains(t, orgSlugs, "org2")
	})
}
