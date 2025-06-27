# PlantD Production Troubleshooting Guide

**Version**: 1.0.0
**Last Updated**: November 2024
**Audience**: Production Support, DevOps, System Administrators

## Overview

This guide provides systematic troubleshooting procedures for PlantD production environments. It covers common issues, diagnostic techniques, and resolution steps with escalation procedures.

## Table of Contents

1. [Emergency Response](#emergency-response)
2. [Service Health Diagnostics](#service-health-diagnostics)
3. [Authentication Issues](#authentication-issues)
4. [Performance Problems](#performance-problems)
5. [SSL/TLS Certificate Issues](#ssltls-certificate-issues)
6. [Database Problems](#database-problems)
7. [Network and Connectivity](#network-and-connectivity)
8. [Monitoring and Alerting](#monitoring-and-alerting)
9. [Backup and Recovery](#backup-and-recovery)
10. [Log Analysis](#log-analysis)
11. [Escalation Procedures](#escalation-procedures)

## Emergency Response

### Critical Service Outage

**Symptoms**: All services unresponsive, 5xx errors, complete system failure

**Immediate Actions**:

```bash
# 1. Check system health
df -h                    # Disk space
free -h                  # Memory usage
uptime                   # System load
docker system df         # Docker space usage

# 2. Check service status
make deploy-production-status

# 3. Check critical logs
make deploy-production-logs | tail -50

# 4. Emergency rollback if recent deployment
make deploy-production-rollback BACKUP_PATH=backups/LATEST
```

**Decision Tree**:
- **Disk Full** → Clean up space, restart services
- **Memory Exhausted** → Restart services, check for memory leaks
- **Network Issues** → Check firewall, DNS, load balancer
- **Database Down** → Check database health, restore from backup
- **Recent Deployment** → Immediate rollback

### Service Degradation

**Symptoms**: High latency, intermittent errors, partial functionality

**Diagnosis Steps**:

```bash
# 1. Identify affected services
curl -f https://api.plantd.com/health
curl -f https://auth.plantd.com/health

# 2. Check resource usage
docker stats --no-stream

# 3. Review recent changes
git log --oneline -10
docker-compose -f docker-compose.production.yml logs --since="1h"

# 4. Scale services if needed
docker-compose -f docker-compose.production.yml up -d --scale state-primary=2
```

## Service Health Diagnostics

### State Service Issues

#### Problem: State Service Returns 500 Errors

**Diagnostic Commands**:

```bash
# Check State service logs
docker-compose -f docker-compose.production.yml logs state-primary | tail -100

# Test gRPC endpoint directly
plant-grpc state state-grpc get test-key --service=test --grpc-endpoint=http://localhost:8080

# Check BoltDB file integrity
docker-compose -f docker-compose.production.yml exec state-primary \
    ls -la /app/data/state.db

# Check file permissions
docker-compose -f docker-compose.production.yml exec state-primary \
    stat /app/data/state.db
```

**Common Solutions**:

1. **Database Corruption**:
   ```bash
   # Restore from backup
   make deploy-production-rollback BACKUP_PATH=backups/LATEST
   ```

2. **Permission Issues**:
   ```bash
   # Fix file permissions
   docker-compose -f docker-compose.production.yml exec state-primary \
       chown -R app:app /app/data/
   ```

3. **Out of Disk Space**:
   ```bash
   # Clean up Docker images
   docker system prune -f
   
   # Check disk usage
   du -sh /var/lib/docker/*
   ```

#### Problem: State Service High Memory Usage

**Diagnostic Commands**:

```bash
# Monitor memory usage
docker stats state-primary --no-stream

# Check for memory leaks
docker-compose -f docker-compose.production.yml exec state-primary \
    ps aux | grep plantd-state

# Review configuration
docker-compose -f docker-compose.production.yml exec state-primary \
    cat /app/config.yml
```

**Solutions**:

1. **Memory Limit Configuration**:
   ```bash
   # Update docker-compose.production.yml
   services:
     state-primary:
       deploy:
         resources:
           limits:
             memory: 1G
   ```

2. **Restart Service**:
   ```bash
   docker-compose -f docker-compose.production.yml restart state-primary
   ```

### Identity Service Issues

#### Problem: Authentication Failures

**Diagnostic Commands**:

```bash
# Check Identity service logs
docker-compose -f docker-compose.production.yml logs identity-primary | grep -i error

# Test database connection
docker-compose -f docker-compose.production.yml exec postgres-primary \
    psql -U plantd -d plantd_identity -c "SELECT COUNT(*) FROM users;"

# Check JWT secret
echo $JWT_SECRET | wc -c  # Should be > 32 characters

# Test login directly
curl -X POST https://auth.plantd.com/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "admin@plantd.com", "password": "test"}'
```

**Common Solutions**:

1. **Database Connection Issues**:
   ```bash
   # Check database status
   docker-compose -f docker-compose.production.yml ps postgres-primary
   
   # Restart database
   docker-compose -f docker-compose.production.yml restart postgres-primary
   ```

2. **JWT Secret Problems**:
   ```bash
   # Regenerate JWT secret
   openssl rand -base64 32
   
   # Update .env.production and restart
   docker-compose -f docker-compose.production.yml restart identity-primary
   ```

#### Problem: Token Validation Errors

**Diagnostic Commands**:

```bash
# Check token expiry settings
docker-compose -f docker-compose.production.yml exec identity-primary \
    cat /app/config.yml | grep -i token

# Test token validation
curl -X POST https://auth.plantd.com/auth/validate \
  -H "Authorization: Bearer YOUR_TOKEN"

# Check system time synchronization
timedatectl status
```

**Solutions**:

1. **Time Synchronization**:
   ```bash
   # Sync system time
   sudo timedatectl set-ntp true
   sudo systemctl restart systemd-timesyncd
   ```

2. **Token Settings**:
   ```bash
   # Check token expiry in config
   # Update config.yml if needed and restart service
   ```

## Authentication Issues

### RBAC Permission Denied

**Symptoms**: 403 Forbidden errors, access denied messages

**Diagnostic Steps**:

```bash
# 1. Check user permissions
curl -X POST https://auth.plantd.com/auth/validate \
  -H "Authorization: Bearer $TOKEN" | jq '.user.permissions'

# 2. Check required permissions for endpoint
docker-compose -f docker-compose.production.yml logs traefik | grep -i "403"

# 3. Review RBAC configuration
docker-compose -f docker-compose.production.yml exec identity-primary \
    psql -U plantd -d plantd_identity -c "SELECT * FROM role_permissions;"
```

**Solutions**:

1. **Update User Permissions**:
   ```bash
   # Add permission to user role
   docker-compose -f docker-compose.production.yml exec identity-primary \
       /app/plantd-identity add-permission \
       --user-email="user@example.com" \
       --permission="state:data:write"
   ```

2. **Fix RBAC Configuration**:
   ```bash
   # Reset RBAC to defaults
   ./scripts/setup-state-rbac.sh
   ```

### Token Expiry Issues

**Symptoms**: Sudden authentication failures, expired token errors

**Diagnostic Steps**:

```bash
# 1. Check token expiry
echo $TOKEN | base64 -d | jq '.exp'

# 2. Compare with current time
date +%s

# 3. Check refresh token
curl -X POST https://auth.plantd.com/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{"refresh_token": "$REFRESH_TOKEN"}'
```

**Solutions**:

1. **Refresh Token**:
   ```bash
   # Use refresh token to get new access token
   plant-grpc auth auth-grpc refresh
   ```

2. **Re-authenticate**:
   ```bash
   # Login again
   plant-grpc auth auth-grpc login
   ```

## Performance Problems

### High Latency

**Symptoms**: Slow response times, timeout errors

**Diagnostic Commands**:

```bash
# 1. Check system resources
htop
iotop

# 2. Monitor service performance
docker stats --no-stream

# 3. Test endpoint latency
time curl -X POST https://api.plantd.com/plantd.state.v1.StateService/Get \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"key": "test", "service": "test"}'

# 4. Check database performance
docker-compose -f docker-compose.production.yml exec postgres-primary \
    psql -U plantd -d plantd_identity -c "SELECT * FROM pg_stat_activity WHERE state = 'active';"
```

**Solutions**:

1. **Scale Services**:
   ```bash
   # Scale up State service
   docker-compose -f docker-compose.production.yml up -d --scale state-primary=3
   
   # Scale up Identity service
   docker-compose -f docker-compose.production.yml up -d --scale identity-primary=2
   ```

2. **Database Optimization**:
   ```bash
   # Vacuum and analyze database
   docker-compose -f docker-compose.production.yml exec postgres-primary \
       psql -U plantd -d plantd_identity -c "VACUUM ANALYZE;"
   
   # Check slow queries
   docker-compose -f docker-compose.production.yml exec postgres-primary \
       psql -U plantd -d plantd_identity -c "SELECT query, calls, total_time, mean_time FROM pg_stat_statements ORDER BY total_time DESC LIMIT 10;"
   ```

3. **Resource Allocation**:
   ```bash
   # Update resource limits in docker-compose.production.yml
   services:
     state-primary:
       deploy:
         resources:
           limits:
             memory: 2G
             cpus: '1.0'
   ```

### High Error Rate

**Symptoms**: Frequent 5xx errors, service instability

**Diagnostic Steps**:

```bash
# 1. Check error logs
docker-compose -f docker-compose.production.yml logs | grep -i error | tail -50

# 2. Monitor error rate
curl -s https://monitoring.plantd.com/api/v1/query?query=rate(http_requests_total{status=~"5.."}[5m])

# 3. Check service health
make deploy-production-health

# 4. Review recent deployments
git log --oneline --since="24 hours ago"
```

**Solutions**:

1. **Rollback Recent Changes**:
   ```bash
   # Rollback to previous version
   make deploy-production-rollback BACKUP_PATH=backups/LATEST
   ```

2. **Restart Failing Services**:
   ```bash
   # Identify and restart failing service
   docker-compose -f docker-compose.production.yml restart SERVICE_NAME
   ```

3. **Circuit Breaker**:
   ```bash
   # Check Traefik circuit breaker status
   curl -s http://localhost:8080/api/http/services | jq '.'
   ```

## SSL/TLS Certificate Issues

### Certificate Expired

**Symptoms**: SSL certificate warnings, HTTPS connection failures

**Diagnostic Commands**:

```bash
# 1. Check certificate expiry
openssl s_client -connect api.plantd.com:443 -servername api.plantd.com < /dev/null 2>/dev/null | openssl x509 -noout -dates

# 2. Check Let's Encrypt logs
docker-compose -f docker-compose.production.yml logs traefik | grep -i acme

# 3. Check certificate files
ls -la config/traefik/letsencrypt/
```

**Solutions**:

1. **Renew Let's Encrypt Certificate**:
   ```bash
   # Force certificate renewal
   docker-compose -f docker-compose.production.yml restart traefik
   
   # Check renewal logs
   docker-compose -f docker-compose.production.yml logs traefik | grep -i certificate
   ```

2. **Manual Certificate Renewal**:
   ```bash
   # Remove existing certificates
   rm -rf config/traefik/letsencrypt/acme.json
   
   # Restart Traefik to obtain new certificates
   docker-compose -f docker-compose.production.yml restart traefik
   ```

### Certificate Chain Issues

**Symptoms**: Certificate chain errors, browser warnings

**Diagnostic Commands**:

```bash
# 1. Check certificate chain
openssl s_client -connect api.plantd.com:443 -showcerts

# 2. Verify certificate chain
curl -I https://api.plantd.com

# 3. Check SSL Labs rating
curl -s "https://api.ssllabs.com/api/v3/analyze?host=api.plantd.com"
```

**Solutions**:

1. **Fix Certificate Chain**:
   ```bash
   # Update Traefik configuration to include intermediate certificates
   # Edit config/traefik/traefik.prod.yml
   ```

2. **Use Different CA**:
   ```bash
   # Switch to ZeroSSL or other ACME provider
   # Update certificatesResolvers in config/traefik/traefik.prod.yml
   ```

## Database Problems

### PostgreSQL Connection Issues

**Symptoms**: Database connection refused, identity service failures

**Diagnostic Commands**:

```bash
# 1. Check PostgreSQL status
docker-compose -f docker-compose.production.yml ps postgres-primary

# 2. Check database logs
docker-compose -f docker-compose.production.yml logs postgres-primary | tail -50

# 3. Test connection
docker-compose -f docker-compose.production.yml exec postgres-primary \
    psql -U plantd -d plantd_identity -c "SELECT 1;"

# 4. Check database size and connections
docker-compose -f docker-compose.production.yml exec postgres-primary \
    psql -U plantd -d plantd_identity -c "SELECT datname, pg_size_pretty(pg_database_size(datname)) FROM pg_database;"
```

**Solutions**:

1. **Restart Database**:
   ```bash
   # Restart PostgreSQL
   docker-compose -f docker-compose.production.yml restart postgres-primary
   
   # Check startup logs
   docker-compose -f docker-compose.production.yml logs postgres-primary
   ```

2. **Check Connection Limits**:
   ```bash
   # Check current connections
   docker-compose -f docker-compose.production.yml exec postgres-primary \
       psql -U plantd -d plantd_identity -c "SELECT count(*) FROM pg_stat_activity;"
   
   # Check max connections
   docker-compose -f docker-compose.production.yml exec postgres-primary \
       psql -U plantd -d plantd_identity -c "SHOW max_connections;"
   ```

3. **Database Corruption**:
   ```bash
   # Check for corruption
   docker-compose -f docker-compose.production.yml exec postgres-primary \
       psql -U plantd -d plantd_identity -c "SELECT * FROM pg_stat_database WHERE datname = 'plantd_identity';"
   
   # Restore from backup if needed
   make deploy-production-rollback BACKUP_PATH=backups/LATEST
   ```

### BoltDB (State Service) Issues

**Symptoms**: State service crashes, data corruption errors

**Diagnostic Commands**:

```bash
# 1. Check BoltDB file
docker-compose -f docker-compose.production.yml exec state-primary \
    ls -la /app/data/state.db

# 2. Check file integrity
docker-compose -f docker-compose.production.yml exec state-primary \
    file /app/data/state.db

# 3. Check disk space
docker-compose -f docker-compose.production.yml exec state-primary \
    df -h /app/data/
```

**Solutions**:

1. **Fix File Permissions**:
   ```bash
   # Correct ownership
   docker-compose -f docker-compose.production.yml exec state-primary \
       chown -R app:app /app/data/
   ```

2. **Restore from Backup**:
   ```bash
   # Stop service
   docker-compose -f docker-compose.production.yml stop state-primary
   
   # Restore backup
   make deploy-production-rollback BACKUP_PATH=backups/LATEST
   ```

## Network and Connectivity

### DNS Resolution Issues

**Symptoms**: Service discovery failures, cannot resolve service names

**Diagnostic Commands**:

```bash
# 1. Check DNS resolution
nslookup api.plantd.com
dig api.plantd.com

# 2. Check Docker network
docker network ls
docker network inspect plantd_default

# 3. Test internal service communication
docker-compose -f docker-compose.production.yml exec traefik \
    nslookup state-primary
```

**Solutions**:

1. **Restart Docker Network**:
   ```bash
   # Recreate network
   docker-compose -f docker-compose.production.yml down
   docker network prune -f
   docker-compose -f docker-compose.production.yml up -d
   ```

2. **Check DNS Configuration**:
   ```bash
   # Check system DNS
   cat /etc/resolv.conf
   
   # Test DNS servers
   nslookup google.com 8.8.8.8
   ```

### Firewall Issues

**Symptoms**: Connection timeouts, ports not accessible

**Diagnostic Commands**:

```bash
# 1. Check firewall status
sudo ufw status verbose

# 2. Check listening ports
sudo netstat -tlnp | grep -E ":(80|443|8080|8443)"

# 3. Test port connectivity
telnet api.plantd.com 443
```

**Solutions**:

1. **Open Required Ports**:
   ```bash
   # Open HTTP/HTTPS ports
   sudo ufw allow 80/tcp
   sudo ufw allow 443/tcp
   sudo ufw allow 8080/tcp
   sudo ufw allow 8443/tcp
   ```

2. **Check Docker Port Mapping**:
   ```bash
   # Verify port mappings
   docker-compose -f docker-compose.production.yml ps
   ```

## Monitoring and Alerting

### Prometheus Issues

**Symptoms**: Metrics not available, monitoring dashboards empty

**Diagnostic Commands**:

```bash
# 1. Check Prometheus status
curl -f https://monitoring.plantd.com/-/healthy

# 2. Check targets
curl -s https://monitoring.plantd.com/api/v1/targets | jq '.data.activeTargets[].health'

# 3. Check configuration
docker-compose -f docker-compose.production.yml exec prometheus \
    cat /etc/prometheus/prometheus.yml
```

**Solutions**:

1. **Restart Prometheus**:
   ```bash
   docker-compose -f docker-compose.production.yml restart prometheus
   ```

2. **Fix Configuration**:
   ```bash
   # Check and fix prometheus.yml
   # Restart after configuration changes
   ```

### Grafana Access Issues

**Symptoms**: Cannot access dashboards, login failures

**Diagnostic Commands**:

```bash
# 1. Check Grafana status
curl -f https://dashboard.plantd.com/api/health

# 2. Check Grafana logs
docker-compose -f docker-compose.production.yml logs grafana

# 3. Check admin password
echo $GRAFANA_PASSWORD
```

**Solutions**:

1. **Reset Admin Password**:
   ```bash
   # Reset Grafana admin password
   docker-compose -f docker-compose.production.yml exec grafana \
       grafana-cli admin reset-admin-password newpassword
   ```

2. **Check Configuration**:
   ```bash
   # Verify Grafana configuration
   docker-compose -f docker-compose.production.yml exec grafana \
       cat /etc/grafana/grafana.ini
   ```

## Backup and Recovery

### Backup Failures

**Symptoms**: Backup script failures, missing backup files

**Diagnostic Commands**:

```bash
# 1. Check backup directory
ls -la backups/
df -h backups/

# 2. Test backup manually
make deploy-production-backup

# 3. Check backup script logs
journalctl -u backup-plantd --since="24 hours ago"
```

**Solutions**:

1. **Fix Disk Space**:
   ```bash
   # Clean old backups
   find backups/ -type d -mtime +7 -exec rm -rf {} \;
   ```

2. **Fix Permissions**:
   ```bash
   # Fix backup directory permissions
   chmod -R 755 backups/
   chown -R plantd:plantd backups/
   ```

### Recovery Failures

**Symptoms**: Restore process fails, data inconsistency

**Diagnostic Commands**:

```bash
# 1. Check backup integrity
ls -la backups/BACKUP_NAME/
file backups/BACKUP_NAME/*.sql.gz

# 2. Test restore in isolated environment
# Create test environment and restore there first
```

**Solutions**:

1. **Manual Restore**:
   ```bash
   # Stop services
   docker-compose -f docker-compose.production.yml down
   
   # Restore manually
   # [specific restore commands based on backup type]
   
   # Start services
   docker-compose -f docker-compose.production.yml up -d
   ```

## Log Analysis

### Centralized Logging Issues

**Symptoms**: Missing logs, log aggregation failures

**Diagnostic Commands**:

```bash
# 1. Check Fluent Bit status
docker-compose -f docker-compose.production.yml ps fluent-bit

# 2. Check log files
ls -la logs/
du -sh logs/*

# 3. Check Fluent Bit configuration
docker-compose -f docker-compose.production.yml exec fluent-bit \
    cat /fluent-bit/etc/fluent-bit.conf
```

**Solutions**:

1. **Restart Log Aggregation**:
   ```bash
   docker-compose -f docker-compose.production.yml restart fluent-bit
   ```

2. **Check Log Rotation**:
   ```bash
   # Configure log rotation
   sudo logrotate -f /etc/logrotate.d/docker
   ```

### Log Analysis Techniques

**Common Log Patterns**:

```bash
# 1. Error analysis
grep -i error logs/*.log | head -20

# 2. Performance issues
grep -i "slow\|timeout\|latency" logs/*.log

# 3. Authentication failures
grep -i "auth\|login\|token" logs/*.log | grep -i "fail\|error"

# 4. Database issues
grep -i "database\|sql\|connection" logs/*.log | grep -i "error"
```

## Escalation Procedures

### Severity Levels

**Critical (P0)**: Complete service outage
- **Response Time**: 5 minutes
- **Action**: Immediate escalation to on-call engineer
- **Notification**: All stakeholders

**High (P1)**: Significant degradation
- **Response Time**: 15 minutes
- **Action**: Escalate to senior engineer
- **Notification**: Technical team + management

**Medium (P2)**: Minor issues
- **Response Time**: 1 hour
- **Action**: Handle within team
- **Notification**: Technical team

**Low (P3)**: Enhancement requests
- **Response Time**: 24 hours
- **Action**: Normal queue
- **Notification**: Requestor only

### Escalation Contacts

```bash
# Emergency Contact Information
# On-call Engineer: +1-XXX-XXX-XXXX
# Senior Engineer: +1-XXX-XXX-XXXX
# Engineering Manager: +1-XXX-XXX-XXXX
# CTO: +1-XXX-XXX-XXXX

# Notification Channels
# Slack: #plantd-alerts
# Email: alerts@plantd.com
# PagerDuty: incident escalation
```

### Post-Incident Process

1. **Immediate Response**: Restore service
2. **Incident Documentation**: Record timeline and actions
3. **Root Cause Analysis**: Identify underlying cause
4. **Post-Mortem**: Review with team
5. **Action Items**: Implement preventive measures

## Quick Reference

### Essential Commands

```bash
# Service Status
make deploy-production-status
make deploy-production-health

# Logs
make deploy-production-logs
make deploy-production-logs SERVICE=traefik

# Emergency Actions
make deploy-production-rollback BACKUP_PATH=backups/LATEST
make deploy-production-stop
make deploy-production-start

# Resource Monitoring
docker stats --no-stream
df -h
free -h
```

### Emergency Checklist

- [ ] Check system resources (CPU, memory, disk)
- [ ] Verify service status
- [ ] Review recent changes
- [ ] Check critical logs
- [ ] Test service endpoints
- [ ] Consider rollback if recent deployment
- [ ] Document incident timeline
- [ ] Notify stakeholders
- [ ] Execute recovery plan
- [ ] Verify restoration
- [ ] Post-incident review

Remember: **Document everything** and **communicate early and often** during incidents. 
