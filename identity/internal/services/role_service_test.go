package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/geoffjay/plantd/identity/internal/models"
)

// MockRoleRepositoryForTest for role service testing.
type MockRoleRepositoryForTest struct {
	mock.Mock
}

func (m *MockRoleRepositoryForTest) Create(ctx context.Context, role *models.Role) error {
	args := m.Called(ctx, role)
	return args.Error(0)
}

func (m *MockRoleRepositoryForTest) GetByID(ctx context.Context, id uint) (*models.Role, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Role), args.Error(1)
}

func (m *MockRoleRepositoryForTest) GetByName(ctx context.Context, name string) (*models.Role, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Role), args.Error(1)
}

func (m *MockRoleRepositoryForTest) Update(ctx context.Context, role *models.Role) error {
	args := m.Called(ctx, role)
	return args.Error(0)
}

func (m *MockRoleRepositoryForTest) Delete(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRoleRepositoryForTest) List(ctx context.Context, offset, limit int) ([]*models.Role, error) {
	args := m.Called(ctx, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Role), args.Error(1)
}

func (m *MockRoleRepositoryForTest) Count(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockRoleRepositoryForTest) SearchByName(ctx context.Context, namePattern string, offset, limit int) ([]*models.Role, error) {
	args := m.Called(ctx, namePattern, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Role), args.Error(1)
}

func (m *MockRoleRepositoryForTest) GetByScope(ctx context.Context, scope models.RoleScope, offset, limit int) ([]*models.Role, error) {
	args := m.Called(ctx, scope, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Role), args.Error(1)
}

func (m *MockRoleRepositoryForTest) GetByUser(ctx context.Context, userID uint, offset, limit int) ([]*models.Role, error) {
	args := m.Called(ctx, userID, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Role), args.Error(1)
}

func (m *MockRoleRepositoryForTest) GetByOrganization(ctx context.Context, organizationID uint, offset, limit int) ([]*models.Role, error) {
	args := m.Called(ctx, organizationID, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Role), args.Error(1)
}

func (m *MockRoleRepositoryForTest) GetWithUsers(ctx context.Context, id uint) (*models.Role, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Role), args.Error(1)
}

func (m *MockRoleRepositoryForTest) GetWithOrganizations(ctx context.Context, id uint) (*models.Role, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Role), args.Error(1)
}

func (m *MockRoleRepositoryForTest) GetWithAll(ctx context.Context, id uint) (*models.Role, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Role), args.Error(1)
}

func (m *MockRoleRepositoryForTest) GetGlobalRoles(ctx context.Context, offset, limit int) ([]*models.Role, error) {
	args := m.Called(ctx, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Role), args.Error(1)
}

func (m *MockRoleRepositoryForTest) GetOrganizationRoles(ctx context.Context, offset, limit int) ([]*models.Role, error) {
	args := m.Called(ctx, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Role), args.Error(1)
}

func (m *MockRoleRepositoryForTest) GetByUserAndOrganization(
	ctx context.Context,
	userID,
	organizationID uint,
	offset,
	limit int,
) ([]*models.Role, error) {
	args := m.Called(ctx, userID, organizationID, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Role), args.Error(1)
}

func (m *MockRoleRepositoryForTest) SearchByDescription(ctx context.Context, descriptionPattern string, offset, limit int) ([]*models.Role, error) {
	args := m.Called(ctx, descriptionPattern, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Role), args.Error(1)
}

func (m *MockRoleRepositoryForTest) GetRolesWithPermission(ctx context.Context, permission string, offset, limit int) ([]*models.Role, error) {
	args := m.Called(ctx, permission, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Role), args.Error(1)
}

func (m *MockRoleRepositoryForTest) AssignToUser(ctx context.Context, roleID, userID uint) error {
	args := m.Called(ctx, roleID, userID)
	return args.Error(0)
}

func (m *MockRoleRepositoryForTest) UnassignFromUser(ctx context.Context, roleID, userID uint) error {
	args := m.Called(ctx, roleID, userID)
	return args.Error(0)
}

func (m *MockRoleRepositoryForTest) AssignToOrganization(ctx context.Context, roleID, organizationID uint) error {
	args := m.Called(ctx, roleID, organizationID)
	return args.Error(0)
}

func (m *MockRoleRepositoryForTest) UnassignFromOrganization(ctx context.Context, roleID, organizationID uint) error {
	args := m.Called(ctx, roleID, organizationID)
	return args.Error(0)
}

// Test setup helper.
func setupRoleService() (*roleServiceImpl, *MockRoleRepositoryForTest) {
	mockRoleRepo := &MockRoleRepositoryForTest{}
	mockUserRepo := &MockUserRepository{}
	mockOrgRepo := &MockOrganizationRepository{}
	service := NewRoleService(mockRoleRepo, mockUserRepo, mockOrgRepo).(*roleServiceImpl)
	return service, mockRoleRepo
}

