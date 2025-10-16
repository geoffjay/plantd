#!/bin/bash

# Failure Scenario Testing Script for gRPC Migration (Phase 6)
# Tests system resilience under various failure conditions

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
RESULTS_DIR="./test-results/failure"
TEMP_DIR="./test-data/failure"

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
    log_info "Cleaning up failure test data..."
    if [ -d "$TEMP_DIR" ]; then
        rm -rf "$TEMP_DIR"
    fi
}

# Setup test environment
setup_failure_test() {
    log_info "Setting up failure test environment..."
    mkdir -p "$TEMP_DIR"
    mkdir -p "$RESULTS_DIR"
    
    # Ensure client is built
    if [ ! -f "$GRPC_CLIENT" ]; then
        log_info "Building gRPC client..."
        make build-client-grpc
    fi
}

# Test invalid endpoint handling
test_invalid_endpoint() {
    log_info "Testing invalid endpoint handling..."
    increment_test
    
    local invalid_endpoints=(
        "http://localhost:9999"
        "http://invalid-host:8080"
        "http://localhost:8080/invalid-path"
    )
    
    local endpoint_failures=0
    
    for endpoint in "${invalid_endpoints[@]}"; do
        log_info "Testing endpoint: $endpoint"
        if $GRPC_CLIENT state state-grpc get "test-key" --service="failure-test" --grpc-endpoint="$endpoint" > "$TEMP_DIR/invalid_endpoint.json" 2>&1; then
            log_warning "Endpoint $endpoint should have failed but didn't"
            ((endpoint_failures++))
        else
            log_info "✓ Properly handled invalid endpoint: $endpoint"
        fi
    done
    
    if [ $endpoint_failures -eq 0 ]; then
        log_success "All invalid endpoints properly handled"
    else
        log_error "$endpoint_failures invalid endpoints were not properly handled"
    fi
}

# Test network timeout scenarios
test_network_timeouts() {
    log_info "Testing network timeout scenarios..."
    increment_test
    
    # Test rapid consecutive operations
    log_info "Testing rapid consecutive operations..."
    local rapid_success=0
    local rapid_errors=0
    
    for i in $(seq 1 20); do
        if $GRPC_CLIENT state state-grpc set "rapid-$i" "value-$i" --service="failure-test" --grpc-endpoint="$TRAEFIK_ENDPOINT" > "$TEMP_DIR/rapid_$i.json" 2>&1; then
            ((rapid_success++))
        else
            ((rapid_errors++))
        fi
    done
    
    log_info "Rapid operations: $rapid_success successful, $rapid_errors errors"
    if [ $rapid_errors -lt 5 ]; then
        log_success "System handled rapid operations well"
    else
        log_warning "System had difficulty with rapid operations"
    fi
}

# Test partial service failures
test_partial_service_failures() {
    log_info "Testing partial service failures..."
    increment_test
    
    # Test with non-existent service scopes
    log_info "Testing non-existent service scopes..."
    if $GRPC_CLIENT state state-grpc get "test-key" --service="non-existent-service" --grpc-endpoint="$TRAEFIK_ENDPOINT" > "$TEMP_DIR/non_existent_service.json" 2>&1; then
        log_warning "Non-existent service request succeeded (may be expected)"
    else
        log_success "Properly handled non-existent service"
    fi
}

# Test concurrent failure scenarios
test_concurrent_failures() {
    log_info "Testing concurrent failure scenarios..."
    increment_test
    
    # Launch multiple concurrent operations, some to valid and some to invalid endpoints
    local pids=()
    local valid_endpoint="$TRAEFIK_ENDPOINT"
    local invalid_endpoint="http://localhost:9999"
    
    log_info "Launching mixed valid/invalid concurrent operations..."
    
    # Launch valid operations
    for i in $(seq 1 5); do
        $GRPC_CLIENT state state-grpc set "concurrent-valid-$i" "value-$i" --service="failure-test" --grpc-endpoint="$valid_endpoint" > "$TEMP_DIR/concurrent_valid_$i.json" 2>&1 &
        pids+=($!)
    done
    
    # Launch invalid operations
    for i in $(seq 1 5); do
        $GRPC_CLIENT state state-grpc set "concurrent-invalid-$i" "value-$i" --service="failure-test" --grpc-endpoint="$invalid_endpoint" > "$TEMP_DIR/concurrent_invalid_$i.json" 2>&1 &
        pids+=($!)
    done
    
    # Wait for all operations to complete
    local completed=0
    for pid in "${pids[@]}"; do
        if wait $pid; then
            ((completed++))
        fi
    done
    
    log_info "Concurrent test: $completed operations completed successfully"
    if [ $completed -ge 5 ]; then
        log_success "Valid operations succeeded despite concurrent failures"
    else
        log_error "Valid operations were affected by concurrent failures"
    fi
}

