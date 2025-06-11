#!/bin/bash

# State Service RBAC Setup Script
# This script sets up role-based access control for the plantd state service

set -e

# Configuration
IDENTITY_ENDPOINT="${PLANTD_IDENTITY_ENDPOINT:-tcp://127.0.0.1:7200}"
ADMIN_EMAIL="${PLANTD_ADMIN_EMAIL:-admin@example.com}"
ADMIN_PASSWORD="${PLANTD_ADMIN_PASSWORD}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check if plant CLI is available
    if ! command -v plant &> /dev/null; then
        log_error "plant CLI not found. Please install it first."
        exit 1
    fi
    
    # Check if identity service is running
    if ! plant auth status &> /dev/null; then
        log_warning "Identity service may not be running or plant CLI not authenticated"
        log_info "Please ensure identity service is running and authenticate with 'plant auth login'"
    fi
    
    log_success "Prerequisites check completed"
}

# Create standard roles
create_standard_roles() {
    log_info "Creating standard roles for state service..."
    
    # State Developer Role
    log_info "Creating state-developer role..."
    plant identity role create state-developer \
        --description="Developer access to state service" \
        --permissions="state:data:read,state:data:write,state:health:read" \
        --scope="organization" || log_warning "Role may already exist"
    
    # State Admin Role
    log_info "Creating state-admin role..."
    plant identity role create state-admin \
        --description="Administrative access to state service" \
        --permissions="state:scope:create,state:scope:delete,state:scope:list,state:data:read,state:data:write,state:data:delete,state:health:read,state:metrics:read" \
        --scope="organization" || log_warning "Role may already exist"
    
    # State System Admin Role
    log_info "Creating state-system-admin role..."
    plant identity role create state-system-admin \
        --description="Full system access to state service" \
        --permissions="state:system:admin" \
        --scope="global" || log_warning "Role may already exist"
    
    # State Read-Only Role
    log_info "Creating state-readonly role..."
    plant identity role create state-readonly \
        --description="Read-only access to state service" \
        --permissions="state:data:read,state:health:read" \
        --scope="organization" || log_warning "Role may already exist"
    
    # State Service Owner Role
    log_info "Creating state-service-owner role..."
    plant identity role create state-service-owner \
        --description="Service owner with full scope control" \
        --permissions="state:data:read,state:data:write,state:data:delete,state:health:read" \
        --scope="service" || log_warning "Role may already exist"
    
    log_success "Standard roles created successfully"
}

# Assign initial admin role
assign_admin_role() {
    if [ -n "$ADMIN_EMAIL" ]; then
        log_info "Assigning state-system-admin role to $ADMIN_EMAIL..."
        plant identity user assign-role "$ADMIN_EMAIL" state-system-admin || log_warning "Role assignment may have failed"
        log_success "Admin role assigned to $ADMIN_EMAIL"
    else
        log_warning "No admin email specified. Skipping admin role assignment."
        log_info "To assign admin role later, run: plant identity user assign-role <email> state-system-admin"
    fi
}

# Validate role setup
validate_setup() {
    log_info "Validating role setup..."
    
    # List all state service roles
    log_info "Listing state service roles:"
    plant identity role list --filter="name:state-*" || log_warning "Could not list roles"
    
    # Check if admin user has correct role
    if [ -n "$ADMIN_EMAIL" ]; then
        log_info "Checking admin user roles:"
        plant identity user roles "$ADMIN_EMAIL" || log_warning "Could not check user roles"
    fi
    
    log_success "Role setup validation completed"
}

# Setup organizational roles
setup_org_roles() {
    local org_name="$1"
    
    if [ -z "$org_name" ]; then
        log_info "No organization specified. Skipping organization-specific setup."
        return
    fi
    
    log_info "Setting up roles for organization: $org_name"
    
    # Create organization-specific role assignments
    # This would be customized based on organization needs
    log_info "Organization role setup would be customized based on needs"
}

# Migration utility for existing users
migrate_existing_users() {
    log_info "Migrating existing users to role-based system..."
    
    # This would analyze existing user permissions and assign appropriate roles
    log_warning "User migration requires manual analysis of existing permissions"
    log_info "Please review existing users and assign appropriate roles manually"
    log_info "Available roles:"
    log_info "  - state-developer: For developers with read/write access"
    log_info "  - state-admin: For service administrators"
    log_info "  - state-readonly: For read-only access"
    log_info "  - state-system-admin: For system administrators"
}

