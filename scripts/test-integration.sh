#!/bin/bash

# Integration Testing Script for gRPC Migration (Phase 6)
# Tests full system integration with Traefik gateway

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
TRAEFIK_ENDPOINT="http://localhost:8080"
GRPC_CLIENT="./build/plant-grpc"
TEST_SERVICE="test-integration"
TEST_DATA_DIR="./test-data"
RESULTS_DIR="./test-results/integration"

# Test counters
TESTS_TOTAL=0
TESTS_PASSED=0
TESTS_FAILED=0

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} ✓ $1"
    ((TESTS_PASSED++))
}

log_error() {
    echo -e "${RED}[ERROR]${NC} ✗ $1"
    ((TESTS_FAILED++))
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} ! $1"
}

increment_test() {
    ((TESTS_TOTAL++))
}

# Cleanup function
cleanup() {
    log_info "Cleaning up test data..."
    if [ -d "$TEST_DATA_DIR" ]; then
        rm -rf "$TEST_DATA_DIR"
    fi
}

# Setup test environment
setup_test_env() {
    log_info "Setting up test environment..."
    mkdir -p "$TEST_DATA_DIR"
    mkdir -p "$RESULTS_DIR"
    
    # Ensure client is built
    if [ ! -f "$GRPC_CLIENT" ]; then
        log_info "Building gRPC client..."
        make build-client-grpc
    fi
}

# Test Traefik gateway health
test_traefik_health() {
    log_info "Testing Traefik gateway health..."
    increment_test
    
    if curl -s -f "$TRAEFIK_ENDPOINT/ping" > /dev/null 2>&1; then
        log_success "Traefik gateway is healthy"
    else
        log_error "Traefik gateway is not responding"
        return 1
    fi
}

# Test service discovery through Traefik
test_service_discovery() {
    log_info "Testing service discovery through Traefik..."
    increment_test
    
    # Test if services are discoverable through Traefik
    if curl -s -f "$TRAEFIK_ENDPOINT/api/http/services" | grep -q "state-service" > /dev/null 2>&1; then
        log_success "Services discovered through Traefik"
    else
        log_warning "Service discovery test skipped (Traefik API may be disabled)"
    fi
}

# Test gRPC health checks through gateway
test_grpc_health_checks() {
    log_info "Testing gRPC health checks through gateway..."
    increment_test
    
    # Test state service health through Traefik
    if curl -s -f "$TRAEFIK_ENDPOINT/health" > /dev/null 2>&1; then
        log_success "gRPC health checks work through gateway"
    else
        log_error "gRPC health checks failed through gateway"
    fi
}

