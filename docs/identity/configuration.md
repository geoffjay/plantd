# Configuration Guide

## Overview

The PlantD Identity Service uses a flexible configuration system that supports multiple sources and environments. This guide covers all configuration options and deployment scenarios.

## Configuration Sources

The service loads configuration from multiple sources in order of precedence:

1. **Command line flags** (highest priority)
2. **Environment variables**
3. **Configuration files** (YAML/JSON)
4. **Default values** (lowest priority)

## Configuration File

### Location

The service looks for configuration files in the following locations:

1. Path specified by `PLANTD_IDENTITY_CONFIG` environment variable
2. Current working directory: `./identity.yaml`
3. User config directory: `$HOME/.config/plantd/identity.yaml`
4. System config directory: `/etc/plantd/identity.yaml`

### Complete Configuration Example

```yaml
# Environment (development, staging, production)
env: production

# Database Configuration
database:
  driver: postgres  # sqlite, postgres, mysql
  dsn: "postgres://user:password@localhost:5432/plantd_identity?sslmode=require"
  max_open_conns: 25
  max_idle_conns: 5
  conn_max_lifetime: 300  # seconds

# Server Configuration
server:
  host: "0.0.0.0"
  port: 8080
  read_timeout: 30     # seconds
  write_timeout: 30    # seconds
  idle_timeout: 120    # seconds
  shutdown_timeout: 30 # seconds

# Security Configuration
security:
  # JWT Token Configuration
  jwt_secret: "your-very-secure-jwt-secret-key-change-in-production"
  jwt_refresh_secret: "your-very-secure-refresh-secret-key-change-in-production"
  jwt_expiration: 900        # 15 minutes (seconds)
  refresh_expiration: 604800 # 7 days (seconds)
  jwt_issuer: "plantd-identity"
  jwt_audience: "plantd-services"
  
  # Password Policy
  bcrypt_cost: 12
  password_min_length: 8
  password_max_length: 128
  require_uppercase: true
  require_lowercase: true
  require_numbers: true
  require_special_chars: true
  
  # Rate Limiting
  rate_limit_rps: 10           # requests per second
  rate_limit_burst: 5          # burst allowance
  max_failed_attempts: 5       # before account lockout
  lockout_duration_minutes: 15 # account lockout duration
  
  # Registration & Verification
  allow_self_registration: true
  require_email_verification: true
  email_verification_expiry_hours: 24
  password_reset_expiry_hours: 2
  
  # RBAC Configuration
  enable_rbac: true
  cache_permissions: true
  permission_cache_ttl: 300    # 5 minutes (seconds)
  enable_audit_logging: true
  
  # CORS Configuration
  cors_allowed_origins:
    - "https://app.plantd.com"
    - "https://admin.plantd.com"
  cors_allowed_methods:
    - GET
    - POST
    - PUT
    - DELETE
    - OPTIONS
  cors_allowed_headers:
    - "Content-Type"
    - "Authorization"
    - "X-Requested-With"

# Logging Configuration
log:
  level: info        # debug, info, warn, error
  formatter: json    # text, json
  output: stdout     # stdout, stderr, file path
  file_path: "/var/log/plantd/identity.log"
  max_size: 100      # MB
  max_backups: 5
  max_age: 30        # days

# Service Configuration
service:
  id: "org.plantd.Identity"
  name: "PlantD Identity Service"
  version: "1.0.0"
  description: "Authentication and authorization service for PlantD"

# Email Configuration (for verification/reset emails)
email:
  smtp_host: "smtp.gmail.com"
  smtp_port: 587
  smtp_username: "noreply@plantd.com"
  smtp_password: "your-email-password"
  from_address: "noreply@plantd.com"
  from_name: "PlantD Identity Service"

# Monitoring & Metrics
monitoring:
  enable_metrics: true
  metrics_port: 9090
  metrics_path: "/metrics"
  enable_health_checks: true
  health_check_path: "/health"

# Cache Configuration (for permissions, tokens, etc.)
cache:
  type: "memory"     # memory, redis
  redis_url: "redis://localhost:6379/0"
  default_ttl: 300   # 5 minutes (seconds)
```

## Environment Variables

All configuration options can be set via environment variables using the prefix `PLANTD_IDENTITY_` and uppercase with underscores:

### Core Settings

