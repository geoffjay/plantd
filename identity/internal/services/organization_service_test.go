package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/geoffjay/plantd/identity/internal/models"
)

// MockOrganizationRepository for testing.
type MockOrganizationRepository struct {
	mock.Mock
}

func (m *MockOrganizationRepository) Create(ctx context.Context, organization *models.Organization) error {
	args := m.Called(ctx, organization)
	return args.Error(0)
}

func (m *MockOrganizationRepository) GetByID(ctx context.Context, id uint) (*models.Organization, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Organization), args.Error(1)
}

func (m *MockOrganizationRepository) GetBySlug(ctx context.Context, slug string) (*models.Organization, error) {
	args := m.Called(ctx, slug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Organization), args.Error(1)
}

func (m *MockOrganizationRepository) Update(ctx context.Context, organization *models.Organization) error {
	args := m.Called(ctx, organization)
	return args.Error(0)
}

func (m *MockOrganizationRepository) Delete(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockOrganizationRepository) List(ctx context.Context, offset, limit int) ([]*models.Organization, error) {
	args := m.Called(ctx, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Organization), args.Error(1)
}

func (m *MockOrganizationRepository) Count(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockOrganizationRepository) GetByUser(ctx context.Context, userID uint, offset, limit int) ([]*models.Organization, error) {
	args := m.Called(ctx, userID, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Organization), args.Error(1)
}

func (m *MockOrganizationRepository) GetActiveOrganizations(ctx context.Context, offset, limit int) ([]*models.Organization, error) {
	args := m.Called(ctx, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Organization), args.Error(1)
}

func (m *MockOrganizationRepository) SearchByName(ctx context.Context, namePattern string, offset, limit int) ([]*models.Organization, error) {
	args := m.Called(ctx, namePattern, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Organization), args.Error(1)
}

func (m *MockOrganizationRepository) SearchBySlug(ctx context.Context, slugPattern string, offset, limit int) ([]*models.Organization, error) {
	args := m.Called(ctx, slugPattern, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Organization), args.Error(1)
}

func (m *MockOrganizationRepository) AddUser(ctx context.Context, organizationID, userID uint) error {
	args := m.Called(ctx, organizationID, userID)
	return args.Error(0)
}

func (m *MockOrganizationRepository) RemoveUser(ctx context.Context, organizationID, userID uint) error {
	args := m.Called(ctx, organizationID, userID)
	return args.Error(0)
}

func (m *MockOrganizationRepository) AddRole(ctx context.Context, organizationID, roleID uint) error {
	args := m.Called(ctx, organizationID, roleID)
	return args.Error(0)
}

func (m *MockOrganizationRepository) RemoveRole(ctx context.Context, organizationID, roleID uint) error {
	args := m.Called(ctx, organizationID, roleID)
	return args.Error(0)
}

func (m *MockOrganizationRepository) GetMembers(ctx context.Context, organizationID uint, offset, limit int) ([]*models.User, error) {
	args := m.Called(ctx, organizationID, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.User), args.Error(1)
}

func (m *MockOrganizationRepository) GetMembersWithRoles(ctx context.Context, organizationID uint, offset, limit int) ([]*models.User, error) {
	args := m.Called(ctx, organizationID, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.User), args.Error(1)
}

func (m *MockOrganizationRepository) CountMembers(ctx context.Context, organizationID uint) (int64, error) {
	args := m.Called(ctx, organizationID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockOrganizationRepository) GetWithUsers(ctx context.Context, id uint) (*models.Organization, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Organization), args.Error(1)
}

func (m *MockOrganizationRepository) GetWithRoles(ctx context.Context, id uint) (*models.Organization, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Organization), args.Error(1)
}

func (m *MockOrganizationRepository) GetWithAll(ctx context.Context, id uint) (*models.Organization, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Organization), args.Error(1)
}

// Test setup helper.
func setupOrganizationService() (*organizationServiceImpl, *MockOrganizationRepository) {
	mockOrgRepo := &MockOrganizationRepository{}
	mockUserRepo := &MockUserRepository{}
	mockRoleRepo := &MockRoleRepositoryForTest{}
	service := NewOrganizationService(mockOrgRepo, mockUserRepo, mockRoleRepo).(*organizationServiceImpl)
	return service, mockOrgRepo
}