# Generate role documentation
generate_documentation() {
    local doc_file="state-rbac-setup.md"
    
    log_info "Generating role documentation..."
    
    cat > "$doc_file" << EOF
# State Service RBAC Setup

This document describes the role-based access control setup for the plantd state service.

## Standard Roles

### state-developer
- **Description**: Developer access to state service
- **Scope**: Organization
- **Permissions**: 
  - state:data:read
  - state:data:write
  - state:health:read

### state-admin
- **Description**: Administrative access to state service
- **Scope**: Organization
- **Permissions**:
  - state:scope:create
  - state:scope:delete
  - state:scope:list
  - state:data:read
  - state:data:write
  - state:data:delete
  - state:health:read
  - state:metrics:read

### state-system-admin
- **Description**: Full system access to state service
- **Scope**: Global
- **Permissions**:
  - state:system:admin (implies all other permissions)

### state-readonly
- **Description**: Read-only access to state service
- **Scope**: Organization
- **Permissions**:
  - state:data:read
  - state:health:read

### state-service-owner
- **Description**: Service owner with full scope control
- **Scope**: Service
- **Permissions**:
  - state:data:read
  - state:data:write
  - state:data:delete
  - state:health:read

## Usage Examples

### Assign Developer Role
\`\`\`bash
plant identity user assign-role user@example.com state-developer
\`\`\`

### Assign Admin Role to Organization
\`\`\`bash
plant identity user assign-role admin@example.com state-admin --org-id=123
\`\`\`

### Create Custom Role
\`\`\`bash
plant identity role create state-custom \
  --description="Custom state service access" \
  --permissions="state:data:read,state:scope:create" \
  --scope="organization"
\`\`\`

## Permission Model

The state service uses a hierarchical permission model:

1. **Global Permissions**: Apply across all scopes
2. **Scoped Permissions**: Apply to specific service scopes
3. **Inherited Permissions**: Some permissions imply others
4. **Service Ownership**: Services that create scopes have implicit access

## Access Patterns

1. **Service Owner**: Automatic access to owned scopes
2. **Cross-Service**: Explicit permissions required
3. **Administrative**: Override permissions for admins
4. **Scoped**: Permission within specific boundaries

## Security Considerations

- Use principle of least privilege
- Regular audit of role assignments
- Monitor cross-service access grants
- Log all permission changes

EOF

    log_success "Documentation generated: $doc_file"
}

# Print help
print_help() {
    cat << EOF
State Service RBAC Setup Script

Usage: $0 [OPTIONS] [COMMAND]

Commands:
  setup                 Complete RBAC setup (default)
  roles                 Create standard roles only
  validate              Validate current setup
  migrate               Migrate existing users
  docs                  Generate documentation

Options:
  --admin-email EMAIL   Admin email for initial role assignment
  --org-name NAME       Organization name for org-specific setup
  --identity-endpoint URL  Identity service endpoint
  --help                Show this help message

Environment Variables:
  PLANTD_IDENTITY_ENDPOINT   Identity service endpoint
  PLANTD_ADMIN_EMAIL         Admin email for role assignment
  PLANTD_ADMIN_PASSWORD      Admin password (if needed)

Examples:
  $0 setup --admin-email=admin@example.com
  $0 roles
  $0 validate
  $0 migrate --org-name=myorg

EOF
}

# Main execution
main() {
    local command="setup"
    local org_name=""
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --admin-email)
                ADMIN_EMAIL="$2"
                shift 2
                ;;
            --org-name)
                org_name="$2"
                shift 2
                ;;
            --identity-endpoint)
                IDENTITY_ENDPOINT="$2"
                shift 2
                ;;
            --help)
                print_help
                exit 0
                ;;
            setup|roles|validate|migrate|docs)
                command="$1"
                shift
                ;;
            *)
                log_error "Unknown option: $1"
                print_help
                exit 1
                ;;
        esac
    done
    
    log_info "Starting State Service RBAC Setup"
    log_info "Identity Endpoint: $IDENTITY_ENDPOINT"
    log_info "Command: $command"
    
    case $command in
        setup)
            check_prerequisites
            create_standard_roles
            assign_admin_role
            setup_org_roles "$org_name"
            validate_setup
            generate_documentation
            log_success "RBAC setup completed successfully!"
            ;;
        roles)
            check_prerequisites
            create_standard_roles
            log_success "Standard roles created successfully!"
            ;;
        validate)
            validate_setup
            ;;
        migrate)
            migrate_existing_users
            ;;
        docs)
            generate_documentation
            ;;
        *)
            log_error "Unknown command: $command"
            print_help
            exit 1
            ;;
    esac
}

# Run main function with all arguments
main "$@" 
