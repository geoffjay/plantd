#!/bin/bash

set -e

echo "Testing Traefik Gateway Setup (Phase 4)"
echo "======================================="

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Test 1: Check if required files exist
log_info "Testing file structure..."

required_files=(
    "config/traefik/traefik.yml"
    "config/traefik/config/services.yml"
    "docker-compose.traefik.yml"
    "docker/state/Dockerfile.grpc"
    "scripts/traefik-dev.sh"
    "config/monitoring/prometheus.yml"
)

missing_files=()
for file in "${required_files[@]}"; do
    if [[ -f "$file" ]]; then
        log_success "✅ $file exists"
    else
        log_error "❌ $file missing"
        missing_files+=("$file")
    fi
done

if [[ ${#missing_files[@]} -eq 0 ]]; then
    log_success "✅ All required files present"
else
    log_error "❌ Missing ${#missing_files[@]} required files"
    exit 1
fi

# Test 2: Validate Traefik configuration syntax
log_info "Validating Traefik configuration..."

if command -v traefik &> /dev/null; then
    if traefik validate --configFile=config/traefik/traefik.yml &> /dev/null; then
        log_success "✅ Traefik configuration is valid"
    else
        log_error "❌ Traefik configuration has syntax errors"
        exit 1
    fi
else
    log_warning "⚠️  Traefik not installed, skipping syntax validation"
fi

# Test 3: Validate Docker Compose syntax
log_info "Validating Docker Compose configuration..."

if command -v docker-compose &> /dev/null; then
    if docker-compose -f docker-compose.traefik.yml config &> /dev/null; then
        log_success "✅ Docker Compose configuration is valid"
    else
        log_error "❌ Docker Compose configuration has syntax errors"
        exit 1
    fi
else
    log_warning "⚠️  Docker Compose not installed, skipping syntax validation"
fi

# Test 4: Check if gRPC service can be built
log_info "Testing gRPC service build..."

if make build-state-grpc &> /dev/null; then
    log_success "✅ State gRPC service builds successfully"
else
    log_error "❌ State gRPC service build failed"
    exit 1
fi

# Test 5: Check script permissions
log_info "Checking script permissions..."

if [[ -x "scripts/traefik-dev.sh" ]]; then
    log_success "✅ Traefik development script is executable"
else
    log_error "❌ Traefik development script is not executable"
    exit 1
fi

# Test 6: Validate Makefile targets
log_info "Checking Makefile targets..."

makefile_targets=(
    "traefik-dev-start"
    "traefik-dev-stop"
    "traefik-dev-status"
    "traefik-dev-test"
)

for target in "${makefile_targets[@]}"; do
    if grep -q "^$target:" Makefile; then
        log_success "✅ Makefile target '$target' exists"
    else
        log_error "❌ Makefile target '$target' missing"
        exit 1
    fi
done

# Test 7: Check service endpoints in Traefik config
log_info "Validating service routing configuration..."

endpoints_to_check=(
    "/plantd.state.v1.StateService/"
    "/mdp/"
    "/health"
    "/auth/"
)

for endpoint in "${endpoints_to_check[@]}"; do
    if grep -q "$endpoint" config/traefik/config/services.yml; then
        log_success "✅ Endpoint '$endpoint' configured in Traefik"
    else
        log_error "❌ Endpoint '$endpoint' not found in Traefik config"
        exit 1
    fi
done

# Test 8: Verify Docker networking configuration
log_info "Checking Docker network configuration..."

if grep -q "network.*plantd" docker-compose.traefik.yml; then
    log_success "✅ Docker network 'plantd' configured"
else
    log_error "❌ Docker network 'plantd' not configured"
    exit 1
fi

# Test 9: Check port mappings
log_info "Validating port mappings..."

expected_ports=(
    "80:80"      # HTTP
    "443:443"    # HTTPS  
    "8080:8080"  # gRPC
    "8443:8443"  # gRPC over TLS
)

for port in "${expected_ports[@]}"; do
    if grep -q "\"$port\"" docker-compose.traefik.yml; then
        log_success "✅ Port mapping '$port' configured"
    else
        log_error "❌ Port mapping '$port' not found"
        exit 1
    fi
done

# Test 10: Verify health check configuration
log_info "Checking health check configuration..."

if grep -q "curl.*health" docker-compose.traefik.yml; then
    log_success "✅ Health checks configured with curl"
else
    log_error "❌ Health checks not properly configured"
    exit 1
fi

echo ""
log_success "🎉 All Traefik Gateway Setup Tests Passed!"
echo ""
echo "Phase 4 Implementation Summary:"
echo "==============================="
echo "✅ Traefik configuration with gRPC support"
echo "✅ Dynamic service routing for State, Identity services"  
echo "✅ Docker Compose setup with networking"
echo "✅ Health checks and monitoring integration"
echo "✅ MDP compatibility routing through gateway"
echo "✅ Development scripts and Makefile integration"
echo "✅ Prometheus monitoring configuration"
echo ""
echo "Ready for deployment! Use 'make traefik-dev-start' to start the gateway." 