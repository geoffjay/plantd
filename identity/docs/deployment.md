# Identity Service Deployment Guide

## Overview

This guide covers the deployment of the PlantD Identity Service using Docker and Docker Compose. The service provides authentication, authorization, and user management capabilities for the PlantD ecosystem.

## Prerequisites

### System Requirements

- **Docker**: Version 20.10+ with BuildKit support
- **Docker Compose**: Version 2.0+
- **Operating System**: Linux (Ubuntu 20.04+, CentOS 8+, RHEL 8+)
- **Memory**: Minimum 2GB RAM (4GB+ recommended for production)
- **Storage**: Minimum 10GB free space (50GB+ recommended for production)
- **Network**: Ports 8080 (HTTP), 5433 (PostgreSQL), 6380 (Redis)

### Required Tools

```bash
# Install Docker and Docker Compose
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
sudo usermod -aG docker $USER

# Install Docker Compose
sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose

# Verify installation
docker --version
docker-compose --version
```

## Quick Start

### 1. Clone and Setup

```bash
# Navigate to the identity service directory
cd identity/

# Create environment file from template
cp scripts/env.example .env

# Edit environment variables (IMPORTANT!)
nano .env
```

### 2. Configure Environment

**Critical:** Update these values in `.env`:

```bash
# Generate a strong JWT secret (256-bit minimum)
JWT_SECRET=$(openssl rand -base64 32)

# Set database passwords
POSTGRES_PASSWORD=$(openssl rand -base64 16)
REDIS_PASSWORD=$(openssl rand -base64 16)

# Configure email (if using email features)
SMTP_HOST=your-smtp-server.com
SMTP_USER=noreply@yourdomain.com
SMTP_PASS=your-smtp-password
EMAIL_FROM="PlantD Identity <noreply@yourdomain.com>"
```

### 3. Deploy

```bash
# Make deployment script executable
chmod +x scripts/deploy.sh

# Deploy the service
./scripts/deploy.sh deploy

# Check status
./scripts/deploy.sh status
```

## Environment Configuration

### Core Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `JWT_SECRET` | **Yes** | - | Secret key for JWT signing (256-bit minimum) |
| `POSTGRES_DB` | **Yes** | `plantd_identity` | PostgreSQL database name |
| `POSTGRES_USER` | **Yes** | `identity_user` | PostgreSQL username |
| `POSTGRES_PASSWORD` | **Yes** | - | PostgreSQL password |
| `BROKER_ENDPOINT` | **Yes** | `tcp://broker:9797` | PlantD broker endpoint |

### Security Configuration

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `BCRYPT_COST` | No | `12` | Bcrypt hashing cost (10-15) |
| `RATE_LIMIT_PER_IP` | No | `100` | Rate limit per IP address |
| `RATE_LIMIT_PER_USER` | No | `50` | Rate limit per user |
| `LOCKOUT_ATTEMPTS` | No | `5` | Failed login attempts before lockout |
| `LOCKOUT_DURATION` | No | `30m` | Account lockout duration |

### Email Configuration (Optional)

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `EMAIL_ENABLED` | No | `false` | Enable email features |
| `SMTP_HOST` | No | - | SMTP server hostname |
| `SMTP_PORT` | No | `587` | SMTP server port |
| `SMTP_USER` | No | - | SMTP username |
| `SMTP_PASS` | No | - | SMTP password |
| `EMAIL_FROM` | No | - | From email address |

## Deployment Modes

### Development Deployment

Use the standard `docker-compose.yml` for development:

```bash
# Start development environment
docker-compose up -d

# View logs
docker-compose logs -f identity

# Stop services
docker-compose down
```

### Production Deployment

Use the production configuration with enhanced security:

```bash
# Use production compose file
docker-compose -f docker/docker-compose.production.yml up -d

# Or use the deployment script
./scripts/deploy.sh deploy
```

### Production Environment Setup