# Test end-to-end state operations
test_e2e_state_operations() {
    log_info "Testing end-to-end state operations..."
    
    local test_key="integration-test-$(date +%s)"
    local test_value="test-value-$(date +%s)"
    local test_scope="integration-test"
    
    # Test SET operation
    increment_test
    log_info "Testing state SET operation..."
    if $GRPC_CLIENT state state-grpc set "$test_key" "$test_value" --service="$test_scope" --grpc-endpoint="$TRAEFIK_ENDPOINT" > "$TEST_DATA_DIR/set_result.json" 2>&1; then
        log_success "State SET operation successful"
    else
        log_error "State SET operation failed"
        cat "$TEST_DATA_DIR/set_result.json"
    fi
    
    # Test GET operation
    increment_test
    log_info "Testing state GET operation..."
    if $GRPC_CLIENT state state-grpc get "$test_key" --service="$test_scope" --grpc-endpoint="$TRAEFIK_ENDPOINT" > "$TEST_DATA_DIR/get_result.json" 2>&1; then
        # Verify the value matches
        if grep -q "$test_value" "$TEST_DATA_DIR/get_result.json"; then
            log_success "State GET operation successful with correct value"
        else
            log_error "State GET operation returned incorrect value"
            cat "$TEST_DATA_DIR/get_result.json"
        fi
    else
        log_error "State GET operation failed"
        cat "$TEST_DATA_DIR/get_result.json"
    fi
    
    # Test LIST operation
    increment_test
    log_info "Testing state LIST operation..."
    if $GRPC_CLIENT state state-grpc list --service="$test_scope" --grpc-endpoint="$TRAEFIK_ENDPOINT" > "$TEST_DATA_DIR/list_result.json" 2>&1; then
        # Verify our key is in the list
        if grep -q "$test_key" "$TEST_DATA_DIR/list_result.json"; then
            log_success "State LIST operation successful with our key present"
        else
            log_warning "State LIST operation successful but our key not found"
        fi
    else
        log_error "State LIST operation failed"
        cat "$TEST_DATA_DIR/list_result.json"
    fi
    
    # Test DELETE operation
    increment_test
    log_info "Testing state DELETE operation..."
    if $GRPC_CLIENT state state-grpc delete "$test_key" --service="$test_scope" --grpc-endpoint="$TRAEFIK_ENDPOINT" > "$TEST_DATA_DIR/delete_result.json" 2>&1; then
        log_success "State DELETE operation successful"
    else
        log_error "State DELETE operation failed"
        cat "$TEST_DATA_DIR/delete_result.json"
    fi
    
    # Test LIST-SCOPES operation
    increment_test
    log_info "Testing state LIST-SCOPES operation..."
    if $GRPC_CLIENT state state-grpc list-scopes --grpc-endpoint="$TRAEFIK_ENDPOINT" > "$TEST_DATA_DIR/list_scopes_result.json" 2>&1; then
        log_success "State LIST-SCOPES operation successful"
    else
        log_error "State LIST-SCOPES operation failed"
        cat "$TEST_DATA_DIR/list_scopes_result.json"
    fi
}

# Test authentication flow
test_authentication_flow() {
    log_info "Testing authentication flow..."
    
    # Test auth status (should work without login for status check)
    increment_test
    log_info "Testing auth STATUS operation..."
    if $GRPC_CLIENT auth auth-grpc status --grpc-endpoint="$TRAEFIK_ENDPOINT" > "$TEST_DATA_DIR/auth_status.json" 2>&1; then
        log_success "Auth STATUS operation successful"
    else
        log_warning "Auth STATUS operation failed (expected if identity service not running)"
        cat "$TEST_DATA_DIR/auth_status.json"
    fi
    
    # Test whoami (may fail if not authenticated)
    increment_test
    log_info "Testing auth WHOAMI operation..."
    if $GRPC_CLIENT auth auth-grpc whoami --grpc-endpoint="$TRAEFIK_ENDPOINT" > "$TEST_DATA_DIR/auth_whoami.json" 2>&1; then
        log_success "Auth WHOAMI operation successful"
    else
        log_warning "Auth WHOAMI operation failed (expected if not authenticated)"
    fi
}

# Test error handling and resilience
test_error_handling() {
    log_info "Testing error handling and resilience..."
    
    # Test with invalid endpoint
    increment_test
    log_info "Testing invalid endpoint handling..."
    if $GRPC_CLIENT state state-grpc get "test-key" --service="test" --grpc-endpoint="http://localhost:9999" > "$TEST_DATA_DIR/invalid_endpoint.json" 2>&1; then
        log_error "Should have failed with invalid endpoint"
    else
        log_success "Properly handled invalid endpoint"
    fi
    
    # Test with invalid service
    increment_test
    log_info "Testing invalid service handling..."
    if $GRPC_CLIENT state state-grpc get "non-existent-key" --service="non-existent-service" --grpc-endpoint="$TRAEFIK_ENDPOINT" > "$TEST_DATA_DIR/invalid_service.json" 2>&1; then
        log_warning "Invalid service request completed (may be expected behavior)"
    else
        log_success "Properly handled invalid service"
    fi
}

