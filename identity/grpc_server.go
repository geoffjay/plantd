package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	commonv1 "github.com/geoffjay/plantd/gen/proto/go/plantd/common/v1"
	healthv1 "github.com/geoffjay/plantd/gen/proto/go/plantd/health/v1"
	identityv1 "github.com/geoffjay/plantd/gen/proto/go/plantd/identity/v1"
	"github.com/geoffjay/plantd/identity/internal/auth"
	"github.com/geoffjay/plantd/identity/internal/models"
	"github.com/geoffjay/plantd/identity/internal/services"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
	ErrUnauthorized  = errors.New("unauthorized")
	ErrForbidden     = errors.New("forbidden")
)

// IdentityGRPCServer implements the Identity service gRPC interface
type IdentityGRPCServer struct {
	authService         *auth.AuthService
	userService         services.UserService
	organizationService services.OrganizationService
	roleService         services.RoleService
	rbacService         *auth.RBACService
	registrationService *auth.RegistrationService
	startTime           time.Time
}

// NewIdentityGRPCServer creates a new Identity gRPC server
func NewIdentityGRPCServer(
	authService *auth.AuthService,
	userService services.UserService,
	organizationService services.OrganizationService,
	roleService services.RoleService,
	rbacService *auth.RBACService,
	registrationService *auth.RegistrationService,
) *IdentityGRPCServer {
	return &IdentityGRPCServer{
		authService:         authService,
		userService:         userService,
		organizationService: organizationService,
		roleService:         roleService,
		rbacService:         rbacService,
		registrationService: registrationService,
		startTime:           time.Now(),
	}
}

// Login authenticates a user and returns tokens
func (s *IdentityGRPCServer) Login(ctx context.Context, req *connect.Request[identityv1.LoginRequest]) (*connect.Response[identityv1.LoginResponse], error) {
	log.Printf("Identity.Login: email=%s organization_id=%s", req.Msg.Email, req.Msg.OrganizationId)

	// Convert protobuf request to auth service request
	authReq := &auth.AuthRequest{
		Identifier: req.Msg.Email,
		Password:   req.Msg.Password,
		IPAddress:  getClientIP(req),
		UserAgent:  getUserAgent(req),
	}

	// Authenticate user
	authResp, err := s.authService.Login(ctx, authReq)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication failed: %w", err))
	}

	// Convert user model to protobuf
	userProto := &identityv1.User{
		Id:             fmt.Sprintf("%d", authResp.User.ID),
		Email:          authResp.User.Email,
		Name:           fmt.Sprintf("%s %s", authResp.User.FirstName, authResp.User.LastName),
		OrganizationId: req.Msg.OrganizationId, // Use requested org ID for now
		Active:         authResp.User.IsActive,
		Metadata: &commonv1.Metadata{
			Id:        fmt.Sprintf("user/%d", authResp.User.ID),
			CreatedAt: timestamppb.New(authResp.User.CreatedAt),
			UpdatedAt: timestamppb.New(authResp.User.UpdatedAt),
			Version:   1,
		},
	}

	if authResp.User.LastLoginAt != nil {
		userProto.LastLogin = timestamppb.New(*authResp.User.LastLoginAt)
	}

	// Get user roles
	roles, err := s.userService.GetUserRoles(ctx, authResp.User.ID)
	if err == nil {
		roleNames := make([]string, len(roles))
		for i, role := range roles {
			if role != nil {
				roleNames[i] = role.Name
			}
		}
		userProto.Roles = roleNames
	}

	response := &identityv1.LoginResponse{
		AccessToken:  authResp.TokenPair.AccessToken,
		RefreshToken: authResp.TokenPair.RefreshToken,
		ExpiresAt:    timestamppb.New(authResp.ExpiresAt),
		User:         userProto,
	}

	return connect.NewResponse(response), nil
}

// Logout invalidates a user's token
func (s *IdentityGRPCServer) Logout(ctx context.Context, req *connect.Request[identityv1.LogoutRequest]) (*connect.Response[emptypb.Empty], error) {
	log.Printf("Identity.Logout: token=%s...", req.Msg.Token[:min(len(req.Msg.Token), 10)])

	err := s.authService.Logout(ctx, req.Msg.Token)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("logout failed: %w", err))
	}

	return connect.NewResponse(&emptypb.Empty{}), nil
}

