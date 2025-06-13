package cmd

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestStateCommands(t *testing.T) {
	tests := []struct {
		name    string
		cmd     *cobra.Command
		use     string
		minArgs int
		maxArgs int
	}{
		{
			name:    "get command",
			cmd:     stateGetCmd,
			use:     "get",
			minArgs: 1,
			maxArgs: 1,
		},
		{
			name:    "set command",
			cmd:     stateSetCmd,
			use:     "set",
			minArgs: 2,
			maxArgs: 2,
		},
		{
			name:    "list command",
			cmd:     stateListCmd,
			use:     "list",
			minArgs: 0,
			maxArgs: 0,
		},
		{
			name:    "delete command",
			cmd:     stateDeleteCmd,
			use:     "delete",
			minArgs: 1,
			maxArgs: 1,
		},
		{
			name:    "create-scope command",
			cmd:     stateCreateScopeCmd,
			use:     "create-scope",
			minArgs: 0,
			maxArgs: 0,
		},
		{
			name:    "delete-scope command",
			cmd:     stateDeleteScopeCmd,
			use:     "delete-scope",
			minArgs: 0,
			maxArgs: 0,
		},
		{
			name:    "list-scopes command",
			cmd:     stateListScopesCmd,
			use:     "list-scopes",
			minArgs: 0,
			maxArgs: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.use, tt.cmd.Use)
			assert.NotEmpty(t, tt.cmd.Short)
			assert.NotEmpty(t, tt.cmd.Long)
			assert.NotNil(t, tt.cmd.Run)

			// Test argument validation
			if tt.minArgs == tt.maxArgs {
				if tt.minArgs == 0 {
					assert.NoError(t, tt.cmd.Args(tt.cmd, []string{}))
				} else {
					// Test with correct number of args
					args := make([]string, tt.minArgs)
					for i := range args {
						args[i] = "test"
					}
					assert.NoError(t, tt.cmd.Args(tt.cmd, args))

					// Test with wrong number of args
					if tt.minArgs > 0 {
						assert.Error(t, tt.cmd.Args(tt.cmd, []string{}))
					}
					assert.Error(t, tt.cmd.Args(tt.cmd, append(args, "extra")))
				}
			}
		})
	}
}

func TestStateCommandStructure(t *testing.T) {
	// Verify the state command has all expected subcommands
	expectedSubcommands := []string{
		"get",
		"set",
		"list",
		"delete",
		"create-scope",
		"delete-scope",
		"list-scopes",
	}

	subcommands := stateCmd.Commands()
	assert.Equal(t, len(expectedSubcommands), len(subcommands))

	for _, expected := range expectedSubcommands {
		found := false
		for _, subcmd := range subcommands {
			if subcmd.Use == expected {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected subcommand %s not found", expected)
	}
}

func TestHelperFunctions(t *testing.T) {
	t.Run("isAuthError", func(t *testing.T) {
		// Mock error strings that should be detected as auth errors
		authErrors := []string{
			"authentication failed",
			"invalid token",
			"token expired",
			"unauthorized",
			"401",
			"403",
		}

		for _, errMsg := range authErrors {
			err := &mockError{errMsg}
			assert.True(t, isAuthError(err), "Error '%s' should be detected as auth error", errMsg)
		}

		// Test non-auth errors
		nonAuthErrors := []string{
			"network error",
			"internal server error",
			"500",
			"bad request",
		}

		for _, errMsg := range nonAuthErrors {
			err := &mockError{errMsg}
			assert.False(t, isAuthError(err), "Error '%s' should not be detected as auth error", errMsg)
		}

		// Test nil error
		assert.False(t, isAuthError(nil))
	})

	t.Run("isPermissionError", func(t *testing.T) {
		permErrors := []string{
			"permission denied",
			"insufficient permissions",
			"forbidden",
			"403",
		}

		for _, errMsg := range permErrors {
			err := &mockError{errMsg}
			assert.True(t, isPermissionError(err), "Error '%s' should be detected as permission error", errMsg)
		}

		err := &mockError{"some other error"}
		assert.False(t, isPermissionError(err))
		assert.False(t, isPermissionError(nil))
	})

	t.Run("isNetworkError", func(t *testing.T) {
		networkErrors := []string{
			"connection refused",
			"network unreachable",
			"timeout",
			"no such host",
			"connection failed",
		}

		for _, errMsg := range networkErrors {
			err := &mockError{errMsg}
			assert.True(t, isNetworkError(err), "Error '%s' should be detected as network error", errMsg)
		}

		err := &mockError{"some other error"}
		assert.False(t, isNetworkError(err))
		assert.False(t, isNetworkError(nil))
	})

	t.Run("containsAny", func(t *testing.T) {
		assert.True(t, containsAny("this is a test string", []string{"test"}))
		assert.True(t, containsAny("authentication failed", []string{"auth", "failed"}))
		assert.False(t, containsAny("hello world", []string{"foo", "bar"}))
		assert.False(t, containsAny("", []string{"test"}))
		assert.False(t, containsAny("test", []string{}))
	})
}

// mockError is a helper for testing error functions
type mockError struct {
	message string
}

func (e *mockError) Error() string {
	return e.message
}
