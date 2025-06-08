package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestOrganization_TableName(t *testing.T) {
	org := Organization{}
	assert.Equal(t, "organizations", org.TableName())
}

func TestOrganization_GenerateSlug(t *testing.T) {
	tests := []struct {
		name         string
		orgName      string
		expectedSlug string
	}{
		{
			name:         "simple name",
			orgName:      "Test Organization",
			expectedSlug: "test-organization",
		},
		{
			name:         "name with special characters",
			orgName:      "Test & Co. LLC",
			expectedSlug: "test-co-llc",
		},
		{
			name:         "name with numbers",
			orgName:      "Company 123",
			expectedSlug: "company-123",
		},
		{
			name:         "name with extra spaces",
			orgName:      "  Test   Organization  ",
			expectedSlug: "test-organization",
		},
		{
			name:         "name with underscores",
			orgName:      "Test_Organization_Name",
			expectedSlug: "test-organization-name",
		},
		{
			name:         "already lowercase",
			orgName:      "test-organization",
			expectedSlug: "test-organization",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			org := &Organization{Name: tt.orgName}
			org.GenerateSlug()
			assert.Equal(t, tt.expectedSlug, org.Slug)
		})
	}
}

func TestOrganization_DatabaseConstraints(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	t.Run("name must be unique", func(t *testing.T) {
		// Create first organization
		org1 := createTestOrganization(t, db, func(o *Organization) {
			o.Name = "Unique Org"
			o.Slug = "unique-org"
		})
		assert.NotZero(t, org1.ID)

		// Try to create second organization with same name
		org2 := &Organization{
			Name:        "Unique Org",
			Slug:        "different-slug",
			Description: "Different description",
			IsActive:    true,
		}

		err := db.Create(org2).Error
		assert.Error(t, err)
	})

	t.Run("slug must be unique", func(t *testing.T) {
		// Create first organization
		org1 := createTestOrganization(t, db, func(o *Organization) {
			o.Name = "First Org"
			o.Slug = "unique-slug"
		})
		assert.NotZero(t, org1.ID)

		// Try to create second organization with same slug
		org2 := &Organization{
			Name:        "Second Org",
			Slug:        "unique-slug",
			Description: "Different description",
			IsActive:    true,
		}

		err := db.Create(org2).Error
		assert.Error(t, err)
	})

	t.Run("soft delete works correctly", func(t *testing.T) {
		org := createTestOrganization(t, db, func(o *Organization) {
			o.Name = "Delete Test Org"
			o.Slug = "delete-test-org"
		})
		orgID := org.ID

		// Delete the organization (soft delete)
		err := db.Delete(org).Error
		require.NoError(t, err)

		// Organization should not be found in normal queries
		var foundOrg Organization
		err = db.First(&foundOrg, orgID).Error
		assert.Error(t, err)
		assert.Equal(t, gorm.ErrRecordNotFound, err)

		// Organization should be found when including deleted records
		err = db.Unscoped().First(&foundOrg, orgID).Error
		require.NoError(t, err)
		assert.NotNil(t, foundOrg.DeletedAt)
	})

	t.Run("name and description cannot be empty", func(t *testing.T) {
		org := &Organization{
			Name:     "",
			Slug:     "empty-name",
			IsActive: true,
		}

		err := db.Create(org).Error
		assert.Error(t, err)
	})
}

func TestOrganization_CRUD_Operations(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	t.Run("create organization", func(t *testing.T) {
		org := &Organization{
			Name:        "Test CRUD Org",
			Slug:        "test-crud-org",
			Description: "A test organization for CRUD operations",
			IsActive:    true,
		}

		err := db.Create(org).Error
		require.NoError(t, err)
		assert.NotZero(t, org.ID)
		assert.NotZero(t, org.CreatedAt)
		assert.NotZero(t, org.UpdatedAt)
	})

	t.Run("read organization", func(t *testing.T) {
		originalOrg := createTestOrganization(t, db, func(o *Organization) {
			o.Name = "Read Test Org"
			o.Slug = "read-test-org"
		})

		var foundOrg Organization
		err := db.First(&foundOrg, originalOrg.ID).Error
		require.NoError(t, err)

		assert.Equal(t, originalOrg.ID, foundOrg.ID)
		assert.Equal(t, originalOrg.Name, foundOrg.Name)
		assert.Equal(t, originalOrg.Slug, foundOrg.Slug)
		assert.Equal(t, originalOrg.Description, foundOrg.Description)
		assert.Equal(t, originalOrg.IsActive, foundOrg.IsActive)
	})

	t.Run("update organization", func(t *testing.T) {
		org := createTestOrganization(t, db, func(o *Organization) {
			o.Name = "Update Test Org"
			o.Slug = "update-test-org"
		})

		// Update the organization
		org.Name = "Updated Organization Name"
		org.Description = "Updated description"
		org.IsActive = false

		err := db.Save(org).Error
		require.NoError(t, err)

		// Verify the update
		var updatedOrg Organization
		err = db.First(&updatedOrg, org.ID).Error
		require.NoError(t, err)

		assert.Equal(t, "Updated Organization Name", updatedOrg.Name)
		assert.Equal(t, "Updated description", updatedOrg.Description)
		assert.False(t, updatedOrg.IsActive)
		assert.True(t, updatedOrg.UpdatedAt.After(updatedOrg.CreatedAt))
	})

	t.Run("find organization by slug", func(t *testing.T) {
		org := createTestOrganization(t, db, func(o *Organization) {
			o.Name = "Find By Slug Org"
			o.Slug = "find-by-slug-org"
		})

		var foundOrg Organization
		err := db.Where("slug = ?", "find-by-slug-org").First(&foundOrg).Error
		require.NoError(t, err)

		assert.Equal(t, org.ID, foundOrg.ID)
		assert.Equal(t, org.Name, foundOrg.Name)
	})
}