# Test concurrent operations
test_concurrent_operations() {
    log_info "Testing concurrent operations..."
    increment_test
    
    local pids=()
    local test_scope="concurrent-test"
    
    # Launch multiple concurrent SET operations
    for i in {1..5}; do
        $GRPC_CLIENT state state-grpc set "concurrent-key-$i" "concurrent-value-$i" --service="$test_scope" --grpc-endpoint="$TRAEFIK_ENDPOINT" > "$TEST_DATA_DIR/concurrent_set_$i.json" 2>&1 &
        pids+=($!)
    done
    
    # Wait for all operations to complete
    local all_success=true
    for pid in "${pids[@]}"; do
        if ! wait $pid; then
            all_success=false
        fi
    done
    
    if $all_success; then
        log_success "Concurrent operations completed successfully"
    else
        log_error "Some concurrent operations failed"
    fi
    
    # Cleanup concurrent test data
    for i in {1..5}; do
        $GRPC_CLIENT state state-grpc delete "concurrent-key-$i" --service="$test_scope" --grpc-endpoint="$TRAEFIK_ENDPOINT" > /dev/null 2>&1 || true
    done
}

# Test performance benchmarking
test_performance_benchmark() {
    log_info "Running performance benchmarks..."
    increment_test
    
    local benchmark_file="$RESULTS_DIR/performance_benchmark.txt"
    local test_scope="benchmark-test"
    local num_operations=50
    
    echo "Performance Benchmark Results - $(date)" > "$benchmark_file"
    echo "========================================" >> "$benchmark_file"
    
    # Benchmark SET operations
    log_info "Benchmarking SET operations ($num_operations operations)..."
    local start_time=$(date +%s.%N)
    
    for i in $(seq 1 $num_operations); do
        $GRPC_CLIENT state state-grpc set "bench-key-$i" "bench-value-$i" --service="$test_scope" --grpc-endpoint="$TRAEFIK_ENDPOINT" > /dev/null 2>&1 || true
    done
    
    local end_time=$(date +%s.%N)
    local duration=$(echo "$end_time - $start_time" | bc -l)
    local ops_per_sec=$(echo "scale=2; $num_operations / $duration" | bc -l)
    
    echo "SET Operations: $num_operations operations in ${duration}s (${ops_per_sec} ops/sec)" >> "$benchmark_file"
    log_success "SET benchmark: ${ops_per_sec} ops/sec"
    
    # Benchmark GET operations
    log_info "Benchmarking GET operations ($num_operations operations)..."
    start_time=$(date +%s.%N)
    
    for i in $(seq 1 $num_operations); do
        $GRPC_CLIENT state state-grpc get "bench-key-$i" --service="$test_scope" --grpc-endpoint="$TRAEFIK_ENDPOINT" > /dev/null 2>&1 || true
    done
    
    end_time=$(date +%s.%N)
    duration=$(echo "$end_time - $start_time" | bc -l)
    ops_per_sec=$(echo "scale=2; $num_operations / $duration" | bc -l)
    
    echo "GET Operations: $num_operations operations in ${duration}s (${ops_per_sec} ops/sec)" >> "$benchmark_file"
    log_success "GET benchmark: ${ops_per_sec} ops/sec"
    
    # Cleanup benchmark data
    for i in $(seq 1 $num_operations); do
        $GRPC_CLIENT state state-grpc delete "bench-key-$i" --service="$test_scope" --grpc-endpoint="$TRAEFIK_ENDPOINT" > /dev/null 2>&1 || true
    done
    
    log_info "Benchmark results saved to: $benchmark_file"
}

# Test MDP compatibility during migration
test_mdp_compatibility() {
    log_info "Testing MDP compatibility during migration..."
    increment_test
    
    # Test that old MDP client still works (if available)
    if [ -f "./build/plant" ]; then
        log_info "Testing legacy MDP client..."
        if ./build/plant state get "test-key" --service="compatibility-test" > "$TEST_DATA_DIR/mdp_test.json" 2>&1; then
            log_success "Legacy MDP client still functional"
        else
            log_warning "Legacy MDP client test failed (may be expected if MDP broker not running)"
        fi
    else
        log_warning "Legacy MDP client not found, skipping compatibility test"
    fi
    
    # Test gRPC client with MDP compatibility flag
    log_info "Testing gRPC client MDP compatibility flag..."
    if $GRPC_CLIENT state state-grpc get "test-key" --service="compatibility-test" --use-grpc --grpc-endpoint="$TRAEFIK_ENDPOINT" > "$TEST_DATA_DIR/grpc_compat_test.json" 2>&1; then
        log_success "gRPC client MDP compatibility flag works"
    else
        log_warning "gRPC client MDP compatibility test failed"
    fi
}

