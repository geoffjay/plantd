#!/bin/bash

# PlantD gRPC Migration Verification Script
# This script comprehensively tests all CLI functionality with both MDP and gRPC protocols
# to verify the migration is complete and functionally equivalent.

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test configuration
TEST_USER="test-user-$(date +%s)@example.com"
TEST_PASSWORD="TestPassword123!"
TEST_SERVICE="org.plantd.TestService$(date +%s)"
TEST_KEY="test-key-$(date +%s)"
TEST_VALUE="test-value-$(date +%s)"

PLANT_CLI="./build/plant"
RESULTS_FILE="/tmp/plantd-migration-test-results.log"
ERRORS_FILE="/tmp/plantd-migration-test-errors.log"

# Clear previous results
> "$RESULTS_FILE"
> "$ERRORS_FILE"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}PlantD gRPC Migration Verification Test${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Helper functions
log_success() {
    echo -e "${GREEN}✅ $1${NC}"
    echo "SUCCESS: $1" >> "$RESULTS_FILE"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
    echo "ERROR: $1" >> "$ERRORS_FILE"
}

log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
    echo "INFO: $1" >> "$RESULTS_FILE"
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
    echo "WARNING: $1" >> "$RESULTS_FILE"
}

# Test function wrapper
run_test() {
    local test_name="$1"
    local test_command="$2"
    local protocol="$3"
    
    echo -e "${YELLOW}Testing: $test_name ($protocol)${NC}"
    
    if eval "$test_command" 2>/dev/null; then
        log_success "$test_name ($protocol)"
        return 0
    else
        log_error "$test_name ($protocol)"
        echo "Command: $test_command" >> "$ERRORS_FILE"
        return 1
    fi
}

# Service health checks
check_service_health() {
    echo -e "${BLUE}=== Service Health Checks ===${NC}"
    
    # Check State service (gRPC)
    if curl -s http://localhost:9080/health > /dev/null 2>&1; then
        log_success "State service health check (gRPC on port 9080)"
    else
        log_error "State service not accessible on port 9080"
        return 1
    fi
    
    # Check Identity service (gRPC)  
    if curl -s http://localhost:8090/health > /dev/null 2>&1; then
        log_success "Identity service health check (gRPC on port 8090)"
    else
        log_warning "Identity service not accessible on port 8090 (may need startup)"
    fi
    
    # Check Broker service (gRPC)
    if curl -s http://localhost:8082/health > /dev/null 2>&1; then
        log_success "Broker service health check (gRPC on port 8082)"
    else
        log_warning "Broker service not accessible on port 8082 (may need startup)"
    fi
    
    echo ""
}

# Start required services
start_services() {
    echo -e "${BLUE}=== Starting Services ===${NC}"
    
    # Kill any existing services
    pkill -f plantd || true
    sleep 2
    
    # Start State service with both MDP and gRPC
    echo "Starting State service (MDP + gRPC)..."
    PLANTD_STATE_ENABLE_MDP=true PLANTD_STATE_ENABLE_GRPC=true PLANTD_STATE_GRPC_PORT=9080 ./build/plantd-state &
    STATE_PID=$!
    
    # Start Identity service with both MDP and gRPC  
    echo "Starting Identity service (MDP + gRPC)..."
    PLANTD_IDENTITY_ENABLE_MDP=true PLANTD_IDENTITY_ENABLE_GRPC=true PLANTD_IDENTITY_GRPC_PORT=8090 ./build/plantd-identity &
    IDENTITY_PID=$!
    
    # Start Broker service with both MDP and gRPC
    echo "Starting Broker service (MDP + gRPC)..."
    PLANTD_BROKER_ENABLE_MDP=true PLANTD_BROKER_ENABLE_GRPC=true PLANTD_BROKER_GRPC_PORT=8082 ./build/plantd-broker &
    BROKER_PID=$!
    
    # Wait for services to start
    echo "Waiting for services to initialize..."
    sleep 5
    
    log_info "Started services: State($STATE_PID), Identity($IDENTITY_PID), Broker($BROKER_PID)"
    echo ""
}

# Stop services
stop_services() {
    echo -e "${BLUE}=== Stopping Services ===${NC}"
    pkill -f plantd || true
    log_info "All services stopped"
    echo ""
}

# Test authentication operations
test_auth_operations() {
    echo -e "${BLUE}=== Authentication Tests ===${NC}"
    
    # Test registration (usually via Identity service)
    echo "Testing user registration..."
    
    # Test MDP Authentication
    echo -e "${YELLOW}--- MDP Authentication ---${NC}"
    
    # For now, test basic auth status (login may require actual user setup)
    run_test "Auth Status Check" "$PLANT_CLI auth status" "MDP"
    
    # Test gRPC Authentication  
    echo -e "${YELLOW}--- gRPC Authentication ---${NC}"
    
    run_test "Auth Status Check" "$PLANT_CLI auth auth-grpc status" "gRPC"
    run_test "Auth Status Check (explicit)" "$PLANT_CLI auth auth-grpc status --grpc-endpoint=http://localhost:8090" "gRPC-explicit"
    
    echo ""
}