// CreateOrganization Tests.
func TestOrganizationService_CreateOrganization_Success(t *testing.T) {
	service, mockOrgRepo := setupOrganizationService()
	ctx := context.Background()

	req := &CreateOrganizationRequest{
		Name:        "Test Organization",
		Description: "Test Description",
	}

	// Mock repository calls
	mockOrgRepo.On("SearchByName", ctx, "Test Organization", 0, 1).Return([]*models.Organization{}, nil)
	mockOrgRepo.On("GetBySlug", ctx, "test-organization").Return(nil, nil)
	mockOrgRepo.On("Create", ctx, mock.AnythingOfType("*models.Organization")).Return(nil)

	// Execute
	org, err := service.CreateOrganization(ctx, req)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, org)
	assert.Equal(t, "Test Organization", org.Name)
	assert.Equal(t, "Test Description", org.Description)
	assert.Equal(t, "test-organization", org.Slug)
	assert.True(t, org.IsActive)

	mockOrgRepo.AssertExpectations(t)
}

func TestOrganizationService_CreateOrganization_NameExists(t *testing.T) {
	service, mockOrgRepo := setupOrganizationService()
	ctx := context.Background()

	req := &CreateOrganizationRequest{
		Name: "Existing Organization",
	}

	existingOrg := &models.Organization{ID: 1, Name: "Existing Organization"}

	// Mock repository calls
	mockOrgRepo.On("SearchByName", ctx, "Existing Organization", 0, 1).Return([]*models.Organization{existingOrg}, nil)

	// Execute
	org, err := service.CreateOrganization(ctx, req)

	// Assert
	require.Error(t, err)
	assert.Nil(t, org)
	assert.Contains(t, err.Error(), "organization name already exists")

	mockOrgRepo.AssertExpectations(t)
}

func TestOrganizationService_CreateOrganization_SlugExists(t *testing.T) {
	service, mockOrgRepo := setupOrganizationService()
	ctx := context.Background()

	req := &CreateOrganizationRequest{
		Name: "New Organization",
	}

	existingOrg := &models.Organization{ID: 1, Slug: "new-organization"}

	// Mock repository calls
	mockOrgRepo.On("SearchByName", ctx, "New Organization", 0, 1).Return([]*models.Organization{}, nil)
	mockOrgRepo.On("GetBySlug", ctx, "new-organization").Return(existingOrg, nil)

	// Execute
	org, err := service.CreateOrganization(ctx, req)

	// Assert
	require.Error(t, err)
	assert.Nil(t, org)
	assert.Contains(t, err.Error(), "organization slug already exists")

	mockOrgRepo.AssertExpectations(t)
}

func TestOrganizationService_CreateOrganization_InvalidName(t *testing.T) {
	service, _ := setupOrganizationService()
	ctx := context.Background()

	req := &CreateOrganizationRequest{
		Name: "", // Empty name
	}

	// Execute
	org, err := service.CreateOrganization(ctx, req)

	// Assert
	require.Error(t, err)
	assert.Nil(t, org)
	assert.Contains(t, err.Error(), "validation failed")
}

// GetOrganizationByID Tests.
func TestOrganizationService_GetOrganizationByID_Success(t *testing.T) {
	service, mockOrgRepo := setupOrganizationService()
	ctx := context.Background()

	expectedOrg := &models.Organization{
		ID:          1,
		Name:        "Test Organization",
		Slug:        "test-organization",
		Description: "Test Description",
	}

	// Mock repository calls
	mockOrgRepo.On("GetByID", ctx, uint(1)).Return(expectedOrg, nil)

	// Execute
	org, err := service.GetOrganizationByID(ctx, 1)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, expectedOrg, org)

	mockOrgRepo.AssertExpectations(t)
}

func TestOrganizationService_GetOrganizationByID_NotFound(t *testing.T) {
	service, mockOrgRepo := setupOrganizationService()
	ctx := context.Background()

	// Mock repository calls
	mockOrgRepo.On("GetByID", ctx, uint(999)).Return(nil, nil)

	// Execute
	org, err := service.GetOrganizationByID(ctx, 999)

	// Assert
	require.Error(t, err)
	assert.Nil(t, org)
	assert.Contains(t, err.Error(), "organization not found")

	mockOrgRepo.AssertExpectations(t)
}

// GetOrganizationBySlug Tests.
func TestOrganizationService_GetOrganizationBySlug_Success(t *testing.T) {
	service, mockOrgRepo := setupOrganizationService()
	ctx := context.Background()

	expectedOrg := &models.Organization{
		ID:   1,
		Name: "Test Organization",
		Slug: "test-organization",
	}

	// Mock repository calls
	mockOrgRepo.On("GetBySlug", ctx, "test-organization").Return(expectedOrg, nil)

	// Execute
	org, err := service.GetOrganizationBySlug(ctx, "test-organization")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, expectedOrg, org)

	mockOrgRepo.AssertExpectations(t)
}