// RefreshToken generates new tokens using a refresh token
func (s *IdentityGRPCServer) RefreshToken(ctx context.Context, req *connect.Request[identityv1.RefreshTokenRequest]) (*connect.Response[identityv1.RefreshTokenResponse], error) {
	log.Printf("Identity.RefreshToken: refresh_token=%s...", req.Msg.RefreshToken[:min(len(req.Msg.RefreshToken), 10)])

	refreshReq := &auth.RefreshRequest{
		RefreshToken: req.Msg.RefreshToken,
		IPAddress:    getClientIP(req),
	}

	tokenPair, err := s.authService.RefreshToken(ctx, refreshReq)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("token refresh failed: %w", err))
	}

	response := &identityv1.RefreshTokenResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    timestamppb.New(tokenPair.AccessTokenExpiresAt),
	}

	return connect.NewResponse(response), nil
}

// ValidateToken validates an access token and returns user info
func (s *IdentityGRPCServer) ValidateToken(ctx context.Context, req *connect.Request[identityv1.ValidateTokenRequest]) (*connect.Response[identityv1.ValidateTokenResponse], error) {
	log.Printf("Identity.ValidateToken: token=%s...", req.Msg.Token[:min(len(req.Msg.Token), 10)])

	claims, err := s.authService.ValidateToken(ctx, req.Msg.Token)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("token validation failed: %w", err))
	}

	// Get user details
	user, err := s.userService.GetUserByID(ctx, claims.UserID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get user: %w", err))
	}

	// Convert user to protobuf
	userProto := &identityv1.User{
		Id:     fmt.Sprintf("%d", user.ID),
		Email:  user.Email,
		Name:   fmt.Sprintf("%s %s", user.FirstName, user.LastName),
		Active: user.IsActive,
		Roles:  claims.Roles,
		Metadata: &commonv1.Metadata{
			Id:        fmt.Sprintf("user/%d", user.ID),
			CreatedAt: timestamppb.New(user.CreatedAt),
			UpdatedAt: timestamppb.New(user.UpdatedAt),
			Version:   1,
		},
	}

	// Check required permissions if provided
	hasAllPermissions := true
	if len(req.Msg.RequiredPermissions) > 0 {
		for _, permission := range req.Msg.RequiredPermissions {
			has, err := s.rbacService.HasPermission(ctx, claims.UserID, auth.Permission(permission), nil)
			if err != nil || !has {
				hasAllPermissions = false
				break
			}
		}
	}

	response := &identityv1.ValidateTokenResponse{
		Valid:       hasAllPermissions,
		User:        userProto,
		Permissions: claims.Permissions,
		ExpiresAt:   timestamppb.New(claims.ExpiresAt.Time),
	}

	return connect.NewResponse(response), nil
}

// CreateUser creates a new user account
func (s *IdentityGRPCServer) CreateUser(ctx context.Context, req *connect.Request[identityv1.CreateUserRequest]) (*connect.Response[identityv1.CreateUserResponse], error) {
	log.Printf("Identity.CreateUser: email=%s name=%s", req.Msg.Email, req.Msg.Name)

	// Parse name into first and last name
	firstName, lastName := parseUserName(req.Msg.Name)

	// Convert to registration request
	regReq := &auth.RegistrationRequest{
		Email:     req.Msg.Email,
		Username:  generateUsername(req.Msg.Email), // Generate username from email
		Password:  req.Msg.Password,
		FirstName: firstName,
		LastName:  lastName,
		IPAddress: getClientIP(req),
		UserAgent: getUserAgent(req),
	}

	regResp, err := s.registrationService.Register(ctx, regReq)
	if err != nil {
		if strings.Contains(err.Error(), "email address is already registered") ||
			strings.Contains(err.Error(), "username is already taken") {
			return nil, connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("user already exists: %w", err))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("user creation failed: %w", err))
	}

	// Convert user to protobuf
	userProto := &identityv1.User{
		Id:             fmt.Sprintf("%d", regResp.User.ID),
		Email:          regResp.User.Email,
		Name:           fmt.Sprintf("%s %s", regResp.User.FirstName, regResp.User.LastName),
		OrganizationId: req.Msg.OrganizationId,
		Active:         regResp.User.IsActive,
		Roles:          req.Msg.Roles, // Use requested roles for now
		Metadata: &commonv1.Metadata{
			Id:        fmt.Sprintf("user/%d", regResp.User.ID),
			CreatedAt: timestamppb.New(regResp.User.CreatedAt),
			UpdatedAt: timestamppb.New(regResp.User.UpdatedAt),
			Version:   1,
		},
	}

	response := &identityv1.CreateUserResponse{
		User: userProto,
	}

	return connect.NewResponse(response), nil
}

