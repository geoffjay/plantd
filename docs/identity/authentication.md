# Authentication System

## Overview

The PlantD Identity Service authentication system provides secure user authentication with JWT token management, password security, and comprehensive protection against common attacks.

## Core Components

### 1. Password Security

#### Password Validation
The system implements robust password validation with configurable policies:

```go
type PasswordConfig struct {
    MinLength           int  `json:"min_length"`           // Default: 8
    MaxLength           int  `json:"max_length"`           // Default: 128
    RequireUppercase    bool `json:"require_uppercase"`    // Default: true
    RequireLowercase    bool `json:"require_lowercase"`    // Default: true
    RequireNumbers      bool `json:"require_numbers"`      // Default: true
    RequireSpecialChars bool `json:"require_special_chars"` // Default: true
    BcryptCost          int  `json:"bcrypt_cost"`          // Default: 12
}
```

#### Password Strength Scoring
Passwords are scored on a 0-100 scale based on:
- **Length** (25 points for ≥8 chars, +15 for ≥12, +10 for ≥16)
- **Character Variety** (10 points each for uppercase, lowercase, numbers, special chars)
- **Variety Bonus** (10 points for ≥3 types, +10 for all 4 types)
- **Weak Pattern Penalty** (-20 points for common patterns)

#### Weak Pattern Detection
The system detects and rejects passwords containing:
- **Sequential characters**: `123`, `abc`, etc.
- **Repeated characters**: `aaa`, `111`, etc.
- **Common patterns**: `password`, `12345`, `qwerty`, etc.

#### Secure Hashing
- Uses **bcrypt** with configurable cost (default: 12)
- Validates passwords before hashing
- Constant-time comparison for verification

### 2. JWT Token Management

#### Token Types
The system supports three token types:

1. **Access Tokens**
   - Short-lived (default: 15 minutes)
   - Used for API authentication
   - Contains user claims and permissions

2. **Refresh Tokens**
   - Long-lived (default: 7 days)
   - Used to obtain new access tokens
   - Can be revoked/blacklisted

3. **Reset Tokens**
   - Very short-lived (default: 2 hours)
   - Used for password reset flows
   - Single-use only

#### Custom Claims Structure
```go
type CustomClaims struct {
    UserID        uint     `json:"user_id"`
    Email         string   `json:"email"`
    Username      string   `json:"username"`
    Organizations []uint   `json:"organizations"`
    Roles         []string `json:"roles"`
    Permissions   []string `json:"permissions"`
    TokenType     string   `json:"token_type"`
    EmailVerified bool     `json:"email_verified"`
    IsActive      bool     `json:"is_active"`
    LastLoginAt   int64    `json:"last_login_at,omitempty"`
    jwt.RegisteredClaims
}
```

#### Token Blacklisting
- In-memory blacklist for revoked tokens
- Automatic cleanup of expired tokens
- Support for immediate token invalidation

### 3. Authentication Service

#### Core Methods

**Login**
```go
func (a *AuthService) Login(ctx context.Context, req *AuthRequest) (*AuthResponse, error)
```
- Validates credentials (email/username + password)
- Implements rate limiting per IP/user
- Tracks failed attempts and account lockout
- Generates JWT token pair on success
- Logs security events

**Token Validation**
```go
func (a *AuthService) ValidateToken(ctx context.Context, tokenString string) (*CustomClaims, error)
```
- Parses and validates JWT tokens
- Checks token blacklist
- Verifies token signature and expiration
- Returns parsed claims

**Token Refresh**
```go
func (a *AuthService) RefreshToken(ctx context.Context, req *RefreshRequest) (*TokenPair, error)
```
- Validates refresh token
- Generates new access token
- Maintains refresh token (unless near expiry)
- Implements refresh token rotation

**Logout**
```go
func (a *AuthService) Logout(ctx context.Context, accessToken string) error
```
- Adds token to blacklist
- Prevents further use of the token
- Logs logout event

#### Account Lockout Protection
- Configurable failed attempt threshold (default: 5)
- Configurable lockout duration (default: 15 minutes)
- Per-user lockout tracking
- Automatic unlock after duration

### 4. Rate Limiting

#### Configuration
```go
type RateLimiterConfig struct {
    RequestsPerSecond int `json:"requests_per_second"` // Default: 10
    BurstSize         int `json:"burst_size"`          // Default: 5
}
```

#### Implementation
- Token bucket algorithm
- Per-IP rate limiting
- Separate limits for different operations
- Configurable burst allowance

### 5. User Registration

#### Registration Flow
1. **Input Validation**: Email format, username uniqueness, password strength
2. **User Creation**: Secure password hashing, database storage
3. **Email Verification**: Optional verification token generation
4. **Response**: Registration confirmation with verification status

#### Email Verification (Optional)
```go
type EmailVerificationRequest struct {
    Token     string `json:"token" validate:"required"`
    IPAddress string `json:"ip_address,omitempty"`
}
```

#### Profile Management
- **Profile Updates**: Name, username changes with validation
- **Password Changes**: Current password verification + new password validation
- **Account Deactivation**: Soft delete with data retention

### 6. Security Features