func TestOrganizationService_GetOrganizationBySlug_EmptySlug(t *testing.T) {
	service, _ := setupOrganizationService()
	ctx := context.Background()

	// Execute
	org, err := service.GetOrganizationBySlug(ctx, "")

	// Assert
	require.Error(t, err)
	assert.Nil(t, org)
	assert.Contains(t, err.Error(), "slug cannot be empty")
}

// UpdateOrganization Tests.
func TestOrganizationService_UpdateOrganization_Success(t *testing.T) {
	service, mockOrgRepo := setupOrganizationService()
	ctx := context.Background()

	existingOrg := &models.Organization{
		ID:          1,
		Name:        "Old Organization",
		Slug:        "old-organization",
		Description: "Old Description",
		IsActive:    true,
	}

	newName := "New Organization"
	newDescription := "New Description"
	req := &UpdateOrganizationRequest{
		Name:        &newName,
		Description: &newDescription,
	}

	// Mock repository calls
	mockOrgRepo.On("GetByID", ctx, uint(1)).Return(existingOrg, nil)
	mockOrgRepo.On("SearchByName", ctx, "New Organization", 0, 1).Return([]*models.Organization{}, nil)
	mockOrgRepo.On("Update", ctx, mock.AnythingOfType("*models.Organization")).Return(nil)

	// Execute
	org, err := service.UpdateOrganization(ctx, 1, req)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, org)
	assert.Equal(t, "New Organization", org.Name)
	assert.Equal(t, "New Description", org.Description)
	assert.Equal(t, "new-organization", org.Slug) // Slug should be regenerated

	mockOrgRepo.AssertExpectations(t)
}

func TestOrganizationService_UpdateOrganization_NotFound(t *testing.T) {
	service, mockOrgRepo := setupOrganizationService()
	ctx := context.Background()

	req := &UpdateOrganizationRequest{
		Name: stringPtr("New Name"),
	}

	// Mock repository calls
	mockOrgRepo.On("GetByID", ctx, uint(999)).Return(nil, nil)

	// Execute
	org, err := service.UpdateOrganization(ctx, 999, req)

	// Assert
	require.Error(t, err)
	assert.Nil(t, org)
	assert.Contains(t, err.Error(), "organization not found")

	mockOrgRepo.AssertExpectations(t)
}

// DeleteOrganization Tests.
func TestOrganizationService_DeleteOrganization_Success(t *testing.T) {
	service, mockOrgRepo := setupOrganizationService()
	ctx := context.Background()

	existingOrg := &models.Organization{ID: 1, Name: "Test Organization"}

	// Mock repository calls
	mockOrgRepo.On("GetByID", ctx, uint(1)).Return(existingOrg, nil)
	mockOrgRepo.On("Delete", ctx, uint(1)).Return(nil)

	// Execute
	err := service.DeleteOrganization(ctx, 1)

	// Assert
	require.NoError(t, err)

	mockOrgRepo.AssertExpectations(t)
}

// ListOrganizations Tests.
func TestOrganizationService_ListOrganizations_Success(t *testing.T) {
	service, mockOrgRepo := setupOrganizationService()
	ctx := context.Background()

	expectedOrgs := []*models.Organization{
		{ID: 1, Name: "Organization 1", Slug: "organization-1"},
		{ID: 2, Name: "Organization 2", Slug: "organization-2"},
	}

	req := &ListOrganizationsRequest{
		Offset: 0,
		Limit:  10,
	}

	// Mock repository calls
	mockOrgRepo.On("List", ctx, 0, 10).Return(expectedOrgs, nil)

	// Execute
	orgs, err := service.ListOrganizations(ctx, req)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, expectedOrgs, orgs)
	assert.Len(t, orgs, 2)

	mockOrgRepo.AssertExpectations(t)
}

func TestOrganizationService_ListOrganizations_InvalidLimit(t *testing.T) {
	service, _ := setupOrganizationService()
	ctx := context.Background()

	req := &ListOrganizationsRequest{
		Offset: 0,
		Limit:  0, // Invalid limit
	}

	// Execute
	orgs, err := service.ListOrganizations(ctx, req)

	// Assert
	require.Error(t, err)
	assert.Nil(t, orgs)
	assert.Contains(t, err.Error(), "validation failed")
}

// CountOrganizations Tests.
func TestOrganizationService_CountOrganizations_Success(t *testing.T) {
	service, mockOrgRepo := setupOrganizationService()
	ctx := context.Background()

	// Mock repository calls
	mockOrgRepo.On("Count", ctx).Return(int64(25), nil)

	// Execute
	count, err := service.CountOrganizations(ctx)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, int64(25), count)

	mockOrgRepo.AssertExpectations(t)
}

