package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultPasswordConfig(t *testing.T) {
	config := DefaultPasswordConfig()

	assert.NotNil(t, config)
	assert.Equal(t, 8, config.MinLength)
	assert.Equal(t, 128, config.MaxLength)
	assert.True(t, config.RequireUppercase)
	assert.True(t, config.RequireLowercase)
	assert.True(t, config.RequireNumbers)
	assert.True(t, config.RequireSpecialChars)
	assert.Equal(t, 12, config.BcryptCost)
}

func TestNewPasswordValidator(t *testing.T) {
	t.Run("with custom config", func(t *testing.T) {
		config := &PasswordConfig{
			MinLength:        10,
			MaxLength:        50,
			RequireUppercase: false,
			BcryptCost:       10,
		}

		validator := NewPasswordValidator(config)
		assert.NotNil(t, validator)
		assert.Equal(t, config, validator.config)
	})

	t.Run("with nil config uses default", func(t *testing.T) {
		validator := NewPasswordValidator(nil)
		assert.NotNil(t, validator)
		assert.Equal(t, DefaultPasswordConfig(), validator.config)
	})
}

func TestPasswordValidator_Validate(t *testing.T) {
	validator := NewPasswordValidator(DefaultPasswordConfig())

	tests := []struct {
		name     string
		password string
		wantErr  bool
		errorMsg string
	}{
		{
			name:     "valid strong password",
			password: "StrongP@ssw0rd!",
			wantErr:  false,
		},
		{
			name:     "too short",
			password: "Short1!",
			wantErr:  true,
			errorMsg: "password must be at least 8 characters long",
		},
		{
			name:     "no uppercase",
			password: "lowercase123!",
			wantErr:  true,
			errorMsg: "password must contain at least one uppercase letter",
		},
		{
			name:     "no lowercase",
			password: "UPPERCASE123!",
			wantErr:  true,
			errorMsg: "password must contain at least one lowercase letter",
		},
		{
			name:     "no numbers",
			password: "NoNumbers!",
			wantErr:  true,
			errorMsg: "password must contain at least one number",
		},
		{
			name:     "no special characters",
			password: "NoSpecial123",
			wantErr:  true,
			errorMsg: "password must contain at least one special character",
		},
		{
			name:     "sequential characters",
			password: "Password123456!",
			wantErr:  true,
			errorMsg: "password must not contain sequential characters",
		},
		{
			name:     "repeated characters",
			password: "TestPasssss!9",
			wantErr:  true,
			errorMsg: "password must not contain more than 2 consecutive identical characters",
		},
		{
			name:     "common pattern - detects sequential first",
			password: "Password123!",
			wantErr:  true,
			errorMsg: "password must not contain sequential characters", // This actually detects "123" sequence
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.password)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPasswordValidator_HashPassword(t *testing.T) {
	validator := NewPasswordValidator(DefaultPasswordConfig())

	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "valid password",
			password: "ValidP@ssw0rd!",
			wantErr:  false,
		},
		{
			name:     "invalid password",
			password: "weak",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := validator.HashPassword(tt.password)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, hash)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, hash)
				assert.NotEqual(t, tt.password, hash)

				// Verify the hash can be validated
				assert.NoError(t, validator.VerifyPassword(hash, tt.password))
			}
		})
	}
}

