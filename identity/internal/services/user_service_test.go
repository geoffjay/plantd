package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/geoffjay/plantd/identity/internal/models"
)

// MockUserRepository is a mock implementation of UserRepository.
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) GetByID(ctx context.Context, id uint) (*models.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) Update(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) Delete(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserRepository) List(ctx context.Context, offset, limit int) ([]*models.User, error) {
	args := m.Called(ctx, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.User), args.Error(1)
}

func (m *MockUserRepository) Count(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockUserRepository) GetActiveUsers(ctx context.Context, offset, limit int) ([]*models.User, error) {
	args := m.Called(ctx, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.User), args.Error(1)
}

func (m *MockUserRepository) GetVerifiedUsers(ctx context.Context, offset, limit int) ([]*models.User, error) {
	args := m.Called(ctx, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.User), args.Error(1)
}

func (m *MockUserRepository) SearchByEmail(ctx context.Context, emailPattern string, offset, limit int) ([]*models.User, error) {
	args := m.Called(ctx, emailPattern, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.User), args.Error(1)
}

func (m *MockUserRepository) SearchByUsername(ctx context.Context, usernamePattern string, offset, limit int) ([]*models.User, error) {
	args := m.Called(ctx, usernamePattern, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.User), args.Error(1)
}

func (m *MockUserRepository) SearchByName(ctx context.Context, namePattern string, offset, limit int) ([]*models.User, error) {
	args := m.Called(ctx, namePattern, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.User), args.Error(1)
}

func (m *MockUserRepository) GetByOrganization(ctx context.Context, organizationID uint, offset, limit int) ([]*models.User, error) {
	args := m.Called(ctx, organizationID, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.User), args.Error(1)
}

func (m *MockUserRepository) GetByRole(ctx context.Context, roleID uint, offset, limit int) ([]*models.User, error) {
	args := m.Called(ctx, roleID, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.User), args.Error(1)
}

func (m *MockUserRepository) GetWithRoles(ctx context.Context, id uint) (*models.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetWithOrganizations(ctx context.Context, id uint) (*models.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetWithAll(ctx context.Context, id uint) (*models.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) AssignRole(ctx context.Context, userID, roleID uint) error {
	args := m.Called(ctx, userID, roleID)
	return args.Error(0)
}

func (m *MockUserRepository) UnassignRole(ctx context.Context, userID, roleID uint) error {
	args := m.Called(ctx, userID, roleID)
	return args.Error(0)
}

func (m *MockUserRepository) AddToOrganization(ctx context.Context, userID, organizationID uint) error {
	args := m.Called(ctx, userID, organizationID)
	return args.Error(0)
}

func (m *MockUserRepository) RemoveFromOrganization(ctx context.Context, userID, organizationID uint) error {
	args := m.Called(ctx, userID, organizationID)
	return args.Error(0)
}

// Test setup helper.
func setupUserService() (*userServiceImpl, *MockUserRepository) {
	mockUserRepo := &MockUserRepository{}

	service := NewUserService(mockUserRepo, nil, nil).(*userServiceImpl)

	return service, mockUserRepo
}

// CreateUser Tests.
func TestUserService_CreateUser_Success(t *testing.T) {
	service, mockUserRepo := setupUserService()
	ctx := context.Background()

	req := &CreateUserRequest{
		Email:     "test@example.com",
		Username:  "testuser",
		Password:  "password123",
		FirstName: "Test",
		LastName:  "User",
	}

	// Mock repository calls
	mockUserRepo.On("GetByEmail", ctx, "test@example.com").Return(nil, nil)
	mockUserRepo.On("GetByUsername", ctx, "testuser").Return(nil, nil)
	mockUserRepo.On("Create", ctx, mock.AnythingOfType("*models.User")).Return(nil)

	// Execute
	user, err := service.CreateUser(ctx, req)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, "testuser", user.Username)
	assert.Equal(t, "Test", user.FirstName)
	assert.Equal(t, "User", user.LastName)
	assert.True(t, user.IsActive)
	assert.False(t, user.EmailVerified)

	mockUserRepo.AssertExpectations(t)
}

func TestUserService_CreateUser_EmailExists(t *testing.T) {
	service, mockUserRepo := setupUserService()
	ctx := context.Background()

	req := &CreateUserRequest{
		Email:    "existing@example.com",
		Username: "testuser",
		Password: "password123",
	}

	existingUser := &models.User{ID: 1, Email: "existing@example.com"}

	// Mock repository calls
	mockUserRepo.On("GetByEmail", ctx, "existing@example.com").Return(existingUser, nil)

	// Execute
	user, err := service.CreateUser(ctx, req)

	// Assert
	require.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "email already exists")

	mockUserRepo.AssertExpectations(t)
}

