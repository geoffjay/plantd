# SSL/TLS Configuration Guide for PlantD

This document provides comprehensive guidance for configuring SSL/TLS certificates across PlantD services, with a focus on development and production environments.

## Overview

PlantD services support both HTTP and HTTPS modes, with HTTPS being the default and recommended configuration for security. This guide covers certificate management, troubleshooting, and best practices.

## 🔐 Certificate Management

### Development Certificates

#### Self-Signed Certificates (Default)

PlantD services automatically generate self-signed certificates for development with the following features:

- **Validity Period**: 1 year from generation
- **Subject Alternative Names (SAN)**: 
  - `localhost`, `*.localhost`
  - `plantd.local`, `*.plantd.local`
  - IP addresses: `127.0.0.1`, `::1`
- **Key Size**: 2048-bit RSA
- **Auto-Regeneration**: Generated automatically if missing

**Certificate Locations by Service:**
- **App Service**: `cert/app-cert.pem`, `cert/app-key.pem`
- **Identity Service**: `cert/identity-cert.pem`, `cert/identity-key.pem`
- **Other Services**: `cert/{service}-cert.pem`, `cert/{service}-key.pem`

#### Browser Certificate Acceptance

When using self-signed certificates, browsers will show security warnings. This is expected and safe for development:

**Chrome/Edge:**
1. Click "Advanced"
2. Click "Proceed to localhost (unsafe)"

**Firefox:**
1. Click "Advanced"
2. Click "Accept the Risk and Continue"

**Safari:**
1. Click "Show Details"
2. Click "visit this website"
3. Click "Visit Website"

### Production Certificates

#### Trusted CA Certificates

For production environments, use certificates from trusted Certificate Authorities:

**Let's Encrypt (Free):**
```bash
# Install certbot
sudo apt install certbot  # Ubuntu/Debian
brew install certbot       # macOS

# Generate certificate
sudo certbot certonly --standalone -d plantd.yourdomain.com

# Certificate files will be in:
# /etc/letsencrypt/live/plantd.yourdomain.com/fullchain.pem
# /etc/letsencrypt/live/plantd.yourdomain.com/privkey.pem
```

**Commercial CA:**
- Purchase certificate from trusted CA (DigiCert, GlobalSign, etc.)
- Follow CA-specific installation instructions
- Ensure certificate includes all required domain names and SANs

#### Certificate Installation

```bash
# Set certificate paths for any service
export PLANTD_{SERVICE}_TLS_CERT="/path/to/certificate.pem"
export PLANTD_{SERVICE}_TLS_KEY="/path/to/private-key.pem"

# Example for App Service
export PLANTD_APP_TLS_CERT="/etc/ssl/certs/plantd-app.pem"
export PLANTD_APP_TLS_KEY="/etc/ssl/private/plantd-app.key"
```

## 🛠️ Advanced Certificate Options

### Option 1: mkcert (Recommended for Development)

`mkcert` creates locally-trusted certificates that don't require browser warnings:

#### Installation
```bash
# macOS
brew install mkcert

# Ubuntu/Debian
sudo apt install mkcert

# Windows
choco install mkcert

# Or download from: https://github.com/FiloSottile/mkcert/releases
```

#### Usage
```bash
# Install local CA
mkcert -install

# Generate certificate for App Service
mkcert localhost 127.0.0.1 ::1 plantd.local

# Install certificate (example for App Service)
mv localhost+3.pem cert/app-cert.pem
mv localhost+3-key.pem cert/app-key.pem

# Restart service
overmind restart app

# Now https://localhost:8443 is fully trusted! ✅
```

#### Benefits
- ✅ **No browser warnings**
- ✅ **Trusted by all browsers**
- ✅ **Perfect for development**
- ✅ **Easy to regenerate**
- ✅ **Works with all PlantD services**

### Option 2: Custom Certificate Authority

For organizations requiring custom CAs:

```bash
# Generate CA private key
openssl genrsa -out ca-key.pem 4096

# Generate CA certificate
openssl req -new -x509 -days 365 -key ca-key.pem -out ca-cert.pem

# Generate service private key
openssl genrsa -out service-key.pem 2048

# Generate certificate signing request
openssl req -new -key service-key.pem -out service.csr

# Sign certificate with CA
openssl x509 -req -days 365 -in service.csr -CA ca-cert.pem -CAkey ca-key.pem -out service-cert.pem
```

## 🌐 HTTP vs HTTPS Modes

### HTTPS Mode (Default)

**Recommended for:**
- ✅ Production environments
- ✅ Security testing
- ✅ Authentication testing
- ✅ Realistic development conditions

**Configuration:**
```bash
# HTTPS is default - no additional configuration needed
# Access via: https://localhost:8443 (App Service)
```

### HTTP Mode (Development Alternative)

**Recommended for:**
- ✅ Rapid development
- ✅ Certificate troubleshooting
- ✅ Network debugging
- ✅ Load testing

**Configuration:**
```bash
# Enable HTTP mode (App Service example)
export PLANTD_APP_USE_HTTP=true
overmind restart app

# Access via: http://localhost:8080
```

**Important Notes:**
- ⚠️ **Development only** - automatically disabled in production
- ⚠️ **No encryption** - not suitable for sensitive data
- ⚠️ **Session cookies** may not work properly without HTTPS

## 🔧 Service-Specific Configuration

### App Service (Port 8443/8080)

**HTTPS Configuration:**
```bash
export PLANTD_APP_BIND_PORT="8443"
export PLANTD_APP_TLS_CERT="cert/app-cert.pem"
export PLANTD_APP_TLS_KEY="cert/app-key.pem"
```

