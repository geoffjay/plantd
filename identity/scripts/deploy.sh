#!/bin/bash

# Identity Service Deployment Script
# This script handles the deployment of the plantd identity service using Docker Compose

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
ROOT_DIR="$(dirname "$PROJECT_DIR")"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
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

# Function to check if command exists
check_command() {
    if ! command -v "$1" &> /dev/null; then
        log_error "Required command '$1' not found. Please install it first."
        exit 1
    fi
}

# Function to check environment file
check_environment() {
    local env_file="$PROJECT_DIR/.env"
    
    if [[ ! -f "$env_file" ]]; then
        log_warning "Environment file not found at $env_file"
        log_info "Creating .env from template..."
        
        if [[ -f "$PROJECT_DIR/scripts/env.example" ]]; then
            cp "$PROJECT_DIR/scripts/env.example" "$env_file"
            log_warning "Please edit $env_file with your actual configuration values"
            log_warning "Especially change the JWT_SECRET for production use!"
        else
            log_error "Template environment file not found. Cannot continue."
            exit 1
        fi
    fi
    
    # Check for critical configuration
    if grep -q "your-super-secret-jwt-key-change-this-in-production" "$env_file"; then
        log_warning "JWT_SECRET is still using the default value. Please change it for production!"
    fi
}

# Function to build the Docker image
build_image() {
    log_info "Building identity service Docker image..."
    
    cd "$ROOT_DIR"
    
    if docker build -t geoffjay/plantd-identity:latest -f identity/Dockerfile .; then
        log_success "Docker image built successfully"
    else
        log_error "Failed to build Docker image"
        exit 1
    fi
}

# Function to start services
start_services() {
    log_info "Starting identity service and dependencies..."
    
    cd "$PROJECT_DIR"
    
    # Create network if it doesn't exist
    if ! docker network ls | grep -q "plantd-network"; then
        log_info "Creating plantd-network..."
        docker network create plantd-network
    fi
    
    # Start services
    if docker-compose up -d; then
        log_success "Services started successfully"
    else
        log_error "Failed to start services"
        exit 1
    fi
}

# Function to check service health
check_health() {
    log_info "Checking service health..."
    
    local max_attempts=30
    local attempt=1
    
    while [[ $attempt -le $max_attempts ]]; do
        if curl -f http://localhost:8080/health &> /dev/null; then
            log_success "Identity service is healthy and responding"
            return 0
        fi
        
        log_info "Attempt $attempt/$max_attempts: Waiting for service to be ready..."
        sleep 5
        ((attempt++))
    done
    
    log_error "Service health check failed after $max_attempts attempts"
    log_info "Checking service logs..."
    docker-compose logs identity
    return 1
}

# Function to show service status
show_status() {
    log_info "Service status:"
    docker-compose ps
    
    echo ""
    log_info "Service logs (last 20 lines):"
    docker-compose logs --tail=20 identity
}

# Function to stop services
stop_services() {
    log_info "Stopping identity service..."
    
    cd "$PROJECT_DIR"
    
    if docker-compose down; then
        log_success "Services stopped successfully"
    else
        log_error "Failed to stop services"
        exit 1
    fi
}

# Function to clean up
cleanup() {
    log_info "Cleaning up Docker resources..."
    
    cd "$PROJECT_DIR"
    
    docker-compose down -v --remove-orphans
    docker system prune -f
    
    log_success "Cleanup completed"
}

# Function to show help
show_help() {
    echo "Identity Service Deployment Script"
    echo ""
    echo "Usage: $0 [COMMAND]"
    echo ""
    echo "Commands:"
    echo "  deploy      Build and deploy the identity service (default)"
    echo "  build       Build Docker image only"
    echo "  start       Start services without building"
    echo "  stop        Stop all services"
    echo "  restart     Restart all services"
    echo "  status      Show service status and logs"
    echo "  health      Check service health"
    echo "  cleanup     Stop services and clean up resources"
    echo "  help        Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 deploy     # Full deployment"
    echo "  $0 restart    # Restart services"
    echo "  $0 status     # Check status"
}

# Main function
main() {
    local command="${1:-deploy}"
    
    case "$command" in
        "deploy")
            log_info "Starting full deployment..."
            check_command docker
            check_command docker-compose
            check_command curl
            check_environment
            build_image
            start_services
            check_health
            show_status
            log_success "Deployment completed successfully!"
            ;;
        "build")
            check_command docker
            build_image
            ;;
        "start")
            check_command docker-compose
            check_environment
            start_services
            check_health
            ;;
        "stop")
            check_command docker-compose
            stop_services
            ;;
        "restart")
            check_command docker-compose
            stop_services
            start_services
            check_health
            ;;
        "status")
            check_command docker-compose
            show_status
            ;;
        "health")
            check_health
            ;;
        "cleanup")
            check_command docker-compose
            cleanup
            ;;
        "help"|"-h"|"--help")
            show_help
            ;;
        *)
            log_error "Unknown command: $command"
            show_help
            exit 1
            ;;
    esac
}

# Run main function with all arguments
main "$@" 
