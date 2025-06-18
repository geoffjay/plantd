# SSL/TLS Quick Reference

**Quick solutions for common SSL/TLS scenarios in PlantD development.**

## 🚀 Quick Start

### Option 1: Accept Self-Signed Certificate (Fastest)
```bash
# 1. Start services
overmind start

# 2. Open browser to https://localhost:8443
# 3. Click "Advanced" → "Proceed to localhost (unsafe)"
# ✅ Done! SSE and all features work.
```

### Option 2: Use HTTP Mode (No Certificate Warnings)
```bash
# 1. Enable HTTP mode
export PLANTD_APP_USE_HTTP=true

# 2. Restart app service
overmind restart app

# 3. Open browser to http://localhost:8080
# ✅ Done! No certificate warnings.
```

### Option 3: Use Trusted Certificates (Best Experience)
```bash
# 1. Install mkcert
brew install mkcert  # macOS
# sudo apt install mkcert  # Ubuntu

# 2. Install local CA
mkcert -install

# 3. Generate certificate
mkcert localhost 127.0.0.1 ::1

# 4. Install certificate
mv localhost+2.pem cert/app-cert.pem
mv localhost+2-key.pem cert/app-key.pem

# 5. Restart service
overmind restart app

# 6. Open browser to https://localhost:8443
# ✅ Done! Fully trusted certificate.
```

## 🔧 Common Issues & Solutions

### "ERR_CERT_AUTHORITY_INVALID" Error
**Quick Fix:**
```bash
# Option A: Accept in browser (recommended)
# Click "Advanced" → "Proceed to localhost (unsafe)"

# Option B: Switch to HTTP
export PLANTD_APP_USE_HTTP=true && overmind restart app

# Option C: Use mkcert (see Option 3 above)
```

### SSE Connection Failures
**Quick Fix:**
```bash
# 1. First, accept the certificate in browser
# 2. Then test SSE:
curl -k https://localhost:8443/sse

# If still failing, try HTTP mode:
export PLANTD_APP_USE_HTTP=true && overmind restart app
```

### Certificate Expired
**Quick Fix:**
```bash
# Remove old certificate and restart
rm cert/app-*.pem && overmind restart app
```

### Port Already in Use
**Quick Fix:**
```bash
# Check what's using the port
lsof -i :8443

# Use different port
export PLANTD_APP_BIND_PORT="9443" && overmind restart app
```

## 🧪 Testing Commands

```bash
# Test SSL configuration
./scripts/test-ssl

# Test HTTPS connection
curl -k -I https://localhost:8443/

# Check certificate details
openssl x509 -in cert/app-cert.pem -text -noout | grep -A 5 "Subject Alternative Name"

# Check what's running on ports
lsof -i :8443 && lsof -i :8080
```

## 📋 Environment Variables

```bash
# Core SSL/TLS settings
export PLANTD_APP_USE_HTTP="true"           # Enable HTTP mode
export PLANTD_APP_BIND_PORT="8443"          # Change port
export PLANTD_APP_TLS_CERT="cert/app-cert.pem"  # Custom cert
export PLANTD_APP_TLS_KEY="cert/app-key.pem"    # Custom key

# Development settings
export PLANTD_APP_LOG_LEVEL="debug"         # Debug logging
export PLANTD_APP_BIND_ADDRESS="0.0.0.0"    # Bind all interfaces
```

## 🎯 Recommendations

**For Daily Development:**
- ✅ **Use Option 1** (accept self-signed) - fastest setup
- ✅ **Use Option 3** (mkcert) - best long-term experience

**For Debugging:**
- ✅ **Use Option 2** (HTTP mode) - eliminates SSL variables

**For Production:**
- ✅ **Use trusted CA certificates** (Let's Encrypt, commercial)
- ❌ **Never use self-signed or HTTP mode**

## 📖 Full Documentation

For comprehensive information, see:
- **[SSL/TLS Configuration Guide](ssl-tls-configuration.md)** - Complete guide
- **[App Service README](../app/README.md)** - App-specific configuration 