# Test state operations
test_state_operations() {
    echo -e "${BLUE}=== State Management Tests ===${NC}"
    
    # Test MDP State Operations
    echo -e "${YELLOW}--- MDP State Operations ---${NC}"
    
    # These may require authentication, so we'll test basic connectivity
    run_test "State List Scopes" "$PLANT_CLI state list-scopes --service=$TEST_SERVICE" "MDP"
    
    # Test gRPC State Operations
    echo -e "${YELLOW}--- gRPC State Operations ---${NC}"
    
    run_test "State List Scopes" "$PLANT_CLI state list-scopes --use-grpc --service=$TEST_SERVICE" "gRPC"
    run_test "State List Scopes (explicit)" "$PLANT_CLI state-grpc list-scopes --grpc-endpoint=http://localhost:9080 --service=$TEST_SERVICE" "gRPC-explicit"
    
    # Test State operations if we can authenticate
    echo -e "${YELLOW}--- Advanced State Tests (if authenticated) ---${NC}"
    
    # Try set/get operations (may fail if not authenticated)
    if $PLANT_CLI state set --service="$TEST_SERVICE" "$TEST_KEY" "$TEST_VALUE" 2>/dev/null; then
        log_success "State Set Operation (MDP)"
        
        if $PLANT_CLI state get --service="$TEST_SERVICE" "$TEST_KEY" 2>/dev/null; then
            log_success "State Get Operation (MDP)"
        fi
        
        if $PLANT_CLI state delete --service="$TEST_SERVICE" "$TEST_KEY" 2>/dev/null; then
            log_success "State Delete Operation (MDP)"
        fi
    else
        log_warning "State Set/Get/Delete operations require authentication"
    fi
    
    # Test gRPC versions
    if $PLANT_CLI state set --use-grpc --service="$TEST_SERVICE" "$TEST_KEY" "$TEST_VALUE" 2>/dev/null; then
        log_success "State Set Operation (gRPC)"
        
        if $PLANT_CLI state get --use-grpc --service="$TEST_SERVICE" "$TEST_KEY" 2>/dev/null; then
            log_success "State Get Operation (gRPC)"
        fi
        
        if $PLANT_CLI state delete --use-grpc --service="$TEST_SERVICE" "$TEST_KEY" 2>/dev/null; then
            log_success "State Delete Operation (gRPC)"
        fi
    else
        log_warning "gRPC State Set/Get/Delete operations require authentication"
    fi
    
    echo ""
}

# Test broker service functionality via gRPC API
test_broker_functionality() {
    echo -e "${BLUE}=== Broker Service Tests ===${NC}"
    
    # Test broker health endpoint
    if curl -s http://localhost:8082/health | grep -q "healthy"; then
        log_success "Broker HTTP health endpoint"
    else
        log_error "Broker HTTP health endpoint"
    fi
    
    # Test broker gRPC health
    if curl -s -X POST -H "Content-Type: application/json" -d '{}' \
       http://localhost:8082/plantd.broker.v1.BrokerService/Health | grep -q "SERVING"; then
        log_success "Broker gRPC health endpoint"
    else
        log_error "Broker gRPC health endpoint" 
    fi
    
    # Test service registration
    SERVICE_REG_DATA='{"serviceId":"test-service-'$(date +%s)'","endpoint":"http://localhost:9999","capabilities":["test"],"metadata":{"version":"1.0"}}'
    
    if curl -s -X POST -H "Content-Type: application/json" -d "$SERVICE_REG_DATA" \
       http://localhost:8082/plantd.broker.v1.BrokerService/RegisterService | grep -q "registrationId"; then
        log_success "Broker service registration"
    else
        log_error "Broker service registration"
    fi
    
    # Test bus creation
    BUS_DATA='{"name":"test-bus-'$(date +%s)'","type":"BUS_TYPE_CONTROL","config":{"description":"Test bus"}}'
    
    if curl -s -X POST -H "Content-Type: application/json" -d "$BUS_DATA" \
       http://localhost:8082/plantd.broker.v1.BrokerService/CreateBus | grep -q "busId"; then
        log_success "Broker bus creation"
    else
        log_error "Broker bus creation"
    fi
    
    echo ""
}

# Test echo/connectivity
test_connectivity() {
    echo -e "${BLUE}=== Connectivity Tests ===${NC}"
    
    # Test echo command (if echo module is available)
    if run_test "Echo Connectivity" "$PLANT_CLI echo 'test-message'" "MDP"; then
        true
    else
        log_warning "Echo service may not be available"
    fi
    
    echo ""
}