```bash
# Create data directories
sudo mkdir -p /var/lib/plantd/{identity,postgres,redis}
sudo chown 1001:1001 /var/lib/plantd/identity
sudo chown 999:999 /var/lib/plantd/postgres
sudo chown 999:999 /var/lib/plantd/redis

# Create backup directories
sudo mkdir -p /var/backups/{postgres,redis}
sudo chown 999:999 /var/backups/{postgres,redis}

# Set up log rotation
sudo cp docker/logrotate.conf /etc/logrotate.d/plantd-identity
```

## SSL/TLS Configuration

### Generate SSL Certificates

```bash
# Create SSL directory
mkdir -p docker/nginx/ssl

# Generate self-signed certificate (development)
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout docker/nginx/ssl/identity.key \
  -out docker/nginx/ssl/identity.crt \
  -subj "/C=US/ST=State/L=City/O=Organization/CN=identity.local"

# Or use Let's Encrypt (production)
sudo certbot certonly --standalone -d identity.yourdomain.com
sudo cp /etc/letsencrypt/live/identity.yourdomain.com/* docker/nginx/ssl/
```

### Nginx Configuration

Create `docker/nginx/nginx.conf`:

```nginx
events {
    worker_connections 1024;
}

http {
    upstream identity {
        server identity:8080;
    }

    server {
        listen 80;
        server_name identity.yourdomain.com;
        return 301 https://$server_name$request_uri;
    }

    server {
        listen 443 ssl http2;
        server_name identity.yourdomain.com;

        ssl_certificate /etc/nginx/ssl/identity.crt;
        ssl_certificate_key /etc/nginx/ssl/identity.key;
        ssl_protocols TLSv1.2 TLSv1.3;
        ssl_ciphers HIGH:!aNULL:!MD5;

        location / {
            proxy_pass http://identity;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }

        location /health {
            proxy_pass http://identity/health;
            access_log off;
        }
    }
}
```

## Security Hardening

### Container Security

1. **Run as non-root user** ✅ (Already configured)
2. **Read-only filesystem** ✅ (Production config)
3. **No new privileges** ✅ (Security options set)
4. **Limited resources** (Configure as needed)

### Network Security

```bash
# Create isolated network
docker network create plantd-network \
  --driver bridge \
  --subnet=172.20.0.0/16 \
  --ip-range=172.20.240.0/20

# Configure firewall (UFW example)
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw allow 22/tcp
sudo ufw deny 8080/tcp  # Block direct access
sudo ufw deny 5433/tcp  # Block direct database access
sudo ufw enable
```

### Database Security

```bash
# Configure PostgreSQL authentication
echo "host all all 0.0.0.0/0 scram-sha-256" >> docker/postgres-init/pg_hba.conf

# Set strong passwords
POSTGRES_PASSWORD=$(openssl rand -base64 32)
REDIS_PASSWORD=$(openssl rand -base64 32)
```

## Monitoring and Logging

### Health Checks

The service includes built-in health checks:

```bash
# Check service health
curl -f http://localhost:8080/health

# Check detailed status
curl -s http://localhost:8080/health | jq .
```

### Log Management

```bash
# View real-time logs
docker-compose logs -f identity

# Export logs for analysis
docker-compose logs --since="24h" identity > identity-logs-$(date +%Y%m%d).log

# Configure log rotation
sudo logrotate -f /etc/logrotate.d/plantd-identity
```

### Metrics and Monitoring

The service exports metrics for monitoring:

- **Health endpoint**: `/health`
- **Metrics endpoint**: `/metrics` (if enabled)
- **Prometheus integration**: Available via environment variables

## Backup and Recovery

### Automated Backups

```bash
# Create full backup
./scripts/backup.sh backup-full

# Schedule automatic backups (crontab)
0 2 * * * /path/to/identity/scripts/backup.sh backup-full
0 3 * * 0 /path/to/identity/scripts/backup.sh clean 30
```

### Manual Backup

```bash
# Database backup
./scripts/backup.sh backup-db

# Configuration backup
./scripts/backup.sh backup-config

# Volume backup
./scripts/backup.sh backup-volumes
```

### Recovery

```bash
# List available backups
./scripts/backup.sh list

# Restore database
./scripts/backup.sh restore-db backups/identity_db_20240101_120000.sql.gz

# Restore volumes
./scripts/backup.sh restore-volumes backups/identity_volumes_20240101_120000.tar.gz
```

