#!/bin/bash

# Load Testing Script for gRPC Migration (Phase 6)
# Stress tests gRPC services through Traefik gateway

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
RESULTS_DIR="./test-results/load"
TEMP_DIR="./test-data/load"

# Load test parameters
CONCURRENT_USERS=${CONCURRENT_USERS:-10}
OPERATIONS_PER_USER=${OPERATIONS_PER_USER:-100}
RAMP_UP_TIME=${RAMP_UP_TIME:-10}
TEST_DURATION=${TEST_DURATION:-300}  # 5 minutes
THINK_TIME=${THINK_TIME:-0.1}  # 100ms between operations

# Test counters
TOTAL_OPERATIONS=0
SUCCESSFUL_OPERATIONS=0
FAILED_OPERATIONS=0
START_TIME=0
END_TIME=0

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} ✓ $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} ✗ $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} ! $1"
}

# Cleanup function
cleanup() {
    log_info "Cleaning up load test data..."
    if [ -d "$TEMP_DIR" ]; then
        rm -rf "$TEMP_DIR"
    fi
    
    # Kill any remaining background processes
    jobs -p | xargs -r kill || true
    wait
}

# Setup test environment
setup_load_test() {
    log_info "Setting up load test environment..."
    mkdir -p "$TEMP_DIR"
    mkdir -p "$RESULTS_DIR"
    
    # Ensure client is built
    if [ ! -f "$GRPC_CLIENT" ]; then
        log_info "Building gRPC client..."
        make build-client-grpc
    fi
    
    # Create test data files
    echo "user_id,operation_count,success_count,error_count,total_time,ops_per_sec" > "$RESULTS_DIR/load_test_results.csv"
    
    # Pre-populate some test data
    log_info "Pre-populating test data..."
    for i in $(seq 1 10); do
        $GRPC_CLIENT state state-grpc set "load-test-base-$i" "base-value-$i" --service="load-test" --grpc-endpoint="$TRAEFIK_ENDPOINT" > /dev/null 2>&1 || true
    done
}

# Simulate user operations
simulate_user() {
    local user_id=$1
    local operations_count=$OPERATIONS_PER_USER
    local user_temp_dir="$TEMP_DIR/user_$user_id"
    local user_results="$user_temp_dir/results.json"
    
    mkdir -p "$user_temp_dir"
    
    local success_count=0
    local error_count=0
    local start_time=$(date +%s.%N)
    
    log_info "Starting user $user_id with $operations_count operations"
    
    for op in $(seq 1 $operations_count); do
        local operation_type=$((RANDOM % 4))  # 0=GET, 1=SET, 2=LIST, 3=DELETE
        local key="load-test-user-$user_id-op-$op"
        local value="value-$user_id-$op-$(date +%s)"
        local service="load-test-user-$user_id"
        
        case $operation_type in
            0) # GET operation
                if $GRPC_CLIENT state state-grpc get "$key" --service="$service" --grpc-endpoint="$TRAEFIK_ENDPOINT" > "$user_temp_dir/op_$op.json" 2>&1; then
                    ((success_count++))
                else
                    ((error_count++))
                fi
                ;;
            1) # SET operation
                if $GRPC_CLIENT state state-grpc set "$key" "$value" --service="$service" --grpc-endpoint="$TRAEFIK_ENDPOINT" > "$user_temp_dir/op_$op.json" 2>&1; then
                    ((success_count++))
                else
                    ((error_count++))
                fi
                ;;
            2) # LIST operation
                if $GRPC_CLIENT state state-grpc list --service="$service" --grpc-endpoint="$TRAEFIK_ENDPOINT" > "$user_temp_dir/op_$op.json" 2>&1; then
                    ((success_count++))
                else
                    ((error_count++))
                fi
                ;;
            3) # DELETE operation (only if we have some data)
                if [ $op -gt 10 ]; then
                    local delete_key="load-test-user-$user_id-op-$((op-10))"
                    if $GRPC_CLIENT state state-grpc delete "$delete_key" --service="$service" --grpc-endpoint="$TRAEFIK_ENDPOINT" > "$user_temp_dir/op_$op.json" 2>&1; then
                        ((success_count++))
                    else
                        ((error_count++))
                    fi
                else
                    # Fall back to SET operation
                    if $GRPC_CLIENT state state-grpc set "$key" "$value" --service="$service" --grpc-endpoint="$TRAEFIK_ENDPOINT" > "$user_temp_dir/op_$op.json" 2>&1; then
                        ((success_count++))
                    else
                        ((error_count++))
                    fi
                fi
                ;;
        esac
        
        # Think time between operations
        sleep $THINK_TIME
    done
    
    local end_time=$(date +%s.%N)
    local total_time=$(echo "$end_time - $start_time" | bc -l)
    local ops_per_sec=$(echo "scale=2; $operations_count / $total_time" | bc -l)
    
    # Write results
    echo "$user_id,$operations_count,$success_count,$error_count,$total_time,$ops_per_sec" >> "$RESULTS_DIR/load_test_results.csv"
    
    log_info "User $user_id completed: $success_count successful, $error_count errors, ${ops_per_sec} ops/sec"
}