func TestPasswordValidator_VerifyPassword(t *testing.T) {
	validator := NewPasswordValidator(DefaultPasswordConfig())
	// Use a password that doesn't trigger validation errors
	password := "TestP@ssw0rd!2024"
	hash, err := validator.HashPassword(password)
	require.NoError(t, err)

	tests := []struct {
		name         string
		hashedPwd    string
		plainPwd     string
		wantErr      bool
		errorMessage string
	}{
		{
			name:      "correct password",
			hashedPwd: hash,
			plainPwd:  password,
			wantErr:   false,
		},
		{
			name:         "incorrect password",
			hashedPwd:    hash,
			plainPwd:     "WrongPassword",
			wantErr:      true,
			errorMessage: "invalid password",
		},
		{
			name:      "empty password",
			hashedPwd: hash,
			plainPwd:  "",
			wantErr:   true,
		},
		{
			name:      "invalid hash",
			hashedPwd: "invalid_hash",
			plainPwd:  password,
			wantErr:   true,
		},
		{
			name:      "empty hash",
			hashedPwd: "",
			plainPwd:  password,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.VerifyPassword(tt.hashedPwd, tt.plainPwd)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorMessage != "" {
					assert.Contains(t, err.Error(), tt.errorMessage)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPasswordValidator_GetPasswordStrength(t *testing.T) {
	validator := NewPasswordValidator(DefaultPasswordConfig())

	tests := []struct {
		name        string
		password    string
		minExpected int
		maxExpected int
	}{
		{
			name:        "empty password",
			password:    "",
			minExpected: 0,
			maxExpected: 10,
		},
		{
			name:        "very weak password",
			password:    "password",
			minExpected: 0,
			maxExpected: 30,
		},
		{
			name:        "weak password",
			password:    "password123",
			minExpected: 10,
			maxExpected: 40,
		},
		{
			name:        "medium password",
			password:    "Password123",
			minExpected: 30,
			maxExpected: 60,
		},
		{
			name:        "strong password",
			password:    "StrongP@ssw0rd!",
			minExpected: 60,
			maxExpected: 100, // Allow up to 100 since it might get max score
		},
		{
			name:        "very strong password",
			password:    "VeryStr0ng!P@ssw0rd#2024",
			minExpected: 80,
			maxExpected: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := validator.GetPasswordStrength(tt.password)
			assert.GreaterOrEqual(t, score, tt.minExpected, "Score should be at least minimum expected")
			assert.LessOrEqual(t, score, tt.maxExpected, "Score should not exceed maximum expected")
			assert.GreaterOrEqual(t, score, 0, "Score should not be negative")
			assert.LessOrEqual(t, score, 100, "Score should not exceed 100")
		})
	}
}

func TestPasswordValidator_IsPasswordExpired(t *testing.T) {
	validator := NewPasswordValidator(DefaultPasswordConfig())

	tests := []struct {
		name        string
		passwordAge int
		maxAge      int
		expected    bool
	}{
		{
			name:        "not expired",
			passwordAge: 30,
			maxAge:      90,
			expected:    false,
		},
		{
			name:        "expired",
			passwordAge: 100,
			maxAge:      90,
			expected:    true,
		},
		{
			name:        "exactly at limit",
			passwordAge: 90,
			maxAge:      90,
			expected:    true,
		},
		{
			name:        "no expiration policy",
			passwordAge: 1000,
			maxAge:      0,
			expected:    false,
		},
		{
			name:        "negative max age",
			passwordAge: 50,
			maxAge:      -1,
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.IsPasswordExpired(tt.passwordAge, tt.maxAge)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPasswordValidator_CustomConfig(t *testing.T) {
	// Test with relaxed config
	relaxedConfig := &PasswordConfig{
		MinLength:           6,
		MaxLength:           50,
		RequireUppercase:    false,
		RequireLowercase:    true,
		RequireNumbers:      false,
		RequireSpecialChars: false,
		BcryptCost:          8,
	}

	validator := NewPasswordValidator(relaxedConfig)

	// This should pass with relaxed config
	err := validator.Validate("simplepass")
	assert.NoError(t, err)

	// Test hashing with lower cost
	hash, err := validator.HashPassword("simplepass")
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)

	// Verify it works
	err = validator.VerifyPassword(hash, "simplepass")
	assert.NoError(t, err)
}

func TestPasswordValidator_HashingConsistency(t *testing.T) {
	validator := NewPasswordValidator(DefaultPasswordConfig())
	// Use a password that doesn't trigger validation errors
	password := "TestP@ssw0rd!2024"

	// Hash the same password multiple times
	hashes := make([]string, 5)
	for i := 0; i < 5; i++ {
		hash, err := validator.HashPassword(password)
		require.NoError(t, err)
		hashes[i] = hash
	}

	// All hashes should be different (due to salt)
	for i := 0; i < len(hashes); i++ {
		for j := i + 1; j < len(hashes); j++ {
			assert.NotEqual(t, hashes[i], hashes[j], "Hashes should be different due to salt")
		}
	}

	// But all should verify correctly
	for _, hash := range hashes {
		assert.NoError(t, validator.VerifyPassword(hash, password))
	}
}

func TestPasswordValidator_WeakPatternDetection(t *testing.T) {
	validator := NewPasswordValidator(DefaultPasswordConfig())

	tests := []struct {
		name     string
		password string
		hasWeak  bool
	}{
		{
			name:     "no weak patterns",
			password: "StrongP@ssw0rd!",
			hasWeak:  false,
		},
		{
			name:     "sequential numbers",
			password: "Test123456!A",
			hasWeak:  true,
		},
		{
			name:     "sequential letters",
			password: "Testabcdef!1",
			hasWeak:  true,
		},
		{
			name:     "repeated characters",
			password: "Testaaa!123",
			hasWeak:  true,
		},
		{
			name:     "common pattern qwerty",
			password: "Testqwerty!1",
			hasWeak:  true,
		},
		{
			name:     "common pattern password",
			password: "TestPassword!1",
			hasWeak:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.password)

			if tt.hasWeak {
				assert.Error(t, err, "Should detect weak pattern")
			} else {
				assert.NoError(t, err, "Should not detect weak pattern")
			}
		})
	}
}