// GetUser retrieves user information by ID
func (s *IdentityGRPCServer) GetUser(ctx context.Context, req *connect.Request[identityv1.GetUserRequest]) (*connect.Response[identityv1.GetUserResponse], error) {
	log.Printf("Identity.GetUser: id=%s", req.Msg.Id)

	userID, err := strconv.ParseUint(req.Msg.Id, 10, 32)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid user ID: %w", err))
	}

	user, err := s.userService.GetUserByID(ctx, uint(userID))
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("user not found: %w", err))
	}

	// Get user roles
	roles, err := s.userService.GetUserRoles(ctx, user.ID)
	if err != nil {
		log.Printf("Failed to get user roles: %v", err)
		roles = []*models.Role{} // Continue with empty roles
	}

	roleNames := make([]string, len(roles))
	for i, role := range roles {
		if role != nil {
			roleNames[i] = role.Name
		}
	}

	// Convert user to protobuf
	userProto := &identityv1.User{
		Id:     fmt.Sprintf("%d", user.ID),
		Email:  user.Email,
		Name:   fmt.Sprintf("%s %s", user.FirstName, user.LastName),
		Active: user.IsActive,
		Roles:  roleNames,
		Metadata: &commonv1.Metadata{
			Id:        fmt.Sprintf("user/%d", user.ID),
			CreatedAt: timestamppb.New(user.CreatedAt),
			UpdatedAt: timestamppb.New(user.UpdatedAt),
			Version:   1,
		},
	}

	if user.LastLoginAt != nil {
		userProto.LastLogin = timestamppb.New(*user.LastLoginAt)
	}

	response := &identityv1.GetUserResponse{
		User: userProto,
	}

	return connect.NewResponse(response), nil
}

// UpdateUser updates user information
func (s *IdentityGRPCServer) UpdateUser(ctx context.Context, req *connect.Request[identityv1.UpdateUserRequest]) (*connect.Response[identityv1.UpdateUserResponse], error) {
	log.Printf("Identity.UpdateUser: id=%s name=%s", req.Msg.Id, req.Msg.Name)

	userID, err := strconv.ParseUint(req.Msg.Id, 10, 32)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid user ID: %w", err))
	}

	// Parse name into first and last name
	firstName, lastName := parseUserName(req.Msg.Name)

	// Convert to update request
	updateReq := &services.UpdateUserRequest{
		Email:     &req.Msg.Email,
		FirstName: &firstName,
		LastName:  &lastName,
		IsActive:  &req.Msg.Active,
	}

	user, err := s.userService.UpdateUser(ctx, uint(userID), updateReq)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("user update failed: %w", err))
	}

	// Convert user to protobuf
	userProto := &identityv1.User{
		Id:     fmt.Sprintf("%d", user.ID),
		Email:  user.Email,
		Name:   fmt.Sprintf("%s %s", user.FirstName, user.LastName),
		Active: user.IsActive,
		Roles:  req.Msg.Roles, // Use requested roles for now
		Metadata: &commonv1.Metadata{
			Id:        fmt.Sprintf("user/%d", user.ID),
			CreatedAt: timestamppb.New(user.CreatedAt),
			UpdatedAt: timestamppb.New(user.UpdatedAt),
			Version:   1,
		},
	}

	response := &identityv1.UpdateUserResponse{
		User: userProto,
	}

	return connect.NewResponse(response), nil
}

// DeleteUser deletes a user account
func (s *IdentityGRPCServer) DeleteUser(ctx context.Context, req *connect.Request[identityv1.DeleteUserRequest]) (*connect.Response[emptypb.Empty], error) {
	log.Printf("Identity.DeleteUser: id=%s", req.Msg.Id)

	userID, err := strconv.ParseUint(req.Msg.Id, 10, 32)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid user ID: %w", err))
	}

	err = s.userService.DeleteUser(ctx, uint(userID))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("user deletion failed: %w", err))
	}

	return connect.NewResponse(&emptypb.Empty{}), nil
}

