package auth

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

// PasswordConfig holds configuration for password policies and security.
type PasswordConfig struct {
	// MinLength is the minimum required password length.
	MinLength int `json:"min_length" yaml:"min_length"`
	// MaxLength is the maximum allowed password length.
	MaxLength int `json:"max_length" yaml:"max_length"`
	// RequireUppercase specifies if uppercase letters are required.
	RequireUppercase bool `json:"require_uppercase" yaml:"require_uppercase"`
	// RequireLowercase specifies if lowercase letters are required.
	RequireLowercase bool `json:"require_lowercase" yaml:"require_lowercase"`
	// RequireNumbers specifies if numbers are required.
	RequireNumbers bool `json:"require_numbers" yaml:"require_numbers"`
	// RequireSpecialChars specifies if special characters are required.
	RequireSpecialChars bool `json:"require_special_chars" yaml:"require_special_chars"`
	// BcryptCost is the cost factor for bcrypt hashing (4-31, recommended 12-15).
	BcryptCost int `json:"bcrypt_cost" yaml:"bcrypt_cost"`
}

// DefaultPasswordConfig returns a secure default password configuration.
func DefaultPasswordConfig() *PasswordConfig {
	return &PasswordConfig{
		MinLength:           8,
		MaxLength:           128,
		RequireUppercase:    true,
		RequireLowercase:    true,
		RequireNumbers:      true,
		RequireSpecialChars: true,
		BcryptCost:          12,
	}
}

// PasswordValidator handles password validation and hashing.
type PasswordValidator struct {
	config *PasswordConfig
}

// NewPasswordValidator creates a new password validator with the given configuration.
func NewPasswordValidator(config *PasswordConfig) *PasswordValidator {
	if config == nil {
		config = DefaultPasswordConfig()
	}
	return &PasswordValidator{
		config: config,
	}
}

// Validate checks if a password meets the configured policy requirements.
func (pv *PasswordValidator) Validate(password string) error {
	if len(password) < pv.config.MinLength {
		return fmt.Errorf("password must be at least %d characters long", pv.config.MinLength)
	}

	if len(password) > pv.config.MaxLength {
		return fmt.Errorf("password must not exceed %d characters", pv.config.MaxLength)
	}

	var hasUpper, hasLower, hasNumber, hasSpecial bool

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	if pv.config.RequireUppercase && !hasUpper {
		return errors.New("password must contain at least one uppercase letter")
	}

	if pv.config.RequireLowercase && !hasLower {
		return errors.New("password must contain at least one lowercase letter")
	}

	if pv.config.RequireNumbers && !hasNumber {
		return errors.New("password must contain at least one number")
	}

	if pv.config.RequireSpecialChars && !hasSpecial {
		return errors.New("password must contain at least one special character")
	}

	// Check for common weak patterns
	return pv.checkWeakPatterns(password)
}

// checkWeakPatterns validates against common weak password patterns.
func (pv *PasswordValidator) checkWeakPatterns(password string) error {
	// Check for sequential characters (123, abc, etc.)
	sequential := regexp.MustCompile(`(?i)(012|123|234|345|456|567|678|789|890|abc|bcd|cde|def|efg|fgh|ghi|hij|ijk|jkl|klm|lmn|mno|nop|opq|pqr|qrs|rst|stu|tuv|uvw|vwx|wxy|xyz)`) //nolint:revive
	if sequential.MatchString(password) {
		return errors.New("password must not contain sequential characters")
	}

	// Check for repeated characters (aaa, 111, etc.)
	// Note: Go's RE2 doesn't support backreferences, so we check for common repeated patterns
	repeatedPatterns := []string{
		"aaa", "bbb", "ccc", "ddd", "eee", "fff", "ggg", "hhh", "iii", "jjj", "kkk", "lll", "mmm",
		"nnn", "ooo", "ppp", "qqq", "rrr", "sss", "ttt", "uuu", "vvv", "www", "xxx", "yyy", "zzz",
		"000", "111", "222", "333", "444", "555", "666", "777", "888", "999",
	}
	passwordLowerForRepeat := strings.ToLower(password)
	for _, pattern := range repeatedPatterns {
		if strings.Contains(passwordLowerForRepeat, pattern) {
			return errors.New("password must not contain more than 2 consecutive identical characters")
		}
	}

	// Check for common patterns
	commonPatterns := []string{
		"password", "12345", "qwerty", "admin", "letmein", "welcome",
		"monkey", "dragon", "master", "123456789", "111111", "1234567890",
	}

	passwordLower := regexp.MustCompile(`\W`).ReplaceAllString(password, "")
	for _, pattern := range commonPatterns {
		matched, _ := regexp.MatchString("(?i)"+pattern, passwordLower)
		if matched {
			return fmt.Errorf("password must not contain common patterns like '%s'", pattern)
		}
	}

	return nil
}

// HashPassword creates a bcrypt hash of the password.
func (pv *PasswordValidator) HashPassword(password string) (string, error) {
	// Validate password before hashing
	if err := pv.Validate(password); err != nil {
		return "", fmt.Errorf("password validation failed: %w", err)
	}

	// Generate bcrypt hash
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), pv.config.BcryptCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	return string(hashedBytes), nil
}

// VerifyPassword checks if the provided password matches the hashed password.
func (pv *PasswordValidator) VerifyPassword(hashedPassword, password string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return errors.New("invalid password")
		}
		return fmt.Errorf("password verification failed: %w", err)
	}
	return nil
}

// GetPasswordStrength returns a score (0-100) indicating password strength.
func (pv *PasswordValidator) GetPasswordStrength(password string) int {
	score := 0
	length := len(password)

	// Length scoring
	if length >= 8 {
		score += 25
	}
	if length >= 12 {
		score += 15
	}
	if length >= 16 {
		score += 10
	}

	var hasUpper, hasLower, hasNumber, hasSpecial bool
	charVariety := 0

	for _, char := range password {
		switch {
		case unicode.IsUpper(char) && !hasUpper:
			hasUpper = true
			charVariety++
			score += 10
		case unicode.IsLower(char) && !hasLower:
			hasLower = true
			charVariety++
			score += 10
		case unicode.IsDigit(char) && !hasNumber:
			hasNumber = true
			charVariety++
			score += 10
		case (unicode.IsPunct(char) || unicode.IsSymbol(char)) && !hasSpecial:
			hasSpecial = true
			charVariety++
			score += 10
		}
	}

	// Bonus for character variety
	if charVariety >= 3 {
		score += 10
	}
	if charVariety >= 4 {
		score += 10
	}

	// Check for weak patterns and deduct points
	if pv.checkWeakPatterns(password) != nil {
		score -= 20
	}

	// Ensure score is between 0 and 100
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return score
}

// IsPasswordExpired checks if a password should be expired based on age.
func (pv *PasswordValidator) IsPasswordExpired(passwordAge int, maxAge int) bool {
	if maxAge <= 0 {
		return false // No expiration policy
	}
	return passwordAge >= maxAge
}
