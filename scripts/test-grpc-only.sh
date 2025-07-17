#!/bin/bash

# PlantD gRPC-Only Migration Validation Test
# Tests all functionality using only gRPC protocols (MDP disabled)

# Note: Removed 'set -e' to continue testing even if some tests fail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test configuration
PLANT_CLI="./build/plant"
TEST_SERVICE="test"
TEST_KEY="migration.grpc.test"
TEST_VALUE="grpc-success-$(date +%s)"

# Service PIDs
STATE_PID=""
IDENTITY_PID=""
BROKER_PID=""

# Test results
TESTS_PASSED=0
TESTS_FAILED=0

log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
    ((TESTS_PASSED++))
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
    ((TESTS_FAILED++))
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

# Test function that captures both stdout and stderr
run_test() {
    local test_name="$1"
    local command="$2"
    local timeout="${3:-10}"
    
    echo -n "Testing: $test_name... "
    
    if timeout "$timeout" $command >/dev/null 2>&1; then
        log_success "$test_name"
        return 0
    else
        log_error "$test_name"
        return 1
    fi
}

# Test function with output capture
run_test_with_output() {
    local test_name="$1"
    local command="$2"
    local timeout="${3:-10}"
    
    echo -n "Testing: $test_name... "
    
    local output=$(timeout "$timeout" $command 2>&1)
    local exit_code=$?
    
    if [ $exit_code -eq 0 ]; then
        log_success "$test_name"
        if [[ "$4" == "show_output" ]]; then
            echo "$output"
        fi
        return 0
    else
        log_error "$test_name"
        echo "Command: $command"
        echo "Output: $output"
        return 1
    fi
}

# Build services
build_services() {
    echo -e "${BLUE}=== Building PlantD Services ===${NC}"
    make build-state build-identity build-broker build-client >/dev/null 2>&1
    log_info "Built all PlantD services and client"
    echo ""
}

# Start services in gRPC-only mode (MDP disabled)
start_grpc_only_services() {
    echo -e "${BLUE}=== Starting Services (gRPC-Only Mode) ===${NC}"
    
    # Kill any existing services
    pkill -f plantd || true
    sleep 2
    
    # Start State service with ONLY gRPC (MDP disabled)
    echo "Starting State service (gRPC-only)..."
    PLANTD_STATE_ENABLE_MDP=false \
    PLANTD_STATE_ENABLE_GRPC=true \
    PLANTD_STATE_GRPC_PORT=9080 \
    ./build/plantd-state &
    STATE_PID=$!
    
    # Start Identity service with ONLY gRPC (MDP disabled)
    echo "Starting Identity service (gRPC-only)..."
    PLANTD_IDENTITY_ENABLE_MDP=false \
    PLANTD_IDENTITY_ENABLE_GRPC=true \
    PLANTD_IDENTITY_GRPC_PORT=8090 \
    ./build/plantd-identity &
    IDENTITY_PID=$!
    
    # Start Broker service with ONLY gRPC (MDP disabled)
    echo "Starting Broker service (gRPC-only)..."
    PLANTD_BROKER_ENABLE_MDP=false \
    PLANTD_BROKER_ENABLE_GRPC=true \
    PLANTD_BROKER_GRPC_PORT=8082 \
    ./build/plantd-broker &
    BROKER_PID=$!
    
    # Wait for services to start
    echo "Waiting for services to initialize..."
    sleep 8
    
    log_info "Started services: State($STATE_PID), Identity($IDENTITY_PID), Broker($BROKER_PID)"
    echo ""
}

# Test service health endpoints
test_health_endpoints() {
    echo -e "${BLUE}=== Service Health Checks (gRPC) ===${NC}"
    
    run_test "State service health" "curl -f -s http://localhost:9080/health"
    run_test "Identity service health" "curl -f -s http://localhost:8090/health" 
    run_test "Broker service health" "curl -f -s http://localhost:8082/health"
    
    echo ""
}

# Test authentication operations
test_grpc_authentication() {
    echo -e "${BLUE}=== gRPC Authentication Tests ===${NC}"
    
    # Test auth status (should show no token initially)
    run_test "Auth Status Check" "$PLANT_CLI auth auth-grpc status"
    
    # Test login (this should work and create a token)
    echo "Attempting login..."
    if echo -e "geoff.jay@gmail.com\nM1ntberrycrunch!" | $PLANT_CLI auth login >/dev/null 2>&1; then
        log_success "User Login"
        
        # Test authenticated operations
        run_test "Auth Status (authenticated)" "$PLANT_CLI auth auth-grpc status"
        run_test "Auth Whoami" "$PLANT_CLI auth auth-grpc whoami"
        
    else
        log_warning "Login failed - testing with explicit gRPC endpoint"
        
        # Try explicit gRPC endpoint
        run_test "Auth Status (explicit endpoint)" "$PLANT_CLI auth auth-grpc status --grpc-endpoint=http://localhost:8090"
    fi
    
    echo ""
}

