# PlantD Production Deployment Guide

**Version**: 1.0.0
**Last Updated**: November 2024
**Prerequisites**: Docker, Docker Compose, 8GB RAM, 50GB Storage

## Overview

This guide provides step-by-step instructions for deploying PlantD to production with high availability, security, and monitoring. The deployment uses Docker Compose with Traefik as the gateway, providing SSL termination, authentication, and load balancing.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Infrastructure Requirements](#infrastructure-requirements)
3. [Pre-deployment Setup](#pre-deployment-setup)
4. [Production Deployment](#production-deployment)
5. [SSL Certificate Configuration](#ssl-certificate-configuration)
6. [Monitoring Setup](#monitoring-setup)
7. [Security Configuration](#security-configuration)
8. [Backup and Recovery](#backup-and-recovery)
9. [Post-deployment Validation](#post-deployment-validation)
10. [Maintenance](#maintenance)
11. [Troubleshooting](#troubleshooting)

## Prerequisites

### System Requirements

- **Operating System**: Ubuntu 20.04+ / CentOS 8+ / RHEL 8+
- **CPU**: 4+ cores (8+ recommended for production)
- **Memory**: 8GB RAM minimum (16GB+ recommended)
- **Storage**: 50GB available disk space (SSD recommended)
- **Network**: Static IP address with ports 80, 443, 8080, 8443 available

### Software Dependencies

```bash
# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
sudo usermod -aG docker $USER

# Install Docker Compose
sudo curl -L "https://github.com/docker/compose/releases/download/v2.20.0/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose

# Verify installations
docker --version
docker-compose --version
```

### Domain Requirements

For production deployment, you'll need:

- **Primary API Domain**: `api.plantd.com` (or your domain)
- **Authentication Domain**: `auth.plantd.com`
- **Monitoring Domain**: `monitoring.plantd.com`
- **Dashboard Domain**: `dashboard.plantd.com`
- **Status Domain**: `status.plantd.com`

## Infrastructure Requirements

### Recommended Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Load Balancer │    │   DNS Provider  │    │  SSL Provider   │
│    (Optional)   │    │   (Cloudflare)  │    │ (Let's Encrypt) │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
┌─────────────────────────────────┼─────────────────────────────────┐
│                         Production Server                        │
│  ┌─────────────────┐   ┌─────────────────┐   ┌─────────────────┐│
│  │     Traefik     │   │   Monitoring    │   │    Services     ││
│  │   (Gateway)     │   │ (Prometheus,    │   │  (State, ID,    ││
│  │                 │   │  Grafana)       │   │   Database)     ││
│  └─────────────────┘   └─────────────────┘   └─────────────────┘│
└─────────────────────────────────────────────────────────────────┘
```

### Port Configuration

| Service | Port | Protocol | Purpose |
|---------|------|----------|---------|
| Traefik Gateway | 80 | HTTP | HTTP redirect to HTTPS |
| Traefik Gateway | 443 | HTTPS | Primary HTTPS traffic |
| gRPC Gateway | 8080 | HTTP/2 | gRPC over HTTP |
| gRPC Secure | 8443 | HTTPS | gRPC over HTTPS |
| Prometheus | 9090 | HTTP | Metrics (internal) |
| Grafana | 3000 | HTTP | Dashboard (internal) |

## Pre-deployment Setup

### 1. Prepare the Server

```bash
# Update system packages
sudo apt update && sudo apt upgrade -y

# Install required tools
sudo apt install -y curl wget git jq htop

# Create deployment user
sudo useradd -m -s /bin/bash plantd
sudo usermod -aG docker plantd
sudo usermod -aG sudo plantd

# Switch to deployment user
sudo su - plantd
```

### 2. Clone and Prepare the Repository

```bash
# Clone the repository
git clone https://github.com/plantd/plantd.git
cd plantd

# Checkout the latest stable release
git checkout v1.0.0  # Replace with actual release

# Verify repository structure
ls -la
```

### 3. Environment Configuration

```bash
# Create production environment file
cp .env.example .env.production

# Edit production configuration
nano .env.production
```

**Required Environment Variables**:

```bash
# Database Configuration
POSTGRES_PASSWORD=your_secure_database_password_here
DATABASE_URL=postgres://plantd:your_secure_database_password_here@postgres-primary:5432/plantd_identity?sslmode=disable

# JWT Configuration (generate with: openssl rand -base64 32)
JWT_SECRET=your_jwt_secret_key_here

# Grafana Configuration
GRAFANA_PASSWORD=your_grafana_admin_password_here

# SSL/TLS Configuration
LETSENCRYPT_EMAIL=admin@your-domain.com

# Domain Configuration
PRIMARY_DOMAIN=plantd.com
API_DOMAIN=api.plantd.com
AUTH_DOMAIN=auth.plantd.com
MONITORING_DOMAIN=monitoring.plantd.com
DASHBOARD_DOMAIN=dashboard.plantd.com

# Performance Configuration
MAX_CONNECTIONS=200
WORKER_PROCESSES=4

# Security Configuration
CORS_ALLOWED_ORIGINS=https://plantd.com,https://*.plantd.com

# Monitoring Configuration
METRICS_ENABLED=true
TRACING_ENABLED=true

# Deployment Configuration
DEPLOYMENT_ENV=production
BLUE_GREEN_ENABLED=true
HEALTH_CHECK_TIMEOUT=60
ROLLBACK_ENABLED=true
```

### 4. DNS Configuration

Configure your DNS provider to point domains to your server:

```bash
# A Records
api.plantd.com      -> YOUR_SERVER_IP
auth.plantd.com     -> YOUR_SERVER_IP
monitoring.plantd.com -> YOUR_SERVER_IP
dashboard.plantd.com -> YOUR_SERVER_IP
status.plantd.com   -> YOUR_SERVER_IP
```

### 5. Firewall Configuration

```bash
# Configure UFW firewall
sudo ufw --force reset
sudo ufw default deny incoming
sudo ufw default allow outgoing

# Allow SSH
sudo ufw allow ssh

# Allow HTTP and HTTPS
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp

# Allow gRPC ports
sudo ufw allow 8080/tcp
sudo ufw allow 8443/tcp

# Enable firewall
sudo ufw --force enable
sudo ufw status
```

## Production Deployment

### 1. Build Production Images

```bash
# Build all production images
make build-production-images

# Verify images are built
docker images | grep plantd
```

### 2. Deploy with Blue/Green Strategy

```bash
# Deploy to production
make deploy-phase7

# Monitor deployment logs
make deploy-production-logs
```

### 3. Verify Service Health

```bash
# Check deployment status
make deploy-production-status

# Check individual service health
make deploy-production-health

# View service logs
make deploy-production-logs SERVICE=traefik
make deploy-production-logs SERVICE=state-primary
make deploy-production-logs SERVICE=identity-primary
```

### 4. Initial Service Setup

```bash
# Wait for services to be fully ready
sleep 60

# Create initial admin user (replace with your details)
docker-compose -f docker-compose.production.yml exec identity-primary \
    /app/plantd-identity create-user \
    --email admin@plantd.com \
    --name "System Administrator" \
    --password "secure_admin_password" \
    --role admin

# Set up initial RBAC permissions
./scripts/setup-state-rbac.sh
```

## SSL Certificate Configuration

### Automatic SSL with Let's Encrypt

The deployment automatically configures SSL certificates using Let's Encrypt:

```bash
# Check certificate status
make ssl-cert-check

# Force certificate renewal (if needed)
make ssl-cert-renew

# Monitor certificate logs
docker-compose -f docker-compose.production.yml logs traefik | grep -i "certificate"
```

### Manual SSL Certificate Installation

If using custom SSL certificates:

```bash
# Create certificates directory
mkdir -p config/traefik/certs

# Copy your certificates
cp your-domain.crt config/traefik/certs/plantd.crt
cp your-domain.key config/traefik/certs/plantd.key

# Set proper permissions
chmod 600 config/traefik/certs/plantd.key
chmod 644 config/traefik/certs/plantd.crt

# Update Traefik configuration
# Edit config/traefik/traefik.prod.yml to reference your certificates
```

## Monitoring Setup

### 1. Access Monitoring Services

Once deployed, access the monitoring stack:

- **Prometheus**: `https://monitoring.plantd.com`
- **Grafana**: `https://dashboard.plantd.com`
- **Traefik Dashboard**: `https://api.plantd.com:8080` (if enabled)

### 2. Configure Grafana

```bash
# Get Grafana admin password
echo $GRAFANA_PASSWORD

# Log into Grafana at https://dashboard.plantd.com
# Username: admin
# Password: (from environment variable)

# Import PlantD dashboards
# Go to + -> Import -> Upload dashboard JSON files from monitoring/dashboards/
```

### 3. Set Up Alerting

```bash
# Configure alert notification channels in Grafana
# - Slack webhook
# - Email SMTP
# - PagerDuty integration

# Verify alert rules are loaded
curl -s https://monitoring.plantd.com/api/v1/rules | jq '.data.groups[].name'
```

## Security Configuration

### 1. Update Default Passwords

```bash
# Update all default passwords in .env.production
# - POSTGRES_PASSWORD
# - JWT_SECRET  
# - GRAFANA_PASSWORD

# Regenerate JWT secret
openssl rand -base64 32

# Update database password
# Edit .env.production and restart services
```

### 2. Configure Authentication

```bash
# Test authentication flow
curl -X POST https://auth.plantd.com/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "admin@plantd.com", "password": "your_admin_password"}'

# Verify token validation
curl -X POST https://auth.plantd.com/auth/validate \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### 3. Security Hardening

```bash
# Review security headers
curl -I https://api.plantd.com

# Check for security vulnerabilities
docker run --rm -v /var/run/docker.sock:/var/run/docker.sock \
  clair-scanner:latest --ip YOUR_SERVER_IP plantd-state:latest

# Update system packages regularly
sudo apt update && sudo apt upgrade -y
```

## Backup and Recovery

### 1. Configure Automated Backups

```bash
# Create backup script
cat > /home/plantd/backup-plantd.sh << 'EOF'
#!/bin/bash
set -e

BACKUP_DIR="/home/plantd/backups/$(date +%Y%m%d_%H%M%S)"
cd /home/plantd/plantd

# Create backup
make deploy-production-backup

# Rotate old backups (keep last 7 days)
find /home/plantd/backups -type d -mtime +7 -exec rm -rf {} \;

echo "Backup completed: $BACKUP_DIR"
EOF

chmod +x /home/plantd/backup-plantd.sh

# Add to crontab (daily at 2 AM)
echo "0 2 * * * /home/plantd/backup-plantd.sh" | crontab -
```

### 2. Test Backup and Restore

```bash
# Create test backup
make deploy-production-backup

# List available backups
ls -la backups/

# Test restore (to staging environment)
make deploy-production-rollback BACKUP_PATH=backups/20241120_020000
```

### 3. Disaster Recovery Plan

1. **Service Failure**: Use blue/green deployment for quick recovery
2. **Data Corruption**: Restore from most recent backup
3. **Complete System Failure**: Rebuild on new server using backup
4. **Network Issues**: Verify DNS and firewall configuration

## Post-deployment Validation

### 1. Health Checks

```bash
# Run comprehensive health checks
./scripts/test-production.sh

# Check all service endpoints
curl -f https://api.plantd.com/health
curl -f https://auth.plantd.com/health
curl -f https://status.plantd.com/health
curl -f https://monitoring.plantd.com/-/healthy
curl -f https://dashboard.plantd.com/api/health
```

### 2. Performance Testing

```bash
# Run load tests
make test-load

# Run stress tests with higher load
make test-load-custom USERS=50 OPERATIONS=1000

# Monitor performance metrics
# Check Grafana dashboards for latency, throughput, errors
```

### 3. Security Testing

```bash
# Run security scans
nmap -sS -O your-server-ip

# Test SSL configuration
curl -I https://api.plantd.com | grep -i security

# Verify authentication
./scripts/test-grpc-client.sh
```

## Maintenance

### 1. Regular Updates

```bash
# Update PlantD to new version
git fetch --tags
git checkout v1.1.0  # New version
make deploy-production-blue-green

# Update Docker images
docker-compose -f docker-compose.production.yml pull
make deploy-production-blue-green

# Update system packages
sudo apt update && sudo apt upgrade -y
```

### 2. Log Management

```bash
# View application logs
make deploy-production-logs

# Rotate Docker logs
sudo logrotate -f /etc/logrotate.d/docker

# Clean up old logs
docker system prune -f
```

### 3. Monitoring and Alerting

```bash
# Review alerts weekly
curl -s https://monitoring.plantd.com/api/v1/alerts | jq '.data.alerts[].labels.alertname'

# Update alert thresholds if needed
# Edit monitoring/alert-rules.yml and redeploy

# Check certificate expiry monthly
make ssl-cert-check
```

### 4. Database Maintenance

```bash
# Database backup verification
docker-compose -f docker-compose.production.yml exec postgres-primary \
    pg_dump -U plantd plantd_identity | gzip > identity_backup.sql.gz

# Database statistics
docker-compose -f docker-compose.production.yml exec postgres-primary \
    psql -U plantd -d plantd_identity -c "SELECT schemaname,tablename,n_tup_ins,n_tup_upd,n_tup_del FROM pg_stat_user_tables;"

# Vacuum and analyze
docker-compose -f docker-compose.production.yml exec postgres-primary \
    psql -U plantd -d plantd_identity -c "VACUUM ANALYZE;"
```

## Troubleshooting

### Common Issues

#### 1. Services Not Starting

```bash
# Check service status
make deploy-production-status

# Check logs for errors
make deploy-production-logs

# Check resource usage
docker stats

# Check disk space
df -h
```

#### 2. SSL Certificate Issues

```bash
# Check certificate status
make ssl-cert-check

# Check Let's Encrypt logs
docker-compose -f docker-compose.production.yml logs traefik | grep -i acme

# Manual certificate renewal
make ssl-cert-renew
```

#### 3. Authentication Problems

```bash
# Check Identity service logs
make deploy-production-logs SERVICE=identity-primary

# Test database connection
docker-compose -f docker-compose.production.yml exec postgres-primary \
    psql -U plantd -d plantd_identity -c "SELECT 1;"

# Verify JWT secret
echo $JWT_SECRET
```

#### 4. Performance Issues

```bash
# Check resource usage
htop
docker stats

# Check database performance
docker-compose -f docker-compose.production.yml exec postgres-primary \
    psql -U plantd -d plantd_identity -c "SELECT * FROM pg_stat_activity;"

# Review Grafana dashboards for bottlenecks
```

### Emergency Procedures

#### Immediate Rollback

```bash
# Emergency rollback to previous version
make deploy-production-rollback BACKUP_PATH=backups/LATEST

# Stop all services
make deploy-production-stop

# Restart specific service
docker-compose -f docker-compose.production.yml restart SERVICE_NAME
```

#### Service Recovery

```bash
# Restart failed service
docker-compose -f docker-compose.production.yml restart state-primary

# Scale services if needed
docker-compose -f docker-compose.production.yml up -d --scale state-primary=2

# Health check after recovery
make deploy-production-health
```

## Support and Documentation

- **Production Support**: support@plantd.com
- **Documentation**: https://docs.plantd.com
- **Issue Reporting**: https://github.com/plantd/plantd/issues
- **Community**: https://discord.gg/plantd

## Deployment Checklist

### Pre-deployment
- [ ] Server meets minimum requirements
- [ ] DNS records configured
- [ ] Environment variables set
- [ ] Firewall configured
- [ ] SSL certificates ready (or Let's Encrypt configured)

### Deployment
- [ ] Production images built
- [ ] Services deployed successfully
- [ ] Health checks passing
- [ ] SSL certificates working
- [ ] Monitoring stack operational

### Post-deployment
- [ ] Admin user created
- [ ] RBAC permissions configured
- [ ] Backup system configured
- [ ] Performance testing completed
- [ ] Security testing completed
- [ ] Monitoring alerts configured

### Go-Live
- [ ] DNS switched to production
- [ ] Load testing completed
- [ ] Runbook documentation updated
- [ ] Team training completed
- [ ] Support procedures in place

**Deployment Complete!** 🚀

Your PlantD production environment is now ready for use. 