# Test resource limits
test_resource_limits() {
    log_info "Testing resource limits and connection pooling..."
    
    local max_connections=50
    local pids=()
    local success_count=0
    local error_count=0
    
    # Launch many concurrent connections
    for i in $(seq 1 $max_connections); do
        $GRPC_CLIENT state state-grpc get "resource-test-$i" --service="resource-test" --grpc-endpoint="$TRAEFIK_ENDPOINT" > "$TEMP_DIR/resource_$i.json" 2>&1 && ((success_count++)) || ((error_count++)) &
        pids+=($!)
    done
    
    # Wait for all to complete
    for pid in "${pids[@]}"; do
        wait $pid
    done
    
    local total_connections=$((success_count + error_count))
    local success_rate=$(echo "scale=2; $success_count * 100 / $total_connections" | bc -l)
    
    log_info "Resource limit test: $success_count successful, $error_count errors out of $total_connections connections"
    log_info "Connection success rate: ${success_rate}%"
    
    # Write results
    echo "Resource Limit Test Results" > "$RESULTS_DIR/resource_limits.txt"
    echo "===========================" >> "$RESULTS_DIR/resource_limits.txt"
    echo "Max Concurrent Connections: $max_connections" >> "$RESULTS_DIR/resource_limits.txt"
    echo "Successful Connections: $success_count" >> "$RESULTS_DIR/resource_limits.txt"
    echo "Failed Connections: $error_count" >> "$RESULTS_DIR/resource_limits.txt"
    echo "Success Rate: ${success_rate}%" >> "$RESULTS_DIR/resource_limits.txt"
}

# Analyze results
analyze_results() {
    log_info "Analyzing load test results..."
    
    local analysis_file="$RESULTS_DIR/load_test_analysis.txt"
    
    echo "Load Test Analysis - $(date)" > "$analysis_file"
    echo "==============================" >> "$analysis_file"
    echo "" >> "$analysis_file"
    
    # Test configuration
    echo "Test Configuration:" >> "$analysis_file"
    echo "  Concurrent Users: $CONCURRENT_USERS" >> "$analysis_file"
    echo "  Operations per User: $OPERATIONS_PER_USER" >> "$analysis_file"
    echo "  Ramp-up Time: ${RAMP_UP_TIME}s" >> "$analysis_file"
    echo "  Think Time: ${THINK_TIME}s" >> "$analysis_file"
    echo "" >> "$analysis_file"
    
    # Analyze CSV results
    if [ -f "$RESULTS_DIR/load_test_results.csv" ]; then
        local total_ops=$(tail -n +2 "$RESULTS_DIR/load_test_results.csv" | awk -F',' '{sum+=$2} END {print sum}')
        local total_success=$(tail -n +2 "$RESULTS_DIR/load_test_results.csv" | awk -F',' '{sum+=$3} END {print sum}')
        local total_errors=$(tail -n +2 "$RESULTS_DIR/load_test_results.csv" | awk -F',' '{sum+=$4} END {print sum}')
        local avg_ops_per_sec=$(tail -n +2 "$RESULTS_DIR/load_test_results.csv" | awk -F',' '{sum+=$6} END {print sum/NR}')
        local success_rate=$(echo "scale=2; $total_success * 100 / $total_ops" | bc -l)
        
        echo "Performance Results:" >> "$analysis_file"
        echo "  Total Operations: $total_ops" >> "$analysis_file"
        echo "  Successful Operations: $total_success" >> "$analysis_file"
        echo "  Failed Operations: $total_errors" >> "$analysis_file"
        echo "  Success Rate: ${success_rate}%" >> "$analysis_file"
        echo "  Average Ops/Sec per User: $avg_ops_per_sec" >> "$analysis_file"
        echo "  Total System Throughput: $(echo "scale=2; $avg_ops_per_sec * $CONCURRENT_USERS" | bc -l) ops/sec" >> "$analysis_file"
        echo "" >> "$analysis_file"
        
        # Performance assessment
        echo "Performance Assessment:" >> "$analysis_file"
        if (( $(echo "$success_rate >= 95" | bc -l) )); then
            echo "  ✅ Excellent: Success rate above 95%" >> "$analysis_file"
        elif (( $(echo "$success_rate >= 90" | bc -l) )); then
            echo "  ✅ Good: Success rate above 90%" >> "$analysis_file"
        elif (( $(echo "$success_rate >= 80" | bc -l) )); then
            echo "  ⚠️  Acceptable: Success rate above 80%" >> "$analysis_file"
        else
            echo "  ❌ Poor: Success rate below 80%" >> "$analysis_file"
        fi
        
        if (( $(echo "$avg_ops_per_sec >= 10" | bc -l) )); then
            echo "  ✅ Good throughput: Average ${avg_ops_per_sec} ops/sec per user" >> "$analysis_file"
        else
            echo "  ⚠️  Low throughput: Average ${avg_ops_per_sec} ops/sec per user" >> "$analysis_file"
        fi
    fi
    
    echo "" >> "$analysis_file"
    echo "Files Generated:" >> "$analysis_file"
    echo "  - load_test_results.csv: Per-user performance data" >> "$analysis_file"
    echo "  - resource_limits.txt: Resource limit test results" >> "$analysis_file"
    echo "  - load_test_analysis.txt: This analysis file" >> "$analysis_file"
    
    log_info "Analysis complete. Results saved to: $analysis_file"
}