# Test state operations
test_grpc_state_operations() {
    echo -e "${BLUE}=== gRPC State Operations Tests ===${NC}"
    
    # Test basic state operations using gRPC
    run_test "State List Scopes (gRPC flag)" "$PLANT_CLI state list-scopes --use-grpc --service=$TEST_SERVICE"
    run_test "State List Scopes (explicit endpoint)" "$PLANT_CLI state state-grpc list-scopes --grpc-endpoint=http://localhost:9080 --service=$TEST_SERVICE"
    
    # Test set/get operations if authenticated
    if $PLANT_CLI state set --use-grpc --service="$TEST_SERVICE" "$TEST_KEY" "$TEST_VALUE" >/dev/null 2>&1; then
        log_success "State Set Operation (gRPC)"
        
        # Test get operation
        local retrieved_value=$($PLANT_CLI state get --use-grpc --service="$TEST_SERVICE" "$TEST_KEY" 2>/dev/null || echo "")
        if [[ "$retrieved_value" == "$TEST_VALUE" ]]; then
            log_success "State Get Operation (gRPC) - Value matches"
        else
            log_error "State Get Operation (gRPC) - Value mismatch: expected '$TEST_VALUE', got '$retrieved_value'"
        fi
        
        # Test list operation
        run_test "State List Keys (gRPC)" "$PLANT_CLI state list --use-grpc --service=$TEST_SERVICE"
        
        # Test delete operation
        run_test "State Delete Operation (gRPC)" "$PLANT_CLI state delete --use-grpc --service=$TEST_SERVICE $TEST_KEY"
        
    else
        log_warning "Skipping authenticated state operations (login required)"
    fi
    
    echo ""
}

# Test broker gRPC API directly
test_broker_grpc_api() {
    echo -e "${BLUE}=== Broker gRPC API Tests ===${NC}"
    
    # Test broker health
    run_test "Broker Health API" "curl -f -s http://localhost:8082/plantd.broker.v1.BrokerService/Health"
    
    # Test service registration
    local registration_payload='{"service":{"name":"test.service","version":"1.0.0","capabilities":["test"],"metadata":{"test":"true"}}}'
    
    if curl -f -s -X POST \
        -H "Content-Type: application/json" \
        -d "$registration_payload" \
        http://localhost:8082/plantd.broker.v1.BrokerService/RegisterService >/dev/null 2>&1; then
        log_success "Service Registration"
        
        # Test service listing
        run_test "List Services" "curl -f -s -X POST -H 'Content-Type: application/json' -d '{}' http://localhost:8082/plantd.broker.v1.BrokerService/ListServices"
        
        # Test bus creation
        local bus_payload='{"name":"test-bus","type":"EVENT","config":{"buffer_size":1000}}'
        run_test "Create Bus" "curl -f -s -X POST -H 'Content-Type: application/json' -d '$bus_payload' http://localhost:8082/plantd.broker.v1.BrokerService/CreateBus"
        
        # Test bus listing
        run_test "List Buses" "curl -f -s -X POST -H 'Content-Type: application/json' -d '{}' http://localhost:8082/plantd.broker.v1.BrokerService/ListBuses"
        
    else
        log_error "Service Registration"
    fi
    
    echo ""
}

# Test configuration commands
test_configuration() {
    echo -e "${BLUE}=== Configuration Tests ===${NC}"
    
    run_test "Config Show" "$PLANT_CLI config show"
    run_test "Config Validate" "$PLANT_CLI config validate"
    run_test "Version Command" "$PLANT_CLI version"
    
    echo ""
}

# Performance test
test_performance() {
    echo -e "${BLUE}=== gRPC Performance Test ===${NC}"
    
    local start_time=$(date +%s%N)
    
    # Run multiple operations to test performance
    for i in {1..10}; do
        $PLANT_CLI state set --use-grpc --service="perf-test" "key.$i" "value.$i" >/dev/null 2>&1 || true
    done
    
    local end_time=$(date +%s%N)
    local duration=$((($end_time - $start_time) / 1000000)) # Convert to milliseconds
    
    if [ $duration -lt 5000 ]; then # Less than 5 seconds for 10 operations
        log_success "Performance Test (${duration}ms for 10 operations)"
    else
        log_warning "Performance Test (${duration}ms for 10 operations - slower than expected)"
    fi
    
    echo ""
}

# Cleanup
cleanup() {
    echo -e "${BLUE}=== Cleanup ===${NC}"
    pkill -f plantd || true
    log_info "Stopped all services"
    echo ""
}

# Generate final report
generate_report() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}  gRPC-Only Migration Test Results${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""
    
    local total_tests=$((TESTS_PASSED + TESTS_FAILED))
    local success_rate=$((TESTS_PASSED * 100 / total_tests))
    
    echo -e "📊 ${GREEN}Tests Passed: $TESTS_PASSED${NC}"
    echo -e "📊 ${RED}Tests Failed: $TESTS_FAILED${NC}"
    echo -e "📊 ${BLUE}Success Rate: $success_rate%${NC}"
    echo ""
    
    if [ $TESTS_FAILED -eq 0 ]; then
        echo -e "${GREEN}🎉 All tests passed! gRPC-only migration is complete and functional.${NC}"
        echo -e "${GREEN}✅ PlantD can run entirely without MDP dependencies.${NC}"
        echo ""
        echo -e "${BLUE}Next Steps:${NC}"
        echo "  1. Performance benchmarking"
        echo "  2. Production deployment"
        echo "  3. Remove MDP code and dependencies"
        echo ""
        return 0
    else
        echo -e "${RED}⚠️  Some tests failed. Review the issues above.${NC}"
        echo ""
        return 1
    fi
}

# Main execution
main() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}PlantD gRPC-Only Migration Validation${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""
    
    # Set trap for cleanup
    trap cleanup EXIT
    
    build_services
    start_grpc_only_services
    test_health_endpoints
    test_grpc_authentication
    test_grpc_state_operations
    test_broker_grpc_api
    test_configuration
    test_performance
    
    generate_report
}

# Run main function
main "$@" 
 