```bash
# Environment
export PLANTD_IDENTITY_ENV=production

# Database
export PLANTD_IDENTITY_DATABASE_DRIVER=postgres
export PLANTD_IDENTITY_DATABASE_DSN="postgres://user:pass@localhost:5432/identity"
export PLANTD_IDENTITY_DATABASE_MAX_OPEN_CONNS=25
export PLANTD_IDENTITY_DATABASE_MAX_IDLE_CONNS=5

# Server
export PLANTD_IDENTITY_SERVER_HOST=0.0.0.0
export PLANTD_IDENTITY_SERVER_PORT=8080
export PLANTD_IDENTITY_SERVER_READ_TIMEOUT=30
export PLANTD_IDENTITY_SERVER_WRITE_TIMEOUT=30

# Security - JWT
export PLANTD_IDENTITY_SECURITY_JWT_SECRET="your-secret-key"
export PLANTD_IDENTITY_SECURITY_JWT_REFRESH_SECRET="your-refresh-secret"
export PLANTD_IDENTITY_SECURITY_JWT_EXPIRATION=900
export PLANTD_IDENTITY_SECURITY_REFRESH_EXPIRATION=604800
export PLANTD_IDENTITY_SECURITY_JWT_ISSUER="plantd-identity"

# Security - Password Policy
export PLANTD_IDENTITY_SECURITY_BCRYPT_COST=12
export PLANTD_IDENTITY_SECURITY_PASSWORD_MIN_LENGTH=8
export PLANTD_IDENTITY_SECURITY_PASSWORD_MAX_LENGTH=128
export PLANTD_IDENTITY_SECURITY_REQUIRE_UPPERCASE=true
export PLANTD_IDENTITY_SECURITY_REQUIRE_LOWERCASE=true
export PLANTD_IDENTITY_SECURITY_REQUIRE_NUMBERS=true
export PLANTD_IDENTITY_SECURITY_REQUIRE_SPECIAL_CHARS=true

# Security - Rate Limiting
export PLANTD_IDENTITY_SECURITY_RATE_LIMIT_RPS=10
export PLANTD_IDENTITY_SECURITY_RATE_LIMIT_BURST=5
export PLANTD_IDENTITY_SECURITY_MAX_FAILED_ATTEMPTS=5
export PLANTD_IDENTITY_SECURITY_LOCKOUT_DURATION_MINUTES=15

# Security - Registration
export PLANTD_IDENTITY_SECURITY_ALLOW_SELF_REGISTRATION=true
export PLANTD_IDENTITY_SECURITY_REQUIRE_EMAIL_VERIFICATION=true
export PLANTD_IDENTITY_SECURITY_EMAIL_VERIFICATION_EXPIRY_HOURS=24
export PLANTD_IDENTITY_SECURITY_PASSWORD_RESET_EXPIRY_HOURS=2

# Logging
export PLANTD_IDENTITY_LOG_LEVEL=info
export PLANTD_IDENTITY_LOG_FORMATTER=json
export PLANTD_IDENTITY_LOG_OUTPUT=stdout

# Email
export PLANTD_IDENTITY_EMAIL_SMTP_HOST="smtp.gmail.com"
export PLANTD_IDENTITY_EMAIL_SMTP_PORT=587
export PLANTD_IDENTITY_EMAIL_SMTP_USERNAME="noreply@plantd.com"
export PLANTD_IDENTITY_EMAIL_SMTP_PASSWORD="your-email-password"
export PLANTD_IDENTITY_EMAIL_FROM_ADDRESS="noreply@plantd.com"
```

## Environment-Specific Configurations

### Development

```yaml
env: development

database:
  driver: sqlite
  dsn: "./identity_dev.db"

security:
  jwt_secret: "dev-secret-change-in-production"
  jwt_refresh_secret: "dev-refresh-secret-change-in-production"
  require_email_verification: false  # Easier for testing
  rate_limit_rps: 100               # Higher limits for development
  
log:
  level: debug
  formatter: text
  output: stdout

monitoring:
  enable_metrics: false
```

### Staging

```yaml
env: staging

database:
  driver: postgres
  dsn: "postgres://staging_user:staging_pass@staging-db:5432/identity_staging"

security:
  jwt_secret: "staging-secret-key"
  jwt_refresh_secret: "staging-refresh-secret"
  require_email_verification: true
  rate_limit_rps: 50
  
log:
  level: info
  formatter: json
  output: "/var/log/plantd/identity-staging.log"

monitoring:
  enable_metrics: true
```

### Production