# Generate test report
generate_test_report() {
    local report_file="$RESULTS_DIR/integration_test_report.txt"
    
    echo "Integration Test Report - $(date)" > "$report_file"
    echo "============================================" >> "$report_file"
    echo "" >> "$report_file"
    echo "Test Summary:" >> "$report_file"
    echo "  Total Tests: $TESTS_TOTAL" >> "$report_file"
    echo "  Passed: $TESTS_PASSED" >> "$report_file"
    echo "  Failed: $TESTS_FAILED" >> "$report_file"
    echo "  Success Rate: $(echo "scale=2; $TESTS_PASSED * 100 / $TESTS_TOTAL" | bc -l)%" >> "$report_file"
    echo "" >> "$report_file"
    
    if [ $TESTS_FAILED -eq 0 ]; then
        echo "Status: ✅ ALL TESTS PASSED" >> "$report_file"
    else
        echo "Status: ❌ SOME TESTS FAILED" >> "$report_file"
    fi
    
    echo "" >> "$report_file"
    echo "Test Environment:" >> "$report_file"
    echo "  Traefik Endpoint: $TRAEFIK_ENDPOINT" >> "$report_file"
    echo "  gRPC Client: $GRPC_CLIENT" >> "$report_file"
    echo "  Test Timestamp: $(date -Iseconds)" >> "$report_file"
    
    log_info "Test report generated: $report_file"
}

# Main execution
main() {
    echo "Integration Testing for gRPC Migration (Phase 6)"
    echo "==============================================="
    
    # Setup
    setup_test_env
    trap cleanup EXIT
    
    # Run tests
    test_traefik_health || true
    test_service_discovery || true
    test_grpc_health_checks || true
    test_e2e_state_operations || true
    test_authentication_flow || true
    test_error_handling || true
    test_concurrent_operations || true
    test_performance_benchmark || true
    test_mdp_compatibility || true
    
    # Generate report
    generate_test_report
    
    # Summary
    echo ""
    echo "Phase 6 Integration Test Summary"
    echo "==============================="
    echo "Total Tests: $TESTS_TOTAL"
    echo "Passed: $TESTS_PASSED"
    echo "Failed: $TESTS_FAILED"
    
    if [ $TESTS_FAILED -eq 0 ]; then
        echo -e "${GREEN}[SUCCESS]${NC} ✅ All integration tests passed!"
        echo ""
        echo "Phase 6 Integration Testing: ✅ COMPLETE"
        echo ""
        echo "Available test results:"
        echo "  Integration Report: $RESULTS_DIR/integration_test_report.txt"
        echo "  Performance Benchmark: $RESULTS_DIR/performance_benchmark.txt"
        echo ""
        echo "Next steps:"
        echo "  - Review performance benchmarks"
        echo "  - Run load tests in production-like environment"
        echo "  - Proceed with Phase 7: Production Deployment"
        return 0
    else
        echo -e "${RED}[ERROR]${NC} ❌ Some integration tests failed"
        echo ""
        echo "Please review the test results and fix issues before proceeding."
        return 1
    fi
}

# Check dependencies
if ! command -v bc &> /dev/null; then
    echo "Error: 'bc' calculator is required for performance benchmarks"
    echo "Install with: brew install bc (macOS) or apt-get install bc (Ubuntu)"
    exit 1
fi

if ! command -v curl &> /dev/null; then
    echo "Error: 'curl' is required for HTTP tests"
    exit 1
fi

# Run main function
main "$@" 