// ListUsers streams user information with pagination
func (s *IdentityGRPCServer) ListUsers(ctx context.Context, req *connect.Request[identityv1.ListUsersRequest], stream *connect.ServerStream[identityv1.ListUsersResponse]) error {
	log.Printf("Identity.ListUsers: organization_id=%s", req.Msg.OrganizationId)

	// Set up pagination defaults
	offset := 0
	limit := 50
	if req.Msg.Page != nil {
		if req.Msg.Page.PageSize > 0 {
			limit = int(req.Msg.Page.PageSize)
		}
		// TODO: Handle page_token for proper pagination in future
		// For now, we'll use simple offset-based pagination
	}

	// List users
	listReq := &services.ListUsersRequest{
		Offset: offset,
		Limit:  limit,
	}

	users, err := s.userService.ListUsers(ctx, listReq)
	if err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list users: %w", err))
	}

	// Stream users
	for _, user := range users {
		if user == nil {
			continue
		}

		// Get user roles
		roles, err := s.userService.GetUserRoles(ctx, user.ID)
		if err != nil {
			log.Printf("Failed to get user roles for user %d: %v", user.ID, err)
			roles = []*models.Role{} // Continue with empty roles
		}

		roleNames := make([]string, len(roles))
		for j, role := range roles {
			if role != nil {
				roleNames[j] = role.Name
			}
		}

		userProto := &identityv1.User{
			Id:             fmt.Sprintf("%d", user.ID),
			Email:          user.Email,
			Name:           fmt.Sprintf("%s %s", user.FirstName, user.LastName),
			OrganizationId: req.Msg.OrganizationId,
			Active:         user.IsActive,
			Roles:          roleNames,
			Metadata: &commonv1.Metadata{
				Id:        fmt.Sprintf("user/%d", user.ID),
				CreatedAt: timestamppb.New(user.CreatedAt),
				UpdatedAt: timestamppb.New(user.UpdatedAt),
				Version:   1,
			},
		}

		if user.LastLoginAt != nil {
			userProto.LastLogin = timestamppb.New(*user.LastLoginAt)
		}

		response := &identityv1.ListUsersResponse{
			User: userProto,
			Page: &commonv1.PageResponse{
				NextPageToken: "", // TODO: Implement proper token-based pagination
				TotalCount:    int32(len(users)),
			},
		}

		if err := stream.Send(response); err != nil {
			return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to send user: %w", err))
		}
	}

	return nil
}

// CreateOrganization creates a new organization
func (s *IdentityGRPCServer) CreateOrganization(ctx context.Context, req *connect.Request[identityv1.CreateOrganizationRequest]) (*connect.Response[identityv1.CreateOrganizationResponse], error) {
	log.Printf("Identity.CreateOrganization: name=%s domain=%s", req.Msg.Name, req.Msg.Domain)

	// Convert to organization service request
	createReq := &services.CreateOrganizationRequest{
		Name:        req.Msg.Name,
		Slug:        generateSlug(req.Msg.Name),
		Description: fmt.Sprintf("Organization for %s", req.Msg.Domain),
	}

	org, err := s.organizationService.CreateOrganization(ctx, createReq)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("organization creation failed: %w", err))
	}

	// Convert organization to protobuf
	orgProto := &identityv1.Organization{
		Id:     fmt.Sprintf("%d", org.ID),
		Name:   org.Name,
		Domain: org.Slug, // Use slug as domain for now
		Metadata: &commonv1.Metadata{
			Id:        fmt.Sprintf("organization/%d", org.ID),
			CreatedAt: timestamppb.New(org.CreatedAt),
			UpdatedAt: timestamppb.New(org.UpdatedAt),
			Version:   1,
		},
	}

	response := &identityv1.CreateOrganizationResponse{
		Organization: orgProto,
	}

	return connect.NewResponse(response), nil
}

// GetOrganization retrieves organization information by ID
func (s *IdentityGRPCServer) GetOrganization(ctx context.Context, req *connect.Request[identityv1.GetOrganizationRequest]) (*connect.Response[identityv1.GetOrganizationResponse], error) {
	log.Printf("Identity.GetOrganization: id=%s", req.Msg.Id)

	orgID, err := strconv.ParseUint(req.Msg.Id, 10, 32)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid organization ID: %w", err))
	}

	org, err := s.organizationService.GetOrganizationByID(ctx, uint(orgID))
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("organization not found: %w", err))
	}

	// Convert organization to protobuf
	orgProto := &identityv1.Organization{
		Id:     fmt.Sprintf("%d", org.ID),
		Name:   org.Name,
		Domain: org.Slug, // Use slug as domain for now
		Metadata: &commonv1.Metadata{
			Id:        fmt.Sprintf("organization/%d", org.ID),
			CreatedAt: timestamppb.New(org.CreatedAt),
			UpdatedAt: timestamppb.New(org.UpdatedAt),
			Version:   1,
		},
	}

	response := &identityv1.GetOrganizationResponse{
		Organization: orgProto,
	}

	return connect.NewResponse(response), nil
}

