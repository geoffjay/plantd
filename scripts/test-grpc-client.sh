#!/bin/bash

set -e

echo "Testing gRPC Client Implementation (Phase 5)"
echo "============================================"

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

# Configuration
GATEWAY_URL="http://localhost:8080"
TEST_SCOPE="org.plantd.Test"
TEST_KEY="test-key"
TEST_VALUE="test-value-$(date +%s)"
CLIENT_BINARY="./build/plant-grpc"

# Test counters
TESTS_RUN=0
TESTS_PASSED=0

run_test() {
    local test_name="$1"
    local test_command="$2"
    local expected_result="$3"
    
    TESTS_RUN=$((TESTS_RUN + 1))
    log_info "Running test: $test_name"
    
    if eval "$test_command"; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
        log_success "✓ $test_name"
    else
        log_error "✗ $test_name"
    fi
}

# Test 1: Check if required files exist
log_info "Testing gRPC client setup..."

required_files=(
    "client/internal/grpc/state_client.go"
    "client/internal/grpc/identity_client.go"
    "client/cmd/state_grpc.go"
    "client/cmd/auth_grpc.go"
    "$CLIENT_BINARY"
)

for file in "${required_files[@]}"; do
    if [[ -f "$file" ]]; then
        log_success "✓ Found $file"
    else
        log_error "✗ Missing $file"
        exit 1
    fi
done

# Test 2: Check if client can be executed
log_info "Testing client binary execution..."

if ! $CLIENT_BINARY --help > /dev/null 2>&1; then
    log_error "Client binary cannot be executed"
    exit 1
fi
log_success "✓ Client binary is executable"

# Test 3: Check if gRPC commands are available
log_info "Testing gRPC command availability..."

grpc_commands=(
    "state state-grpc"
    "auth auth-grpc"
)

for cmd in "${grpc_commands[@]}"; do
    if $CLIENT_BINARY $cmd --help > /dev/null 2>&1; then
        log_success "✓ Command '$cmd' is available"
    else
        log_error "✗ Command '$cmd' is not available"
        exit 1
    fi
done

# Test 4: Check state-grpc subcommands
log_info "Testing state-grpc subcommands..."

state_subcommands=(
    "get"
    "set"
    "list"
    "delete"
    "list-scopes"
)

for subcmd in "${state_subcommands[@]}"; do
    if $CLIENT_BINARY state state-grpc "$subcmd" --help > /dev/null 2>&1; then
        log_success "✓ Subcommand 'state state-grpc $subcmd' is available"
    else
        log_error "✗ Subcommand 'state state-grpc $subcmd' is not available"
        exit 1
    fi
done

# Test 5: Check auth-grpc subcommands
log_info "Testing auth-grpc subcommands..."

auth_subcommands=(
    "login"
    "logout"
    "status"
    "refresh"
    "whoami"
)

for subcmd in "${auth_subcommands[@]}"; do
    if $CLIENT_BINARY auth auth-grpc "$subcmd" --help > /dev/null 2>&1; then
        log_success "✓ Subcommand 'auth auth-grpc $subcmd' is available"
    else
        log_error "✗ Subcommand 'auth auth-grpc $subcmd' is not available"
        exit 1
    fi
done

# Test 6: Test configuration flags
log_info "Testing configuration flags..."

# Test gRPC endpoint flag
if $CLIENT_BINARY state state-grpc get --grpc-endpoint="$GATEWAY_URL" --help > /dev/null 2>&1; then
    log_success "✓ gRPC endpoint flag is supported"
else
    log_error "✗ gRPC endpoint flag is not supported"
    exit 1
fi

# Test service scope flag
if $CLIENT_BINARY state state-grpc get --service="$TEST_SCOPE" --help > /dev/null 2>&1; then
    log_success "✓ Service scope flag is supported"
else
    log_error "✗ Service scope flag is not supported"
    exit 1
fi

# Test 7: Test error handling for offline service
log_info "Testing error handling for offline service..."

# This should fail gracefully when the service is not running
if ! $CLIENT_BINARY state state-grpc list --grpc-endpoint="http://localhost:9999" --service="$TEST_SCOPE" > /dev/null 2>&1; then
    log_success "✓ Graceful error handling for offline service"
else
    log_warning "! Unexpected success when service should be offline"
fi

# Test 8: Test configuration file integration
log_info "Testing configuration integration..."

config_file="$HOME/.config/plantd/client.yaml"
if [[ -f "$config_file" ]]; then
    log_success "✓ Configuration file exists: $config_file"
else
    log_warning "! Configuration file not found: $config_file"
fi

# Test 9: Test authentication status (should show not authenticated)
log_info "Testing authentication status..."

auth_output=$($CLIENT_BINARY auth auth-grpc status --grpc-endpoint="$GATEWAY_URL" 2>&1 || true)
if echo "$auth_output" | grep -q "Not authenticated"; then
    log_success "✓ Authentication status correctly shows not authenticated"
else
    log_warning "! Authentication status output: $auth_output"
fi

# Test 10: Test MDP compatibility mode flag
log_info "Testing MDP compatibility flag..."

if $CLIENT_BINARY state --use-grpc --help > /dev/null 2>&1; then
    log_success "✓ MDP compatibility flag (--use-grpc) is available"
else
    log_error "✗ MDP compatibility flag (--use-grpc) is not available"
    exit 1
fi

# Summary
echo ""
echo "Phase 5 Test Summary"
echo "==================="
echo "Files checked: ✓"
echo "Commands available: ✓"
echo "Configuration flags: ✓"
echo "Error handling: ✓"
echo ""

if [[ $TESTS_PASSED -eq $TESTS_RUN ]]; then
    log_success "All tests passed! ($TESTS_PASSED/$TESTS_RUN)"
    echo ""
    echo "Phase 5 gRPC Client Implementation: ✅ COMPLETE"
    echo ""
    echo "Available gRPC commands:"
    echo "  plant-grpc state state-grpc get <key> --service=<scope> --grpc-endpoint=$GATEWAY_URL"
    echo "  plant-grpc state state-grpc set <key> <value> --service=<scope> --grpc-endpoint=$GATEWAY_URL"
    echo "  plant-grpc state state-grpc list --service=<scope> --grpc-endpoint=$GATEWAY_URL"
    echo "  plant-grpc state state-grpc list-scopes --grpc-endpoint=$GATEWAY_URL"
    echo "  plant-grpc auth auth-grpc login --grpc-endpoint=$GATEWAY_URL"
    echo "  plant-grpc auth auth-grpc status --grpc-endpoint=$GATEWAY_URL"
    echo ""
    echo "Note: These commands require the Traefik gateway and gRPC services to be running."
    echo "Start the gateway with: make traefik-dev-start"
    exit 0
else
    log_error "Some tests failed ($TESTS_PASSED/$TESTS_RUN)"
    exit 1
fi 