func TestUserService_GetUserByID_Success(t *testing.T) {
	service, mockUserRepo := setupUserService()
	ctx := context.Background()

	expectedUser := &models.User{
		ID:       1,
		Email:    "test@example.com",
		Username: "testuser",
	}

	// Mock repository calls
	mockUserRepo.On("GetByID", ctx, uint(1)).Return(expectedUser, nil)

	// Execute
	user, err := service.GetUserByID(ctx, 1)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, expectedUser, user)

	mockUserRepo.AssertExpectations(t)
}

func TestUserService_GetUserByID_NotFound(t *testing.T) {
	service, mockUserRepo := setupUserService()
	ctx := context.Background()

	// Mock repository calls
	mockUserRepo.On("GetByID", ctx, uint(999)).Return(nil, nil)

	// Execute
	user, err := service.GetUserByID(ctx, 999)

	// Assert
	require.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "user not found")

	mockUserRepo.AssertExpectations(t)
}

func TestUserService_CreateUser_UsernameExists(t *testing.T) {
	service, mockUserRepo := setupUserService()
	ctx := context.Background()

	req := &CreateUserRequest{
		Email:    "test@example.com",
		Username: "existinguser",
		Password: "password123",
	}

	existingUser := &models.User{ID: 1, Username: "existinguser"}

	// Mock repository calls
	mockUserRepo.On("GetByEmail", ctx, "test@example.com").Return(nil, nil)
	mockUserRepo.On("GetByUsername", ctx, "existinguser").Return(existingUser, nil)

	// Execute
	user, err := service.CreateUser(ctx, req)

	// Assert
	require.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "username already exists")

	mockUserRepo.AssertExpectations(t)
}

func TestUserService_CreateUser_InvalidEmail(t *testing.T) {
	service, _ := setupUserService()
	ctx := context.Background()

	req := &CreateUserRequest{
		Email:    "invalid-email",
		Username: "testuser",
		Password: "password123",
	}

	// Execute
	user, err := service.CreateUser(ctx, req)

	// Assert
	require.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "validation failed")
}

func TestUserService_GetUserByEmail_Success(t *testing.T) {
	service, mockUserRepo := setupUserService()
	ctx := context.Background()

	expectedUser := &models.User{
		ID:       1,
		Email:    "test@example.com",
		Username: "testuser",
	}

	// Mock repository calls
	mockUserRepo.On("GetByEmail", ctx, "test@example.com").Return(expectedUser, nil)

	// Execute
	user, err := service.GetUserByEmail(ctx, "test@example.com")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, expectedUser, user)

	mockUserRepo.AssertExpectations(t)
}

func TestUserService_GetUserByEmail_EmptyEmail(t *testing.T) {
	service, _ := setupUserService()
	ctx := context.Background()

	// Execute
	user, err := service.GetUserByEmail(ctx, "")

	// Assert
	require.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "email cannot be empty")
}

func TestUserService_UpdateUser_Success(t *testing.T) {
	service, mockUserRepo := setupUserService()
	ctx := context.Background()

	existingUser := &models.User{
		ID:        1,
		Email:     "old@example.com",
		Username:  "olduser",
		FirstName: "Old",
		LastName:  "User",
		IsActive:  true,
	}

	newEmail := "new@example.com"
	newFirstName := "New"
	req := &UpdateUserRequest{
		Email:     &newEmail,
		FirstName: &newFirstName,
	}

	// Mock repository calls
	mockUserRepo.On("GetByID", ctx, uint(1)).Return(existingUser, nil)
	mockUserRepo.On("GetByEmail", ctx, "new@example.com").Return(nil, nil)
	mockUserRepo.On("Update", ctx, mock.AnythingOfType("*models.User")).Return(nil)

	// Execute
	user, err := service.UpdateUser(ctx, 1, req)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "new@example.com", user.Email)
	assert.Equal(t, "New", user.FirstName)
	assert.Equal(t, "olduser", user.Username)
	assert.Equal(t, "User", user.LastName)

	mockUserRepo.AssertExpectations(t)
}