```yaml
env: production

database:
  driver: postgres
  dsn: "postgres://prod_user:secure_pass@prod-db:5432/identity_prod?sslmode=require"
  max_open_conns: 25
  max_idle_conns: 5

security:
  jwt_secret: "${JWT_SECRET}"        # From environment/secrets
  jwt_refresh_secret: "${REFRESH_SECRET}"
  require_email_verification: true
  rate_limit_rps: 20
  enable_audit_logging: true
  
log:
  level: warn
  formatter: json
  output: "/var/log/plantd/identity.log"
  max_size: 100
  max_backups: 10

monitoring:
  enable_metrics: true
  enable_health_checks: true

email:
  smtp_host: "smtp.sendgrid.net"
  smtp_port: 587
  smtp_username: "apikey"
  smtp_password: "${SENDGRID_API_KEY}"
```

## Database Configuration

### SQLite (Development)

```yaml
database:
  driver: sqlite
  dsn: "./identity.db"
  # SQLite specific options
  pragma:
    foreign_keys: "ON"
    journal_mode: "WAL"
    synchronous: "NORMAL"
```

### PostgreSQL (Production)

```yaml
database:
  driver: postgres
  dsn: "postgres://username:password@hostname:5432/database?sslmode=require"
  max_open_conns: 25
  max_idle_conns: 5
  conn_max_lifetime: 300
  
  # PostgreSQL specific options
  search_path: "identity,public"
  application_name: "plantd-identity"
```

### MySQL

```yaml
database:
  driver: mysql
  dsn: "username:password@tcp(hostname:3306)/database?charset=utf8mb4&parseTime=True&loc=Local"
  max_open_conns: 25
  max_idle_conns: 5
  conn_max_lifetime: 300
```

## Security Configuration Details

### JWT Configuration

```yaml
security:
  # JWT signing algorithm (HS256, HS384, HS512, RS256, RS384, RS512)
  jwt_algorithm: "HS256"
  
  # JWT secrets (use strong, random keys in production)
  jwt_secret: "minimum-32-character-secret-key"
  jwt_refresh_secret: "different-32-character-refresh-key"
  
  # Token expiration times
  jwt_expiration: 900        # 15 minutes for access tokens
  refresh_expiration: 604800 # 7 days for refresh tokens
  reset_expiration: 7200     # 2 hours for password reset tokens
  
  # JWT claims
  jwt_issuer: "plantd-identity"
  jwt_audience: "plantd-services"
  
  # Token blacklist cleanup interval
  blacklist_cleanup_interval: 3600  # 1 hour
```

### Password Policy

```yaml
security:
  password_min_length: 8
  password_max_length: 128
  require_uppercase: true      # At least one A-Z
  require_lowercase: true      # At least one a-z  
  require_numbers: true        # At least one 0-9
  require_special_chars: true  # At least one special character
  
  # Password strength scoring
  min_password_score: 50       # 0-100 scale
  
  # Password history
  password_history_count: 5    # Prevent reusing last 5 passwords
  
  # Bcrypt cost (higher = more secure but slower)
  bcrypt_cost: 12              # Recommended: 12-14
```

### Rate Limiting

```yaml
security:
  # Global rate limiting
  rate_limit_rps: 10           # Requests per second per IP
  rate_limit_burst: 5          # Burst allowance
  
  # Authentication specific limits
  login_rate_limit_rps: 5      # Login attempts per second
  login_rate_limit_burst: 3    # Login burst allowance
  
  # Account lockout
  max_failed_attempts: 5       # Failed attempts before lockout
  lockout_duration_minutes: 15 # Lockout duration
  lockout_reset_hours: 24      # Auto-reset lockout after 24h
```

## Command Line Usage

### Basic Startup

```bash
# Start with default configuration
./identity-service

# Start with specific config file
./identity-service --config /path/to/config.yaml

# Start with environment override
./identity-service --env production

# Start with port override
./identity-service --port 8080
```

### Command Line Flags

```bash
Usage: identity-service [OPTIONS]

Options:
  --config PATH           Configuration file path
  --env ENV              Environment (development, staging, production)
  --port PORT            Server port (default: 8080)
  --host HOST            Server host (default: 0.0.0.0)
  --log-level LEVEL      Log level (debug, info, warn, error)
  --database-dsn DSN     Database connection string
  --jwt-secret SECRET    JWT signing secret
  --help                 Show this help message
  --version              Show version information
```

## Docker Configuration

### Docker Compose