# Main execution
main() {
    echo "Load Testing for gRPC Migration (Phase 6)"
    echo "========================================="
    echo ""
    echo "Test Configuration:"
    echo "  Concurrent Users: $CONCURRENT_USERS"
    echo "  Operations per User: $OPERATIONS_PER_USER"
    echo "  Ramp-up Time: ${RAMP_UP_TIME}s"
    echo "  Think Time: ${THINK_TIME}s"
    echo ""
    
    # Setup
    setup_load_test
    trap cleanup EXIT
    
    START_TIME=$(date +%s.%N)
    
    # Run load test
    log_info "Starting load test with gradual ramp-up..."
    
    local ramp_delay=$(echo "scale=2; $RAMP_UP_TIME / $CONCURRENT_USERS" | bc -l)
    local pids=()
    
    for user_id in $(seq 1 $CONCURRENT_USERS); do
        simulate_user $user_id &
        pids+=($!)
        
        # Ramp up delay
        sleep $ramp_delay
    done
    
    log_info "All users started, waiting for completion..."
    
    # Wait for all users to complete
    for pid in "${pids[@]}"; do
        wait $pid
    done
    
    log_success "All users completed"
    
    # Test resource limits
    test_resource_limits
    
    END_TIME=$(date +%s.%N)
    
    # Analyze results
    analyze_results
    
    # Final summary
    echo ""
    echo "Load Test Summary"
    echo "================"
    echo "Total Test Time: $(echo "scale=2; $END_TIME - $START_TIME" | bc -l)s"
    echo ""
    echo "Results available in: $RESULTS_DIR/"
    echo "  - load_test_analysis.txt: Performance analysis"
    echo "  - load_test_results.csv: Raw performance data"
    echo "  - resource_limits.txt: Resource limit results"
    echo ""
    echo "Load Testing: ✅ COMPLETE"
    echo ""
    echo "Next steps:"
    echo "  - Review performance metrics"
    echo "  - Identify bottlenecks if any"
    echo "  - Optimize configuration if needed"
}

# Check dependencies
if ! command -v bc &> /dev/null; then
    echo "Error: 'bc' calculator is required for performance calculations"
    echo "Install with: brew install bc (macOS) or apt-get install bc (Ubuntu)"
    exit 1
fi

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --users)
            CONCURRENT_USERS="$2"
            shift 2
            ;;
        --operations)
            OPERATIONS_PER_USER="$2"
            shift 2
            ;;
        --ramp-up)
            RAMP_UP_TIME="$2"
            shift 2
            ;;
        --think-time)
            THINK_TIME="$2"
            shift 2
            ;;
        --help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --users NUM          Number of concurrent users (default: 10)"
            echo "  --operations NUM     Operations per user (default: 100)"
            echo "  --ramp-up SECONDS    Ramp-up time (default: 10)"
            echo "  --think-time SECONDS Think time between operations (default: 0.1)"
            echo "  --help              Show this help message"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

# Run main function
main "$@" 