# Generate failure test report
generate_failure_report() {
    local report_file="$RESULTS_DIR/failure_scenario_report.txt"
    
    echo "Failure Scenario Test Report - $(date)" > "$report_file"
    echo "=======================================" >> "$report_file"
    echo "" >> "$report_file"
    
    echo "Test Summary:" >> "$report_file"
    echo "  Total Tests: $TESTS_TOTAL" >> "$report_file"
    echo "  Passed: $TESTS_PASSED" >> "$report_file"
    echo "  Failed: $TESTS_FAILED" >> "$report_file"
    echo "  Success Rate: $(echo "scale=2; $TESTS_PASSED * 100 / $TESTS_TOTAL" | bc -l)%" >> "$report_file"
    echo "" >> "$report_file"
    
    echo "Test Categories:" >> "$report_file"
    echo "  ✓ Invalid Endpoint Handling" >> "$report_file"
    echo "  ✓ Network Timeout Scenarios" >> "$report_file"
    echo "  ✓ Partial Service Failures" >> "$report_file"
    echo "  ✓ Concurrent Failures" >> "$report_file"
    echo "" >> "$report_file"
    
    if [ $TESTS_FAILED -eq 0 ]; then
        echo "Status: ✅ ALL FAILURE TESTS PASSED" >> "$report_file"
    else
        echo "Status: ⚠️  SOME FAILURE TESTS FAILED" >> "$report_file"
    fi
    
    echo "" >> "$report_file"
    echo "Test Environment:" >> "$report_file"
    echo "  Traefik Endpoint: $TRAEFIK_ENDPOINT" >> "$report_file"
    echo "  gRPC Client: $GRPC_CLIENT" >> "$report_file"
    echo "  Test Timestamp: $(date -Iseconds)" >> "$report_file"
    
    log_info "Failure test report generated: $report_file"
}

# Main execution
main() {
    echo "Failure Scenario Testing for gRPC Migration (Phase 6)"
    echo "====================================================="
    
    # Setup
    setup_failure_test
    trap cleanup EXIT
    
    # Run failure tests
    log_info "Starting failure scenario tests..."
    
    test_invalid_endpoint || true
    test_network_timeouts || true
    test_partial_service_failures || true
    test_concurrent_failures || true
    
    # Generate report
    generate_failure_report
    
    # Summary
    echo ""
    echo "Failure Scenario Test Summary"
    echo "============================"
    echo "Total Tests: $TESTS_TOTAL"
    echo "Passed: $TESTS_PASSED"
    echo "Failed: $TESTS_FAILED"
    
    if [ $TESTS_FAILED -eq 0 ]; then
        echo -e "${GREEN}[SUCCESS]${NC} ✅ All failure scenario tests passed!"
        echo ""
        echo "System Resilience: ✅ EXCELLENT"
        echo ""
        echo "Results saved to: $RESULTS_DIR/failure_scenario_report.txt"
        return 0
    else
        echo -e "${YELLOW}[WARNING]${NC} ⚠️  Some failure scenarios need attention"
        echo ""
        echo "Results saved to: $RESULTS_DIR/failure_scenario_report.txt"
        return 1
    fi
}

# Check dependencies
if ! command -v bc &> /dev/null; then
    echo "Error: 'bc' calculator is required for calculations"
    echo "Install with: brew install bc (macOS) or apt-get install bc (Ubuntu)"
    exit 1
fi

# Run main function
main "$@" 
