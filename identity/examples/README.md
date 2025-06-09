# PlantD Identity Service Examples

This directory contains examples demonstrating the PlantD Identity Service functionality.

## Examples

### Authentication Example
Location: `auth_example/`
- **File**: `auth_example.go`
- **Purpose**: Demonstrates the complete authentication flow including user registration, login, token management, and password operations.

**Run with:**
```bash
cd identity/examples/auth_example
PLANTD_IDENTITY_CONFIG=./identity.yaml go run auth_example.go
```

### RBAC Example  
Location: `rbac_example/`
- **File**: `rbac_simple_example.go`
- **Purpose**: Demonstrates the Role-Based Access Control (RBAC) system with permissions, roles, and organization-scoped access control.

**Run with:**
```bash
cd identity/examples/rbac_example
PLANTD_IDENTITY_CONFIG=./identity.yaml go run rbac_simple_example.go
```

## Configuration

Each example directory contains an `identity.yaml` configuration file with the necessary settings. The examples use an in-memory SQLite database for simplicity.

## Directory Structure

```
examples/
├── README.md                    # This file
├── auth_example/
│   ├── auth_example.go         # Authentication flow example
│   └── identity.yaml           # Configuration for auth example
└── rbac_example/
    ├── rbac_simple_example.go  # RBAC system example
    └── identity.yaml           # Configuration for RBAC example
```

This structure ensures that each example is in its own package, preventing Go module conflicts while keeping the examples organized and easy to run. 
