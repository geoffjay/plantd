package auth

// State service specific permissions as defined in the execution plan.
const (
	// Scope Management Permissions (Global)
	StateScopeCreate = "state:scope:create" // Create new service scopes
	StateScopeDelete = "state:scope:delete" // Delete entire service scopes
	StateScopeList   = "state:scope:list"   // List available scopes

	// Data Access Permissions (per-scope)
	StateDataRead   = "state:data:read"   // Read key-value pairs
	StateDataWrite  = "state:data:write"  // Set/update key-value pairs
	StateDataDelete = "state:data:delete" // Delete key-value pairs

	// Administrative Permissions
	StateAdminFull  = "state:admin:full"  // Full administrative access
	StateHealthRead = "state:health:read" // Health check endpoint access
)

// Permission represents a user permission with scope.
type Permission struct {
	Name  string `json:"name"`
	Scope string `json:"scope"`
}

// StandardRoles defines common role templates for the state service.
var StandardRoles = []Role{
	{
		Name:        "state-developer",
		Description: "Developer access to state service",
		Permissions: []string{
			StateDataRead,
			StateDataWrite,
			StateHealthRead,
		},
		Scope: "organization",
	},
	{
		Name:        "state-admin",
		Description: "Administrative access to state service",
		Permissions: []string{
			StateScopeCreate,
			StateScopeDelete,
			StateScopeList,
			StateDataRead,
			StateDataWrite,
			StateDataDelete,
			StateHealthRead,
		},
		Scope: "organization",
	},
	{
		Name:        "state-system-admin",
		Description: "Full system access to state service",
		Permissions: []string{
			"state:*", // Wildcard for all state permissions
		},
		Scope: "global",
	},
	{
		Name:        "state-readonly",
		Description: "Read-only access to state service",
		Permissions: []string{
			StateDataRead,
			StateHealthRead,
		},
		Scope: "organization",
	},
}

// Role represents a role with permissions.
type Role struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
	Scope       string   `json:"scope"`
}
