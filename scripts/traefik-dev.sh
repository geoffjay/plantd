#!/bin/bash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
COMPOSE_FILE="$PROJECT_ROOT/docker-compose.traefik.yml"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_dependencies() {
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed or not in PATH"
        exit 1
    fi

    if ! command -v docker-compose &> /dev/null; then
        log_error "Docker Compose is not installed or not in PATH"
        exit 1
    fi
}

build_protobuf() {
    log_info "Building protocol buffers..."
    cd "$PROJECT_ROOT"
    make proto-gen
    log_success "Protocol buffers built successfully"
}

start_gateway() {
    log_info "Starting Traefik gateway environment..."
    
    # Ensure logs directory exists
    mkdir -p "$PROJECT_ROOT/logs/traefik"
    
    # Build protocol buffers first
    build_protobuf
    
    # Start services
    cd "$PROJECT_ROOT"
    docker-compose -f "$COMPOSE_FILE" up -d
    
    log_success "Traefik gateway environment started"
    
    # Wait for services to be healthy
    log_info "Waiting for services to be healthy..."
    sleep 10
    
    # Check service status
    show_status
}

stop_gateway() {
    log_info "Stopping Traefik gateway environment..."
    cd "$PROJECT_ROOT"
    docker-compose -f "$COMPOSE_FILE" down
    log_success "Traefik gateway environment stopped"
}

restart_gateway() {
    log_info "Restarting Traefik gateway environment..."
    stop_gateway
    start_gateway
}

show_logs() {
    local service=${1:-""}
    cd "$PROJECT_ROOT"
    
    if [ -n "$service" ]; then
        log_info "Showing logs for service: $service"
        docker-compose -f "$COMPOSE_FILE" logs -f "$service"
    else
        log_info "Showing logs for all services"
        docker-compose -f "$COMPOSE_FILE" logs -f
    fi
}

show_status() {
    cd "$PROJECT_ROOT"
    
    log_info "Service Status:"
    echo "==============="
    
    # Check if services are running
    docker-compose -f "$COMPOSE_FILE" ps
    
    echo ""
    log_info "Gateway Endpoints:"
    echo "=================="
    echo "🌐 Traefik Dashboard: http://localhost:9000"
    echo "🔧 gRPC Gateway:      http://localhost:8080"
    echo "🌍 HTTP Gateway:      http://localhost:80"
    echo "🔒 HTTPS Gateway:     http://localhost:443"
    echo ""
    
    log_info "Service Health Checks:"
    echo "====================="
    
    # Check health endpoints
    local endpoints=(
        "http://localhost:8080/health:State Service"
        "http://localhost:9000/ping:Traefik Dashboard"
    )
    
    for endpoint_info in "${endpoints[@]}"; do
        local endpoint="${endpoint_info%%:*}"
        local name="${endpoint_info##*:}"
        
        if curl -s -f "$endpoint" > /dev/null 2>&1; then
            echo -e "✅ $name: ${GREEN}Healthy${NC}"
        else
            echo -e "❌ $name: ${RED}Unhealthy${NC}"
        fi
    done
}

test_gateway() {
    log_info "Testing Traefik gateway functionality..."
    
    # Test State service through gateway
    log_info "Testing State service through Traefik gateway..."
    
    # Test health endpoint
    if curl -s -f "http://localhost:8080/health" > /dev/null; then
        log_success "✅ Health endpoint accessible through gateway"
    else
        log_error "❌ Health endpoint not accessible through gateway"
        return 1
    fi
    
    # Test MDP compatibility endpoint
    log_info "Testing MDP compatibility endpoint..."
    response=$(curl -s -X POST "http://localhost:8080/mdp/set/test-gateway-key/test-gateway-value" || echo "failed")
    if [[ "$response" == *"success"* ]]; then
        log_success "✅ MDP compatibility endpoint working through gateway"
        
        # Test GET
        get_response=$(curl -s -X POST "http://localhost:8080/mdp/get/test-gateway-key" || echo "failed")
        if [[ "$get_response" == *"test-gateway-value"* ]]; then
            log_success "✅ MDP GET working through gateway"
        else
            log_warning "⚠️  MDP GET might have issues through gateway"
        fi
    else
        log_error "❌ MDP compatibility endpoint not working through gateway"
        return 1
    fi
    
    log_success "🎉 All gateway tests passed!"
}

clean_gateway() {
    log_warning "Cleaning up Traefik gateway environment (this will remove volumes)..."
    read -p "Are you sure? This will delete all data! (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        cd "$PROJECT_ROOT"
        docker-compose -f "$COMPOSE_FILE" down -v --remove-orphans
        docker system prune -f
        log_success "Traefik gateway environment cleaned"
    else
        log_info "Clean operation cancelled"
    fi
}

show_help() {
    echo "Traefik Gateway Development Script"
    echo "================================="
    echo ""
    echo "Usage: $0 [COMMAND]"
    echo ""
    echo "Commands:"
    echo "  start       Start the Traefik gateway environment"
    echo "  stop        Stop the Traefik gateway environment"
    echo "  restart     Restart the Traefik gateway environment"
    echo "  status      Show service status and endpoints"
    echo "  logs [SVC]  Show logs (optionally for specific service)"
    echo "  test        Test gateway functionality"
    echo "  clean       Clean up environment (removes volumes)"
    echo "  help        Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 start                 # Start all services"
    echo "  $0 logs state            # Show logs for state service"
    echo "  $0 test                  # Test gateway functionality"
}

# Main script logic
check_dependencies

case "${1:-help}" in
    start)
        start_gateway
        ;;
    stop)
        stop_gateway
        ;;
    restart)
        restart_gateway
        ;;
    status)
        show_status
        ;;
    logs)
        show_logs "$2"
        ;;
    test)
        test_gateway
        ;;
    clean)
        clean_gateway
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        log_error "Unknown command: $1"
        show_help
        exit 1
        ;;
esac 