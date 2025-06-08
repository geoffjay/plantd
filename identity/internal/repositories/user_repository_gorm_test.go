package repositories

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/geoffjay/plantd/identity/internal/models"
	"github.com/geoffjay/plantd/identity/internal/testhelpers"
)

func TestUserRepositoryGORM_Create(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer testhelpers.CleanupTestDB(t, db)

	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &models.User{
		Email:          "test@example.com",
		Username:       "testuser",
		HashedPassword: "hashedpassword",
		IsActive:       true,
	}

	err := repo.Create(ctx, user)
	require.NoError(t, err)
	assert.NotZero(t, user.ID)
	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, "testuser", user.Username)
}

func TestUserRepositoryGORM_GetByID(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer testhelpers.CleanupTestDB(t, db)

	repo := NewUserRepository(db)
	ctx := context.Background()

	// Create a user
	user := &models.User{
		Email:          "test@example.com",
		Username:       "testuser",
		HashedPassword: "hashedpassword",
		IsActive:       true,
	}
	require.NoError(t, repo.Create(ctx, user))

	// Get by ID
	retrieved, err := repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, user.ID, retrieved.ID)
	assert.Equal(t, user.Email, retrieved.Email)
}

func TestUserRepositoryGORM_GetByEmail(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer testhelpers.CleanupTestDB(t, db)

	repo := NewUserRepository(db)
	ctx := context.Background()

	// Create a user
	user := &models.User{
		Email:          "test@example.com",
		Username:       "testuser",
		HashedPassword: "hashedpassword",
		IsActive:       true,
	}
	require.NoError(t, repo.Create(ctx, user))

	// Get by email
	retrieved, err := repo.GetByEmail(ctx, "test@example.com")
	require.NoError(t, err)
	assert.Equal(t, user.ID, retrieved.ID)
	assert.Equal(t, user.Email, retrieved.Email)
}

func TestUserRepositoryGORM_Update(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer testhelpers.CleanupTestDB(t, db)

	repo := NewUserRepository(db)
	ctx := context.Background()

	// Create a user
	user := &models.User{
		Email:          "update@example.com",
		Username:       "updateuser",
		HashedPassword: "hashedpassword",
		IsActive:       true,
	}
	require.NoError(t, repo.Create(ctx, user))

	// Update the user
	user.Username = "updateduserupdated"
	user.Email = "updatedupdated@example.com"
	err := repo.Update(ctx, user)
	require.NoError(t, err)

	// Verify the update
	retrieved, err := repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, "updateduserupdated", retrieved.Username)
	assert.Equal(t, "updatedupdated@example.com", retrieved.Email)
}

func TestUserRepositoryGORM_Delete(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer testhelpers.CleanupTestDB(t, db)

	repo := NewUserRepository(db)
	ctx := context.Background()

	// Create a user
	user := &models.User{
		Email:          "test@example.com",
		Username:       "testuser",
		HashedPassword: "hashedpassword",
		IsActive:       true,
	}
	require.NoError(t, repo.Create(ctx, user))

	// Delete the user
	err := repo.Delete(ctx, user.ID)
	require.NoError(t, err)

	// Verify deletion (should return error)
	_, err = repo.GetByID(ctx, user.ID)
	assert.Error(t, err)
}

func TestUserRepositoryGORM_List(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer testhelpers.CleanupTestDB(t, db)

	repo := NewUserRepository(db)
	ctx := context.Background()

	// Create multiple users
	users := []*models.User{
		{Email: "user1@example.com", Username: "user1", HashedPassword: "password1", IsActive: true},
		{Email: "user2@example.com", Username: "user2", HashedPassword: "password2", IsActive: true},
		{Email: "user3@example.com", Username: "user3", HashedPassword: "password3", IsActive: true},
	}

	for _, user := range users {
		require.NoError(t, repo.Create(ctx, user))
	}

	// List users
	retrieved, err := repo.List(ctx, 0, 10)
	require.NoError(t, err)
	assert.Len(t, retrieved, 3)
}

func TestUserRepositoryGORM_Count(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer testhelpers.CleanupTestDB(t, db)

	repo := NewUserRepository(db)
	ctx := context.Background()

	// Initially should be 0
	count, err := repo.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)

	// Create users
	users := []*models.User{
		{Email: "user1@example.com", Username: "user1", HashedPassword: "password1", IsActive: true},
		{Email: "user2@example.com", Username: "user2", HashedPassword: "password2", IsActive: true},
	}

	for _, user := range users {
		require.NoError(t, repo.Create(ctx, user))
	}

	// Count should be 2
	count, err = repo.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