// AssignRole assigns a role to a user
func (s *IdentityGRPCServer) AssignRole(ctx context.Context, req *connect.Request[identityv1.AssignRoleRequest]) (*connect.Response[emptypb.Empty], error) {
	log.Printf("Identity.AssignRole: user_id=%s role=%s", req.Msg.UserId, req.Msg.Role)

	userID, err := strconv.ParseUint(req.Msg.UserId, 10, 32)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid user ID: %w", err))
	}

	// For now, return not implemented as role assignment needs more complex implementation
	_ = userID
	return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("role assignment not yet implemented"))
}

// RevokeRole revokes a role from a user
func (s *IdentityGRPCServer) RevokeRole(ctx context.Context, req *connect.Request[identityv1.RevokeRoleRequest]) (*connect.Response[emptypb.Empty], error) {
	log.Printf("Identity.RevokeRole: user_id=%s role=%s", req.Msg.UserId, req.Msg.Role)

	userID, err := strconv.ParseUint(req.Msg.UserId, 10, 32)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid user ID: %w", err))
	}

	// For now, return not implemented
	_ = userID
	return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("role revocation not yet implemented"))
}

// CheckPermission checks if a user has a specific permission
func (s *IdentityGRPCServer) CheckPermission(ctx context.Context, req *connect.Request[identityv1.CheckPermissionRequest]) (*connect.Response[identityv1.CheckPermissionResponse], error) {
	log.Printf("Identity.CheckPermission: user_id=%s permission=%s", req.Msg.UserId, req.Msg.Permission)

	userID, err := strconv.ParseUint(req.Msg.UserId, 10, 32)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid user ID: %w", err))
	}

	// Parse organization ID if provided
	var orgID *uint
	if req.Msg.OrganizationId != "" {
		orgIDParsed, err := strconv.ParseUint(req.Msg.OrganizationId, 10, 32)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid organization ID: %w", err))
		}
		orgIDUint := uint(orgIDParsed)
		orgID = &orgIDUint
	}

	hasPermission, err := s.rbacService.HasPermission(ctx, uint(userID), auth.Permission(req.Msg.Permission), orgID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("permission check failed: %w", err))
	}

	response := &identityv1.CheckPermissionResponse{
		Allowed: hasPermission,
	}

	return connect.NewResponse(response), nil
}

// Health returns the health status of the Identity service
func (s *IdentityGRPCServer) Health(ctx context.Context, req *connect.Request[emptypb.Empty]) (*connect.Response[healthv1.HealthCheckResponse], error) {
	// Perform basic health checks
	healthy := true
	message := "Identity service is healthy"

	// Try to check database connectivity by getting user count
	if s.userService != nil {
		_, err := s.userService.CountUsers(ctx)
		if err != nil {
			healthy = false
			message = "Identity service is unhealthy - database connectivity issues"
		}
	}

	status := healthv1.HealthCheckResponse_SERVING_STATUS_SERVING
	if !healthy {
		status = healthv1.HealthCheckResponse_SERVING_STATUS_NOT_SERVING
	}

	response := &healthv1.HealthCheckResponse{
		Status:    status,
		Message:   message,
		Timestamp: timestamppb.Now(),
		Uptime:    durationpb.New(time.Since(s.startTime)),
	}

	return connect.NewResponse(response), nil
}

// Helper functions

func getClientIP(req interface{}) string {
	// Extract client IP from request headers if available
	// For now, return empty string
	return ""
}

func getUserAgent(req interface{}) string {
	// Extract user agent from request headers if available
	// For now, return empty string
	return ""
}

func parseUserName(fullName string) (firstName, lastName string) {
	parts := strings.Fields(fullName)
	if len(parts) == 0 {
		return "", ""
	}
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], strings.Join(parts[1:], " ")
}

func generateUsername(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) > 0 {
		return parts[0]
	}
	return email
}

func generateSlug(name string) string {
	return strings.ToLower(strings.ReplaceAll(name, " ", "-"))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