**HTTP Configuration:**
```bash
export PLANTD_APP_USE_HTTP="true"
export PLANTD_APP_BIND_PORT="8080"
```

### Identity Service (Port 8080)

**HTTPS Configuration:**
```bash
export PLANTD_IDENTITY_HTTP_PORT="8080"
export PLANTD_IDENTITY_TLS_CERT="cert/identity-cert.pem"
export PLANTD_IDENTITY_TLS_KEY="cert/identity-key.pem"
```

### Other Services

Follow similar patterns for other PlantD services:
```bash
export PLANTD_{SERVICE}_TLS_CERT="cert/{service}-cert.pem"
export PLANTD_{SERVICE}_TLS_KEY="cert/{service}-key.pem"
```

## 🧪 Testing and Validation

### SSL/TLS Testing Script

Use the built-in testing script for comprehensive validation:

```bash
# Test App Service SSL/TLS
./scripts/test-ssl

# Test specific aspects
./scripts/test-ssl https    # HTTPS only
./scripts/test-ssl http     # HTTP only
./scripts/test-ssl help     # Show help
```

### Manual Testing Commands

```bash
# Test HTTPS connection
curl -k -I https://localhost:8443/

# Test certificate details
openssl s_client -connect localhost:8443 -servername localhost

# Check certificate file
openssl x509 -in cert/app-cert.pem -text -noout

# Verify certificate chain
openssl verify -CAfile ca-cert.pem service-cert.pem
```

### Certificate Validation Checklist

- [ ] **Valid dates**: Certificate not expired
- [ ] **Subject Alternative Names**: Includes all required hostnames/IPs
- [ ] **Key usage**: Includes `Digital Signature` and `Key Encipherment`
- [ ] **Extended key usage**: Includes `Server Authentication`
- [ ] **Chain of trust**: Certificate chain is complete
- [ ] **Private key match**: Private key matches certificate

## 🚨 Troubleshooting

### Common Certificate Errors

#### 1. `ERR_CERT_AUTHORITY_INVALID`
**Cause**: Self-signed certificate not trusted
**Solutions**:
- Accept certificate in browser (development)
- Use mkcert for trusted certificates
- Use HTTP mode temporarily
- Install custom CA certificate

#### 2. `ERR_CERT_COMMON_NAME_INVALID`
**Cause**: Certificate doesn't include hostname in SAN
**Solutions**:
- Regenerate certificate with proper SAN entries
- Access via IP address if included in certificate
- Add hostname to certificate SAN list

#### 3. `ERR_CERT_DATE_INVALID`
**Cause**: Certificate expired or not yet valid
**Solutions**:
```bash
# Check certificate validity
openssl x509 -in cert/app-cert.pem -text -noout | grep "Not Before\|Not After"

# Regenerate expired certificate
rm cert/app-*.pem && overmind restart app
```

#### 4. `ERR_SSL_PROTOCOL_ERROR`
**Cause**: SSL/TLS configuration mismatch
**Solutions**:
- Check certificate and key file permissions
- Verify certificate and key file formats
- Ensure certificate and private key match

### Certificate Regeneration

```bash
# Remove existing certificates
rm cert/{service}-*.pem

# Restart service to regenerate
overmind restart {service}

# Or manually regenerate using mkcert
mkcert localhost 127.0.0.1 ::1
mv localhost+2.pem cert/{service}-cert.pem
mv localhost+2-key.pem cert/{service}-key.pem
```

### Network Debugging

```bash
# Check what's listening on ports
lsof -i :8443
lsof -i :8080

# Test network connectivity
telnet localhost 8443
nc -zv localhost 8443

# Check certificate from remote
echo | openssl s_client -connect hostname:8443 -servername hostname
```

## 📋 Best Practices

### Development Environment

1. **Use mkcert** for the best development experience
2. **Accept self-signed certificates** if mkcert isn't available
3. **Use HTTP mode** only for specific debugging scenarios
4. **Test both HTTP and HTTPS** modes during development
5. **Regenerate certificates** periodically (every 6 months)

### Production Environment

1. **Never use self-signed certificates** in production
2. **Use certificates from trusted CAs** (Let's Encrypt, commercial)
3. **Implement certificate monitoring** and renewal automation
4. **Use strong cipher suites** and modern TLS versions
5. **Implement HSTS headers** for enhanced security
6. **Monitor certificate expiration** dates

### Security Considerations

1. **Protect private keys** with appropriate file permissions (600)
2. **Use separate certificates** for each service/domain
3. **Implement certificate pinning** where appropriate
4. **Regular security audits** of SSL/TLS configuration
5. **Keep certificates in secure storage** (HashiCorp Vault, etc.)

## 🔗 Related Documentation

- [App Service README](../app/README.md) - App-specific SSL/TLS configuration
- [Identity Service Documentation](identity/README.md) - Identity service SSL setup
- [Production Deployment Guide](deployment.md) - Production SSL/TLS setup
- [Security Best Practices](security.md) - Overall security guidelines

## 📞 Support

For SSL/TLS related issues:

1. **Check this documentation** first
2. **Run the test script**: `./scripts/test-ssl`
3. **Check service logs**: `overmind logs {service}`
4. **Verify certificate details**: `openssl x509 -in cert/{service}-cert.pem -text -noout`
5. **Test network connectivity**: Basic network troubleshooting

Common solutions often involve certificate regeneration or switching between HTTP/HTTPS modes for development. 