func TestUserRepositoryGORM_SearchByEmail(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer testhelpers.CleanupTestDB(t, db)

	repo := NewUserRepository(db)
	ctx := context.Background()

	// Create users
	users := []*models.User{
		{Email: "john.doe@example.com", Username: "john", HashedPassword: "password1", IsActive: true},
		{Email: "jane.smith@example.com", Username: "jane", HashedPassword: "password2", IsActive: true},
		{Email: "bob.johnson@test.com", Username: "bob", HashedPassword: "password3", IsActive: true},
	}

	for _, user := range users {
		require.NoError(t, repo.Create(ctx, user))
	}

	// Search by email pattern
	results, err := repo.SearchByEmail(ctx, "@example.com", 0, 10)
	require.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestUserRepositoryGORM_SearchByName(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer testhelpers.CleanupTestDB(t, db)

	repo := NewUserRepository(db)
	ctx := context.Background()

	// Create users
	users := []*models.User{
		{Email: "user1@example.com", Username: "user1", FirstName: "John", LastName: "Doe", HashedPassword: "password1", IsActive: true},
		{Email: "user2@example.com", Username: "user2", FirstName: "Jane", LastName: "Smith", HashedPassword: "password2", IsActive: true},
		{Email: "user3@example.com", Username: "user3", FirstName: "Bob", LastName: "Johnson", HashedPassword: "password3", IsActive: true},
	}

	for _, user := range users {
		require.NoError(t, repo.Create(ctx, user))
	}

	// Search by name pattern
	results, err := repo.SearchByName(ctx, "John", 0, 10)
	require.NoError(t, err)
	assert.Len(t, results, 2) // Should find "John Doe" and "Bob Johnson"
}

func TestUserRepositoryGORM_GetActiveUsers(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer testhelpers.CleanupTestDB(t, db)

	repo := NewUserRepository(db)
	ctx := context.Background()

	// Create users with different active states
	users := []*models.User{
		{Email: "active1@example.com", Username: "active1", HashedPassword: "password1", IsActive: true},
		{Email: "active2@example.com", Username: "active2", HashedPassword: "password2", IsActive: true},
		{Email: "inactive@example.com", Username: "inactive", HashedPassword: "password3", IsActive: false},
	}

	for _, user := range users {
		require.NoError(t, repo.Create(ctx, user))
	}

	// Get active users
	results, err := repo.GetActiveUsers(ctx, 0, 10)
	require.NoError(t, err)
	assert.Len(t, results, 2)
	for _, user := range results {
		assert.True(t, user.IsActive)
	}
}

func TestUserRepositoryGORM_GetVerifiedUsers(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer testhelpers.CleanupTestDB(t, db)

	repo := NewUserRepository(db)
	ctx := context.Background()

	// Create users with different verification states
	users := []*models.User{
		{Email: "verified1@example.com", Username: "verified1", HashedPassword: "password1", IsActive: true, EmailVerified: true},
		{Email: "verified2@example.com", Username: "verified2", HashedPassword: "password2", IsActive: true, EmailVerified: true},
		{Email: "unverified@example.com", Username: "unverified", HashedPassword: "password3", IsActive: true, EmailVerified: false},
	}

	for _, user := range users {
		require.NoError(t, repo.Create(ctx, user))
	}

	// Get verified users
	results, err := repo.GetVerifiedUsers(ctx, 0, 10)
	require.NoError(t, err)
	assert.Len(t, results, 2)
	for _, user := range results {
		assert.True(t, user.EmailVerified)
	}
}

func TestUserRepositoryGORM_Pagination(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer testhelpers.CleanupTestDB(t, db)

	repo := NewUserRepository(db)
	ctx := context.Background()

	// Create multiple users
	for i := 0; i < 15; i++ {
		user := &models.User{
			Email:          fmt.Sprintf("user%d@example.com", i),
			Username:       fmt.Sprintf("user%d", i),
			HashedPassword: "password",
			IsActive:       true,
		}
		require.NoError(t, repo.Create(ctx, user))
	}

	// Test pagination
	firstPage, err := repo.List(ctx, 0, 5)
	require.NoError(t, err)
	assert.Len(t, firstPage, 5)

	secondPage, err := repo.List(ctx, 5, 5)
	require.NoError(t, err)
	assert.Len(t, secondPage, 5)

	// Ensure pages are different
	assert.NotEqual(t, firstPage[0].ID, secondPage[0].ID)
}

func TestUserRepositoryGORM_ErrorHandling(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer testhelpers.CleanupTestDB(t, db)

	repo := NewUserRepository(db)
	ctx := context.Background()

	t.Run("create duplicate email", func(t *testing.T) {
		user1 := &models.User{
			Email:          "duplicate@example.com",
			Username:       "user1",
			HashedPassword: "password123",
		}

		err := repo.Create(ctx, user1)
		require.NoError(t, err)

		user2 := &models.User{
			Email:          "duplicate@example.com",
			Username:       "user2",
			HashedPassword: "password123",
		}

		err = repo.Create(ctx, user2)
		assert.Error(t, err)
	})

	t.Run("create duplicate username", func(t *testing.T) {
		user1 := &models.User{
			Email:          "user1@example.com",
			Username:       "duplicateuser",
			HashedPassword: "password123",
		}

		err := repo.Create(ctx, user1)
		require.NoError(t, err)

		user2 := &models.User{
			Email:          "user2@example.com",
			Username:       "duplicateuser",
			HashedPassword: "password123",
		}

		err = repo.Create(ctx, user2)
		assert.Error(t, err)
	})

	t.Run("update non-existing user", func(t *testing.T) {
		user := &models.User{
			ID:       99999,
			Email:    "nonexistent@example.com",
			Username: "nonexistent",
		}

		err := repo.Update(ctx, user)
		assert.Error(t, err)
	})

	t.Run("delete non-existing user", func(t *testing.T) {
		err := repo.Delete(ctx, 99999)
		assert.Error(t, err)
	})
}