// CreateRole Tests.
func TestRoleService_CreateRole_Success(t *testing.T) {
	service, mockRoleRepo := setupRoleService()
	ctx := context.Background()

	permissions := []string{"read", "write"}
	req := &CreateRoleRequest{
		Name:        "Test Role",
		Description: "Test Description",
		Permissions: permissions,
		Scope:       models.RoleScopeGlobal,
	}

	// Mock repository calls
	mockRoleRepo.On("SearchByName", ctx, "Test Role", 0, 10).Return([]*models.Role{}, nil)
	mockRoleRepo.On("Create", ctx, mock.AnythingOfType("*models.Role")).Return(nil)

	// Execute
	role, err := service.CreateRole(ctx, req)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, role)
	assert.Equal(t, "Test Role", role.Name)
	assert.Equal(t, "Test Description", role.Description)
	assert.Equal(t, models.RoleScopeGlobal, role.Scope)
	assert.Contains(t, role.Permissions, "read")
	assert.Contains(t, role.Permissions, "write")

	mockRoleRepo.AssertExpectations(t)
}

func TestRoleService_CreateRole_NameExists(t *testing.T) {
	service, mockRoleRepo := setupRoleService()
	ctx := context.Background()

	req := &CreateRoleRequest{
		Name:        "Existing Role",
		Permissions: []string{"read"},
		Scope:       models.RoleScopeGlobal,
	}

	existingRole := &models.Role{ID: 1, Name: "Existing Role", Scope: models.RoleScopeGlobal}

	// Mock repository calls - use limit 10 to match service implementation
	// The service validates uniqueness first, so mock that call
	mockRoleRepo.On("SearchByName", ctx, "Existing Role", 0, 10).Return([]*models.Role{existingRole}, nil)

	// Execute
	role, err := service.CreateRole(ctx, req)

	// Assert
	require.Error(t, err)
	assert.Nil(t, role)
	assert.Contains(t, err.Error(), "role name already exists")

	mockRoleRepo.AssertExpectations(t)
}

func TestRoleService_GetRoleByID_Success(t *testing.T) {
	service, mockRoleRepo := setupRoleService()
	ctx := context.Background()

	expectedRole := &models.Role{
		ID:          1,
		Name:        "Test Role",
		Description: "Test Description",
		Scope:       models.RoleScopeGlobal,
	}

	// Mock repository calls
	mockRoleRepo.On("GetByID", ctx, uint(1)).Return(expectedRole, nil)

	// Execute
	role, err := service.GetRoleByID(ctx, 1)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, expectedRole, role)

	mockRoleRepo.AssertExpectations(t)
}

func TestRoleService_GetRoleByID_NotFound(t *testing.T) {
	service, mockRoleRepo := setupRoleService()
	ctx := context.Background()

	// Mock repository calls
	mockRoleRepo.On("GetByID", ctx, uint(999)).Return(nil, nil)

	// Execute
	role, err := service.GetRoleByID(ctx, 999)

	// Assert
	require.Error(t, err)
	assert.Nil(t, role)
	assert.Contains(t, err.Error(), "role not found")

	mockRoleRepo.AssertExpectations(t)
}

func TestRoleService_DeleteRole_Success(t *testing.T) {
	service, mockRoleRepo := setupRoleService()
	ctx := context.Background()

	existingRole := &models.Role{ID: 1, Name: "Test Role"}

	// Mock repository calls
	mockRoleRepo.On("GetByID", ctx, uint(1)).Return(existingRole, nil)
	mockRoleRepo.On("Delete", ctx, uint(1)).Return(nil)

	// Execute
	err := service.DeleteRole(ctx, 1)

	// Assert
	require.NoError(t, err)

	mockRoleRepo.AssertExpectations(t)
}

func TestRoleService_ListRoles_Success(t *testing.T) {
	service, mockRoleRepo := setupRoleService()
	ctx := context.Background()

	expectedRoles := []*models.Role{
		{ID: 1, Name: "Role 1"},
		{ID: 2, Name: "Role 2"},
	}

	req := &ListRolesRequest{
		Offset: 0,
		Limit:  10,
	}

	// Mock repository calls
	mockRoleRepo.On("List", ctx, 0, 10).Return(expectedRoles, nil)

	// Execute
	roles, err := service.ListRoles(ctx, req)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, expectedRoles, roles)
	assert.Len(t, roles, 2)

	mockRoleRepo.AssertExpectations(t)
}

func TestRoleService_CountRoles_Success(t *testing.T) {
	service, mockRoleRepo := setupRoleService()
	ctx := context.Background()

	// Mock repository calls
	mockRoleRepo.On("Count", ctx).Return(int64(15), nil)

	// Execute
	count, err := service.CountRoles(ctx)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, int64(15), count)

	mockRoleRepo.AssertExpectations(t)
}