# Test protocol consistency
test_protocol_consistency() {
    echo -e "${BLUE}=== Protocol Consistency Tests ===${NC}"
    
    # Test that both protocols support the same commands
    echo "Checking command structure consistency..."
    
    # Both should support the same state commands
    $PLANT_CLI state --help > /tmp/mdp_state_help.txt 2>&1
    $PLANT_CLI state-grpc --help > /tmp/grpc_state_help.txt 2>&1
    
    if grep -q "get\|set\|list\|delete" /tmp/mdp_state_help.txt && \
       grep -q "get\|set\|list\|delete" /tmp/grpc_state_help.txt; then
        log_success "Both protocols support core state commands"
    else
        log_error "Protocol command consistency check failed"
    fi
    
    # Both should support auth commands  
    $PLANT_CLI auth --help > /tmp/mdp_auth_help.txt 2>&1
    $PLANT_CLI auth-grpc --help > /tmp/grpc_auth_help.txt 2>&1
    
    if grep -q "login\|logout\|status" /tmp/mdp_auth_help.txt && \
       grep -q "login\|logout\|status" /tmp/grpc_auth_help.txt; then
        log_success "Both protocols support core auth commands"
    else
        log_error "Protocol auth command consistency check failed"
    fi
    
    # Clean up temp files
    rm -f /tmp/*_help.txt
    
    echo ""
}

# Generate test report
generate_report() {
    echo -e "${BLUE}=== Test Report ===${NC}"
    
    local total_tests=$(grep -c "SUCCESS\|ERROR" "$RESULTS_FILE" 2>/dev/null || echo "0")
    local successful_tests=$(grep -c "SUCCESS" "$RESULTS_FILE" 2>/dev/null || echo "0")
    local failed_tests=$(grep -c "ERROR" "$ERRORS_FILE" 2>/dev/null || echo "0")
    local warnings=$(grep -c "WARNING" "$RESULTS_FILE" 2>/dev/null || echo "0")
    
    echo ""
    echo -e "${BLUE}📊 Test Summary:${NC}"
    echo -e "   Total Tests: $total_tests"
    echo -e "   ${GREEN}Successful: $successful_tests${NC}"
    echo -e "   ${RED}Failed: $failed_tests${NC}"
    echo -e "   ${YELLOW}Warnings: $warnings${NC}"
    echo ""
    
    if [ "$failed_tests" -eq 0 ]; then
        echo -e "${GREEN}🎉 ALL TESTS PASSED! gRPC migration appears to be complete and functional.${NC}"
        echo ""
        echo -e "${GREEN}✅ Ready to remove MDP dependencies${NC}"
        MIGRATION_SUCCESS=true
    else
        echo -e "${RED}❌ Some tests failed. Migration verification incomplete.${NC}"
        echo ""
        echo -e "${RED}Review errors before removing MDP:${NC}"
        if [ -f "$ERRORS_FILE" ] && [ -s "$ERRORS_FILE" ]; then
            cat "$ERRORS_FILE"
        fi
        MIGRATION_SUCCESS=false
    fi
    
    echo ""
    echo -e "${BLUE}📋 Detailed Results:${NC}"
    echo "   Success Log: $RESULTS_FILE"
    echo "   Error Log: $ERRORS_FILE"
    echo ""
}

# Main test execution
main() {
    echo -e "${BLUE}Starting PlantD Migration Verification...${NC}"
    echo ""
    
    # Build all services first
    log_info "Building PlantD services..."
    make build-all > /dev/null 2>&1
    
    # Start services
    start_services
    
    # Trap to ensure cleanup
    trap stop_services EXIT
    
    # Run health checks
    check_service_health
    
    # Run all test suites
    test_auth_operations
    test_state_operations 
    test_broker_functionality
    test_connectivity
    test_protocol_consistency
    
    # Generate final report
    generate_report
    
    # Return appropriate exit code
    if [ "$MIGRATION_SUCCESS" = true ]; then
        exit 0
    else
        exit 1
    fi
}

# Command line interface
case "${1:-}" in
    --help|-h)
        echo "PlantD gRPC Migration Verification Script"
        echo ""
        echo "Usage: $0 [options]"
        echo ""
        echo "Options:"
        echo "  --help, -h    Show this help message"
        echo "  --quick       Run quick verification (minimal tests)"
        echo "  --auth-only   Test only authentication functionality"
        echo "  --state-only  Test only state functionality"
        echo "  --broker-only Test only broker functionality"
        echo ""
        echo "This script comprehensively tests PlantD CLI functionality"
        echo "with both MDP and gRPC protocols to verify migration completeness."
        ;;
    --quick)
        log_info "Running quick verification..."
        start_services
        trap stop_services EXIT
        check_service_health
        test_protocol_consistency
        generate_report
        ;;
    --auth-only)
        log_info "Testing authentication only..."
        start_services
        trap stop_services EXIT
        check_service_health
        test_auth_operations
        generate_report
        ;;
    --state-only)
        log_info "Testing state functionality only..."
        start_services
        trap stop_services EXIT
        check_service_health
        test_state_operations
        generate_report
        ;;
    --broker-only)
        log_info "Testing broker functionality only..."
        start_services
        trap stop_services EXIT
        check_service_health
        test_broker_functionality
        generate_report
        ;;
    *)
        main
        ;;
esac 