func TestUserService_DeleteUser_Success(t *testing.T) {
	service, mockUserRepo := setupUserService()
	ctx := context.Background()

	existingUser := &models.User{ID: 1, Email: "test@example.com"}

	// Mock repository calls
	mockUserRepo.On("GetByID", ctx, uint(1)).Return(existingUser, nil)
	mockUserRepo.On("Delete", ctx, uint(1)).Return(nil)

	// Execute
	err := service.DeleteUser(ctx, 1)

	// Assert
	require.NoError(t, err)

	mockUserRepo.AssertExpectations(t)
}

func TestUserService_ListUsers_Success(t *testing.T) {
	service, mockUserRepo := setupUserService()
	ctx := context.Background()

	expectedUsers := []*models.User{
		{ID: 1, Email: "user1@example.com"},
		{ID: 2, Email: "user2@example.com"},
	}

	req := &ListUsersRequest{
		Offset: 0,
		Limit:  10,
	}

	// Mock repository calls
	mockUserRepo.On("List", ctx, 0, 10).Return(expectedUsers, nil)

	// Execute
	users, err := service.ListUsers(ctx, req)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, expectedUsers, users)
	assert.Len(t, users, 2)

	mockUserRepo.AssertExpectations(t)
}

func TestUserService_CountUsers_Success(t *testing.T) {
	service, mockUserRepo := setupUserService()
	ctx := context.Background()

	// Mock repository calls
	mockUserRepo.On("Count", ctx).Return(int64(42), nil)

	// Execute
	count, err := service.CountUsers(ctx)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, int64(42), count)

	mockUserRepo.AssertExpectations(t)
}

func TestUserService_ActivateUser_Success(t *testing.T) {
	service, mockUserRepo := setupUserService()
	ctx := context.Background()

	user := &models.User{
		ID:       1,
		Email:    "test@example.com",
		IsActive: false,
	}

	// Mock repository calls
	mockUserRepo.On("GetByID", ctx, uint(1)).Return(user, nil)
	mockUserRepo.On("Update", ctx, user).Return(nil)

	// Execute
	err := service.ActivateUser(ctx, 1)

	// Assert
	require.NoError(t, err)
	assert.True(t, user.IsActive)

	mockUserRepo.AssertExpectations(t)
}

func TestUserService_DeactivateUser_Success(t *testing.T) {
	service, mockUserRepo := setupUserService()
	ctx := context.Background()

	user := &models.User{
		ID:       1,
		Email:    "test@example.com",
		IsActive: true,
	}

	// Mock repository calls
	mockUserRepo.On("GetByID", ctx, uint(1)).Return(user, nil)
	mockUserRepo.On("Update", ctx, user).Return(nil)

	// Execute
	err := service.DeactivateUser(ctx, 1)

	// Assert
	require.NoError(t, err)
	assert.False(t, user.IsActive)

	mockUserRepo.AssertExpectations(t)
}

func TestUserService_VerifyUserEmail_Success(t *testing.T) {
	service, mockUserRepo := setupUserService()
	ctx := context.Background()

	user := &models.User{
		ID:            1,
		Email:         "test@example.com",
		EmailVerified: false,
	}

	// Mock repository calls
	mockUserRepo.On("GetByID", ctx, uint(1)).Return(user, nil)
	mockUserRepo.On("Update", ctx, user).Return(nil)

	// Execute
	err := service.VerifyUserEmail(ctx, 1)

	// Assert
	require.NoError(t, err)
	assert.True(t, user.EmailVerified)

	mockUserRepo.AssertExpectations(t)
}

func TestUserService_SearchUsers_Success(t *testing.T) {
	service, mockUserRepo := setupUserService()
	ctx := context.Background()

	expectedUsers := []*models.User{
		{ID: 1, Email: "john@example.com", Username: "john"},
	}

	req := &SearchUsersRequest{
		Query:        "john",
		SearchFields: []string{"email"},
		Offset:       0,
		Limit:        10,
	}

	// Mock repository calls
	mockUserRepo.On("SearchByEmail", ctx, "john", 0, 10).Return(expectedUsers, nil)

	// Execute
	users, err := service.SearchUsers(ctx, req)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, expectedUsers, users)

	mockUserRepo.AssertExpectations(t)
}

// Helper function.
func stringPtr(s string) *string {
	return &s
}