```yaml
version: '3.8'

services:
  identity:
    image: plantd/identity:latest
    ports:
      - "8080:8080"
      - "9090:9090"  # metrics
    environment:
      PLANTD_IDENTITY_ENV: production
      PLANTD_IDENTITY_DATABASE_DSN: "postgres://postgres:password@db:5432/identity"
      PLANTD_IDENTITY_SECURITY_JWT_SECRET: "${JWT_SECRET}"
      PLANTD_IDENTITY_SECURITY_JWT_REFRESH_SECRET: "${REFRESH_SECRET}"
      PLANTD_IDENTITY_LOG_LEVEL: info
    depends_on:
      - db
    volumes:
      - ./config/identity.yaml:/etc/plantd/identity.yaml:ro
      - identity_logs:/var/log/plantd
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3

  db:
    image: postgres:15
    environment:
      POSTGRES_DB: identity
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: password
    volumes:
      - postgres_data:/var/lib/postgresql/data
    restart: unless-stopped

volumes:
  postgres_data:
  identity_logs:
```

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: identity-service
spec:
  replicas: 3
  selector:
    matchLabels:
      app: identity-service
  template:
    metadata:
      labels:
        app: identity-service
    spec:
      containers:
      - name: identity
        image: plantd/identity:latest
        ports:
        - containerPort: 8080
        - containerPort: 9090
        env:
        - name: PLANTD_IDENTITY_ENV
          value: production
        - name: PLANTD_IDENTITY_DATABASE_DSN
          valueFrom:
            secretKeyRef:
              name: identity-secrets
              key: database-dsn
        - name: PLANTD_IDENTITY_SECURITY_JWT_SECRET
          valueFrom:
            secretKeyRef:
              name: identity-secrets
              key: jwt-secret
        - name: PLANTD_IDENTITY_SECURITY_JWT_REFRESH_SECRET
          valueFrom:
            secretKeyRef:
              name: identity-secrets
              key: refresh-secret
        volumeMounts:
        - name: config
          mountPath: /etc/plantd
          readOnly: true
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
      volumes:
      - name: config
        configMap:
          name: identity-config
```

## Validation and Testing

### Configuration Validation

The service validates configuration on startup and will fail if:

- Required fields are missing
- Invalid values are provided
- Database connection fails
- JWT secrets are too weak

### Testing Configuration

```bash
# Validate configuration without starting service
./identity-service --config config.yaml --validate

# Test database connection
./identity-service --config config.yaml --test-db

# Generate sample configuration
./identity-service --generate-config > identity.yaml
```

## Security Best Practices

### Production Secrets

1. **Never commit secrets to version control**
2. **Use strong, random keys** (minimum 32 characters)
3. **Rotate secrets regularly**
4. **Use environment variables or secret managers**
5. **Different secrets for different environments**

### Database Security

1. **Use SSL/TLS connections** in production
2. **Limit database user permissions**
3. **Regular backups and testing**
4. **Monitor database access logs**

### Network Security

1. **Use HTTPS only** in production
2. **Configure proper CORS policies**
3. **Implement rate limiting**
4. **Use reverse proxy** (nginx, traefik)
5. **Network segmentation** for database

## Monitoring and Observability

### Metrics Configuration

```yaml
monitoring:
  enable_metrics: true
  metrics_port: 9090
  metrics_path: "/metrics"
  
  # Custom metrics
  collect_detailed_metrics: true
  metrics_namespace: "plantd_identity"
  
  # Health checks
  enable_health_checks: true
  health_check_path: "/health"
  health_check_timeout: 5
```

### Log Configuration

```yaml
log:
  level: info
  formatter: json
  output: stdout
  
  # Structured logging fields
  include_caller: true
  include_timestamp: true
  timestamp_format: "2006-01-02T15:04:05.000Z07:00"
  
  # Log rotation (for file output)
  file_path: "/var/log/plantd/identity.log"
  max_size: 100      # MB
  max_backups: 5
  max_age: 30        # days
  compress: true
```

## Troubleshooting

### Common Configuration Issues

1. **"Configuration file not found"**
   - Check file path and permissions
   - Verify PLANTD_IDENTITY_CONFIG environment variable

2. **"Database connection failed"**
   - Verify DSN format and credentials
   - Check network connectivity
   - Ensure database exists

3. **"JWT secret too weak"**
   - Use minimum 32-character secrets
   - Use cryptographically random keys

4. **"Permission denied"**
   - Check file permissions for config files
   - Verify user has database access

### Configuration Debugging

```bash
# Show effective configuration
./identity-service --config config.yaml --show-config

# Validate configuration
./identity-service --config config.yaml --validate

# Test with debug logging
PLANTD_IDENTITY_LOG_LEVEL=debug ./identity-service
```