// ActivateOrganization Tests.
func TestOrganizationService_ActivateOrganization_Success(t *testing.T) {
	service, mockOrgRepo := setupOrganizationService()
	ctx := context.Background()

	org := &models.Organization{
		ID:       1,
		Name:     "Test Organization",
		IsActive: false,
	}

	// Mock repository calls
	mockOrgRepo.On("GetByID", ctx, uint(1)).Return(org, nil)
	mockOrgRepo.On("Update", ctx, org).Return(nil)

	// Execute
	err := service.ActivateOrganization(ctx, 1)

	// Assert
	require.NoError(t, err)
	assert.True(t, org.IsActive)

	mockOrgRepo.AssertExpectations(t)
}

func TestOrganizationService_DeactivateOrganization_Success(t *testing.T) {
	service, mockOrgRepo := setupOrganizationService()
	ctx := context.Background()

	org := &models.Organization{
		ID:       1,
		Name:     "Test Organization",
		IsActive: true,
	}

	// Mock repository calls
	mockOrgRepo.On("GetByID", ctx, uint(1)).Return(org, nil)
	mockOrgRepo.On("Update", ctx, org).Return(nil)

	// Execute
	err := service.DeactivateOrganization(ctx, 1)

	// Assert
	require.NoError(t, err)
	assert.False(t, org.IsActive)

	mockOrgRepo.AssertExpectations(t)
}

// SearchOrganizations Tests.
func TestOrganizationService_SearchOrganizations_Success(t *testing.T) {
	service, mockOrgRepo := setupOrganizationService()
	ctx := context.Background()

	expectedOrgs := []*models.Organization{
		{ID: 1, Name: "Test Organization", Slug: "test-organization"},
	}

	req := &SearchOrganizationsRequest{
		Query:        "test",
		SearchFields: []string{"name"},
		Offset:       0,
		Limit:        10,
	}

	// Mock repository calls
	mockOrgRepo.On("SearchByName", ctx, "test", 0, 10).Return(expectedOrgs, nil)

	// Execute
	orgs, err := service.SearchOrganizations(ctx, req)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, expectedOrgs, orgs)

	mockOrgRepo.AssertExpectations(t)
}

// GetActiveOrganizations Tests.

// AddUserToOrganization Tests.
func TestOrganizationService_AddUserToOrganization_Success(t *testing.T) {
	service, mockOrgRepo := setupOrganizationService()
	ctx := context.Background()

	// Mock repository calls
	mockOrgRepo.On("AddUser", ctx, uint(1), uint(2)).Return(nil)

	// Execute
	err := service.AddUserToOrganization(ctx, 1, 2)

	// Assert
	require.NoError(t, err)

	mockOrgRepo.AssertExpectations(t)
}

// RemoveUserFromOrganization Tests.
func TestOrganizationService_RemoveUserFromOrganization_Success(t *testing.T) {
	service, mockOrgRepo := setupOrganizationService()
	ctx := context.Background()

	// Mock repository calls
	mockOrgRepo.On("RemoveUser", ctx, uint(1), uint(2)).Return(nil)

	// Execute
	err := service.RemoveUserFromOrganization(ctx, 1, 2)

	// Assert
	require.NoError(t, err)

	mockOrgRepo.AssertExpectations(t)
}

// GetOrganizationMembers Tests.
func TestOrganizationService_GetOrganizationMembers_Success(t *testing.T) {
	service, mockOrgRepo := setupOrganizationService()
	ctx := context.Background()

	expectedUsers := []*models.User{
		{ID: 1, Email: "user1@example.com"},
		{ID: 2, Email: "user2@example.com"},
	}

	// Mock repository calls
	mockOrgRepo.On("GetMembers", ctx, uint(1), 0, 10).Return(expectedUsers, nil)

	// Execute
	users, err := service.GetOrganizationMembers(ctx, 1, 0, 10)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, expectedUsers, users)

	mockOrgRepo.AssertExpectations(t)
}

// CountOrganizationMembers Tests.
func TestOrganizationService_CountOrganizationMembers_Success(t *testing.T) {
	service, mockOrgRepo := setupOrganizationService()
	ctx := context.Background()

	// Mock repository calls
	mockOrgRepo.On("CountMembers", ctx, uint(1)).Return(int64(5), nil)

	// Execute
	count, err := service.GetOrganizationMemberCount(ctx, 1)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, int64(5), count)

	mockOrgRepo.AssertExpectations(t)
}