#### Audit Logging
All security events are logged with structured data:
```go
type SecurityEvent struct {
    UserID        *uint             `json:"user_id,omitempty"`
    Email         string            `json:"email,omitempty"`
    EventType     string            `json:"event_type"`
    Success       bool              `json:"success"`
    FailureReason string            `json:"failure_reason,omitempty"`
    IPAddress     string            `json:"ip_address,omitempty"`
    UserAgent     string            `json:"user_agent,omitempty"`
    Timestamp     time.Time         `json:"timestamp"`
    Metadata      map[string]string `json:"metadata,omitempty"`
}
```

#### Event Types
- `registration_success/failure`
- `login_success/failure`
- `login_invalid_password`
- `login_account_locked`
- `token_refresh_success/failure`
- `password_change_success/failure`
- `profile_update_success/failure`
- `password_reset_initiated`
- `logout`

## API Usage Examples

### User Registration
```go
registrationReq := &auth.RegistrationRequest{
    Email:     "user@example.com",
    Username:  "username",
    Password:  "MySecure123!Pass",
    FirstName: "John",
    LastName:  "Doe",
    IPAddress: "192.168.1.100",
    UserAgent: "MyApp/1.0",
}

response, err := registrationService.Register(ctx, registrationReq)
if err != nil {
    // Handle registration error
}
```

### User Authentication
```go
loginReq := &auth.AuthRequest{
    Identifier: "user@example.com", // Email or username
    Password:   "MySecure123!Pass",
    IPAddress:  "192.168.1.100",
    UserAgent:  "MyApp/1.0",
}

authResp, err := authService.Login(ctx, loginReq)
if err != nil {
    // Handle authentication error
}

// Use the access token for API calls
accessToken := authResp.TokenPair.AccessToken
```

### Token Validation
```go
claims, err := authService.ValidateToken(ctx, accessToken)
if err != nil {
    // Token is invalid
}

// Access user information from claims
userID := claims.UserID
email := claims.Email
permissions := claims.Permissions
```

### Token Refresh
```go
refreshReq := &auth.RefreshRequest{
    RefreshToken: authResp.TokenPair.RefreshToken,
    IPAddress:    "192.168.1.100",
}

newTokens, err := authService.RefreshToken(ctx, refreshReq)
if err != nil {
    // Handle refresh error - may need to re-authenticate
}
```

## Configuration

### Environment Variables
```bash
# JWT Configuration
PLANTD_IDENTITY_SECURITY_JWT_SECRET="your-secret-key"
PLANTD_IDENTITY_SECURITY_JWT_REFRESH_SECRET="your-refresh-secret"
PLANTD_IDENTITY_SECURITY_JWT_EXPIRATION=900  # 15 minutes
PLANTD_IDENTITY_SECURITY_REFRESH_EXPIRATION=604800  # 7 days

# Password Policy
PLANTD_IDENTITY_SECURITY_PASSWORD_MIN_LENGTH=8
PLANTD_IDENTITY_SECURITY_BCRYPT_COST=12
PLANTD_IDENTITY_SECURITY_REQUIRE_UPPERCASE=true

# Rate Limiting
PLANTD_IDENTITY_SECURITY_RATE_LIMIT_RPS=10
PLANTD_IDENTITY_SECURITY_RATE_LIMIT_BURST=5
PLANTD_IDENTITY_SECURITY_MAX_FAILED_ATTEMPTS=5
PLANTD_IDENTITY_SECURITY_LOCKOUT_DURATION_MINUTES=15

# Registration
PLANTD_IDENTITY_SECURITY_ALLOW_SELF_REGISTRATION=true
PLANTD_IDENTITY_SECURITY_REQUIRE_EMAIL_VERIFICATION=false
```

### YAML Configuration
```yaml
security:
  jwt_secret: "your-secret-key"
  jwt_refresh_secret: "your-refresh-secret"
  jwt_expiration: 900  # 15 minutes
  refresh_expiration: 604800  # 7 days
  jwt_issuer: "plantd-identity"
  
  bcrypt_cost: 12
  password_min_length: 8
  password_max_length: 128
  require_uppercase: true
  require_lowercase: true
  require_numbers: true
  require_special_chars: true
  
  rate_limit_rps: 10
  rate_limit_burst: 5
  max_failed_attempts: 5
  lockout_duration_minutes: 15
  
  allow_self_registration: true
  require_email_verification: false
  email_verification_expiry_hours: 24
  password_reset_expiry_hours: 2
```

## Security Best Practices

### Production Deployment
1. **Use strong JWT secrets** (minimum 32 characters, cryptographically random)
2. **Configure HTTPS only** for all token exchanges
3. **Set appropriate token expiry times** based on your security requirements
4. **Enable rate limiting** to prevent abuse
5. **Monitor failed login attempts** and implement alerting
6. **Regular secret rotation** for JWT signing keys
7. **Secure token storage** on the client side
8. **Implement proper CORS** policies

### Development
1. **Never commit secrets** to version control
2. **Use environment variables** for configuration
3. **Test with realistic data** but avoid real credentials
4. **Implement comprehensive logging** for security events
5. **Regular dependency updates** for security patches

## Troubleshooting

### Common Issues

**"Token validation failed"**
- Check token expiration
- Verify JWT secret configuration
- Ensure token hasn't been blacklisted

**"Account locked"**
- Check failed attempt count
- Wait for lockout duration to expire
- Review rate limiting configuration

**"Password validation failed"**
- Check password meets policy requirements
- Verify no common patterns are used
- Review password strength scoring

**"Rate limit exceeded"**
- Implement exponential backoff
- Check rate limiting configuration
- Verify IP-based limiting behavior 