## Troubleshooting

### Common Issues

#### Service Won't Start

```bash
# Check container logs
docker-compose logs identity

# Check system resources
docker stats

# Verify environment variables
docker-compose config
```

#### Database Connection Issues

```bash
# Test database connectivity
docker exec plantd-identity-postgres pg_isready -U identity_user

# Check database logs
docker-compose logs postgres

# Verify network connectivity
docker network ls
docker network inspect plantd-network
```

#### Performance Issues

```bash
# Monitor resource usage
docker stats plantd-identity

# Check database performance
docker exec plantd-identity-postgres psql -U identity_user -d plantd_identity -c "
SELECT query, mean_exec_time, calls 
FROM pg_stat_statements 
ORDER BY mean_exec_time DESC 
LIMIT 10;"
```

### Log Analysis

```bash
# Search for errors
docker-compose logs identity | grep -i error

# Check authentication failures
docker-compose logs identity | grep "authentication failed"

# Monitor rate limiting
docker-compose logs identity | grep "rate limit"
```

### Debug Mode

Enable debug logging:

```bash
# Set debug environment
echo "LOG_LEVEL=debug" >> .env

# Restart service
docker-compose restart identity

# View debug logs
docker-compose logs -f identity
```

## Maintenance

### Regular Maintenance Tasks

1. **Weekly**: Check service health and logs
2. **Weekly**: Review security events and rate limiting
3. **Monthly**: Update Docker images and dependencies
4. **Monthly**: Clean old backups and logs
5. **Quarterly**: Review and rotate secrets

### Updates and Upgrades

```bash
# Pull latest images
docker-compose pull

# Restart with new images
docker-compose up -d --force-recreate

# Verify deployment
./scripts/deploy.sh health
```

### Security Updates

```bash
# Update base images
docker pull alpine:latest
docker pull postgres:15-alpine
docker pull redis:7-alpine

# Rebuild identity image
docker build -t geoffjay/plantd-identity:latest -f Dockerfile .

# Rolling update
docker-compose up -d --no-deps identity
```

## Integration with Other Services

### Broker Integration

The identity service integrates with the PlantD broker using MDP protocol:

```yaml
environment:
  PLANTD_BROKER_ENDPOINT: tcp://broker:9797
```

### Client Integration

Other PlantD services can use the identity client library:

```go
import "github.com/plantd/identity/pkg/client"

client := client.New("tcp://broker:9797")
token, err := client.Login("user@example.com", "password")
```

### Load Balancing

For high availability, deploy multiple instances:

```yaml
# docker-compose.ha.yml
services:
  identity-1:
    <<: *identity-service
    container_name: plantd-identity-1
  
  identity-2:
    <<: *identity-service
    container_name: plantd-identity-2
  
  identity-lb:
    image: nginx:alpine
    ports:
      - "8080:80"
    volumes:
      - ./nginx-lb.conf:/etc/nginx/nginx.conf
```

## Performance Tuning

### Database Optimization

```sql
-- PostgreSQL optimization
ALTER SYSTEM SET shared_buffers = '256MB';
ALTER SYSTEM SET effective_cache_size = '1GB';
ALTER SYSTEM SET maintenance_work_mem = '64MB';
SELECT pg_reload_conf();
```

### Connection Pooling

Configure connection limits:

```yaml
environment:
  PLANTD_IDENTITY_DATABASE_MAX_OPEN_CONNS: 25
  PLANTD_IDENTITY_DATABASE_MAX_IDLE_CONNS: 10
  PLANTD_IDENTITY_DATABASE_CONN_MAX_LIFETIME: 1h
```

### Redis Optimization

```bash
# Redis memory optimization
redis-cli CONFIG SET maxmemory 512mb
redis-cli CONFIG SET maxmemory-policy allkeys-lru
```

## Support and Documentation

- **Service Documentation**: See `docs/` directory
- **API Reference**: See `docs/api-reference.md`
- **Architecture Guide**: See `docs/architecture.md`
- **Configuration Reference**: See `docs/configuration.md`

For issues and support, check the service logs and health endpoints first, then consult the troubleshooting section above. 
