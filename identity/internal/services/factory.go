// Package services provides the implementation of the service layer.
package services

import (
	"github.com/geoffjay/plantd/identity/internal/repositories"
)

// ServiceContainer holds all service implementations.
type ServiceContainer struct {
	UserService         UserService
	OrganizationService OrganizationService
	RoleService         RoleService
}

// NewServiceContainer creates a new service container with all service dependencies wired up.
func NewServiceContainer(repos *repositories.Container) *ServiceContainer {
	return &ServiceContainer{
		UserService:         NewUserService(repos.User, repos.Role, repos.Organization),
		OrganizationService: NewOrganizationService(repos.Organization, repos.User, repos.Role),
		RoleService:         NewRoleService(repos.Role, repos.User, repos.Organization),
	}
}

// ServiceFactory provides a factory pattern for creating services.
type ServiceFactory struct {
	repos *repositories.Container
}

// NewServiceFactory creates a new service factory.
func NewServiceFactory(repos *repositories.Container) *ServiceFactory {
	return &ServiceFactory{
		repos: repos,
	}
}

// CreateUserService creates a new UserService instance.
func (f *ServiceFactory) CreateUserService() UserService {
	return NewUserService(f.repos.User, f.repos.Role, f.repos.Organization)
}

// CreateOrganizationService creates a new OrganizationService instance.
func (f *ServiceFactory) CreateOrganizationService() OrganizationService {
	return NewOrganizationService(f.repos.Organization, f.repos.User, f.repos.Role)
}

// CreateRoleService creates a new RoleService instance.
func (f *ServiceFactory) CreateRoleService() RoleService {
	return NewRoleService(f.repos.Role, f.repos.User, f.repos.Organization)
}

// CreateAllServices creates all services and returns them in a container.
func (f *ServiceFactory) CreateAllServices() *ServiceContainer {
	return NewServiceContainer(f.repos)
}
