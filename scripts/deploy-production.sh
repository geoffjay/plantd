#!/bin/bash

# PlantD Production Deployment Script
# Phase 7: Production Deployment with Blue/Green Strategy

set -euo pipefail

# Script Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
DEPLOY_CONFIG="$PROJECT_ROOT/.env.production"
BACKUP_DIR="$PROJECT_ROOT/backups/$(date +%Y%m%d_%H%M%S)"
LOG_FILE="$PROJECT_ROOT/logs/deploy-$(date +%Y%m%d_%H%M%S).log"

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging function
log() {
    local level="$1"
    shift
    local message="$*"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    
    echo -e "[$timestamp] [$level] $message" | tee -a "$LOG_FILE"
    
    case "$level" in
        ERROR)   echo -e "${RED}[ERROR]${NC} $message" >&2 ;;
        SUCCESS) echo -e "${GREEN}[SUCCESS]${NC} $message" ;;
        WARNING) echo -e "${YELLOW}[WARNING]${NC} $message" ;;
        INFO)    echo -e "${BLUE}[INFO]${NC} $message" ;;
    esac
}

# Error handling
trap 'handle_error $? $LINENO' ERR

handle_error() {
    local exit_code=$1
    local line_no=$2
    log "ERROR" "Deployment failed at line $line_no with exit code $exit_code"
    log "INFO" "Check the log file: $LOG_FILE"
    exit $exit_code
}

# Check prerequisites
check_prerequisites() {
    log "INFO" "Checking deployment prerequisites..."
    
    # Check if Docker is available
    if ! command -v docker &> /dev/null; then
        log "ERROR" "Docker is not installed or not in PATH"
        return 1
    fi
    
    # Check if Docker Compose is available
    if ! command -v docker compose &> /dev/null; then
        log "ERROR" "Docker Compose is not installed or not in PATH"
        return 1
    fi
    
    # Check if environment file exists
    if [[ ! -f "$DEPLOY_CONFIG" ]]; then
        log "WARNING" "Production environment file not found: $DEPLOY_CONFIG"
        log "INFO" "Creating default production environment file..."
        create_default_env
    fi
    
    # Check disk space (minimum 5GB)
    local available_space=$(df "$PROJECT_ROOT" | awk 'NR==2 {print $4}')
    local min_space=5242880  # 5GB in KB
    
    if [[ $available_space -lt $min_space ]]; then
        log "ERROR" "Insufficient disk space. Available: ${available_space}KB, Required: ${min_space}KB"
        return 1
    fi
    
    log "SUCCESS" "Prerequisites check passed"
}

# Create default production environment
create_default_env() {
    cat > "$DEPLOY_CONFIG" << 'EOF'
# PlantD Production Environment Configuration
# Generated automatically - Please customize for your environment

# Database Configuration
POSTGRES_PASSWORD=secure_password_change_me
DATABASE_URL=postgres://plantd:secure_password_change_me@postgres-primary:5432/plantd_identity?sslmode=disable

# JWT Configuration
JWT_SECRET=super_secret_jwt_key_change_me

# Grafana Configuration
GRAFANA_PASSWORD=admin_change_me

# SSL/TLS Configuration
LETSENCRYPT_EMAIL=admin@plantd.com

# Performance Configuration
MAX_CONNECTIONS=200
WORKER_PROCESSES=4

# Security Configuration
CORS_ALLOWED_ORIGINS=https://plantd.com,https://*.plantd.com

# Monitoring Configuration
METRICS_ENABLED=true
TRACING_ENABLED=true

# Deployment Configuration
DEPLOYMENT_ENV=production
BLUE_GREEN_ENABLED=true
HEALTH_CHECK_TIMEOUT=60
ROLLBACK_ENABLED=true
EOF

    log "INFO" "Default production environment created at: $DEPLOY_CONFIG"
    log "WARNING" "Please customize the environment variables before deploying!"
}

# Load environment configuration
load_config() {
    log "INFO" "Loading production configuration..."
    
    if [[ -f "$DEPLOY_CONFIG" ]]; then
        # shellcheck source=/dev/null
        source "$DEPLOY_CONFIG"
        log "SUCCESS" "Configuration loaded from: $DEPLOY_CONFIG"
    else
        log "ERROR" "Configuration file not found: $DEPLOY_CONFIG"
        return 1
    fi
}

# Create backup
create_backup() {
    log "INFO" "Creating deployment backup..."
    
    mkdir -p "$BACKUP_DIR"
    
    # Backup data volumes
    if docker volume ls | grep -q "plantd-production_state-data-primary"; then
        log "INFO" "Backing up state data (primary)..."
        docker run --rm \
            -v plantd-production_state-data-primary:/source:ro \
            -v "$BACKUP_DIR":/backup \
            alpine tar czf /backup/state-data-primary.tar.gz -C /source .
    fi
    
    if docker volume ls | grep -q "plantd-production_identity-db-primary"; then
        log "INFO" "Backing up identity database (primary)..."
        docker run --rm \
            -v plantd-production_identity-db-primary:/source:ro \
            -v "$BACKUP_DIR":/backup \
            alpine tar czf /backup/identity-db-primary.tar.gz -C /source .
    fi
    
    # Backup configuration files
    log "INFO" "Backing up configuration files..."
    cp -r "$PROJECT_ROOT/config" "$BACKUP_DIR/"
    cp -r "$PROJECT_ROOT/traefik" "$BACKUP_DIR/"
    cp "$DEPLOY_CONFIG" "$BACKUP_DIR/"
    
    log "SUCCESS" "Backup created at: $BACKUP_DIR"
}

# Build images
build_images() {
    log "INFO" "Building production images..."
    
    cd "$PROJECT_ROOT"
    
    # Build State service
    log "INFO" "Building State service image..."
    docker build -f docker/state/Dockerfile.grpc -t plantd-state:latest .
    
    # Build Identity service
    log "INFO" "Building Identity service image..."
    docker build -f identity/Dockerfile -t plantd-identity:latest ./identity
    
    log "SUCCESS" "Images built successfully"
}

# Health check function
health_check() {
    local service="$1"
    local url="$2"
    local timeout="${3:-60}"
    local interval="${4:-5}"
    
    log "INFO" "Health checking $service at $url (timeout: ${timeout}s)..."
    
    local start_time=$(date +%s)
    local end_time=$((start_time + timeout))
    
    while [[ $(date +%s) -lt $end_time ]]; do
        if curl -f -s --max-time 10 "$url" > /dev/null 2>&1; then
            log "SUCCESS" "$service health check passed"
            return 0
        fi
        
        log "INFO" "Waiting for $service to be healthy..."
        sleep "$interval"
    done
    
    log "ERROR" "$service health check failed after ${timeout}s"
    return 1
}

# Wait for services
wait_for_services() {
    log "INFO" "Waiting for services to be healthy..."
    
    # Check Traefik
    health_check "Traefik" "http://localhost:8080/ping" 60 5
    
    # Check State service through Traefik
    health_check "State Service" "http://localhost:8080/health" 120 10
    
    # Check Identity service
    health_check "Identity Service" "http://localhost:8080/auth/health" 120 10
    
    # Check Prometheus
    health_check "Prometheus" "http://localhost:9090/-/healthy" 60 5
    
    # Check Grafana
    health_check "Grafana" "http://localhost:3000/api/health" 60 5
    
    log "SUCCESS" "All services are healthy"
}

# Blue/Green deployment
blue_green_deploy() {
    log "INFO" "Starting Blue/Green deployment..."
    
    local current_stack="blue"
    local new_stack="green"
    
    # Check if there's a current deployment
    if docker compose -f docker/docker-compose.production.yml ps | grep -q "Up"; then
        log "INFO" "Existing deployment detected"
        current_stack="green"
        new_stack="blue"
    fi
    
    log "INFO" "Deploying to $new_stack environment..."
    
    # Deploy new stack
    COMPOSE_PROJECT_NAME="plantd-$new_stack" docker compose \
        -f docker/docker-compose.production.yml \
        --env-file "$DEPLOY_CONFIG" \
        up -d --build --remove-orphans
    
    # Wait for new stack to be healthy
    log "INFO" "Waiting for $new_stack stack to be healthy..."
    sleep 30
    
    # Verify new stack health
    if wait_for_services; then
        log "SUCCESS" "$new_stack stack is healthy"
        
        # Switch traffic (would require load balancer configuration)
        log "INFO" "Switching traffic to $new_stack stack..."
        
        # Stop old stack
        if [[ "$current_stack" != "$new_stack" ]]; then
            log "INFO" "Stopping $current_stack stack..."
            COMPOSE_PROJECT_NAME="plantd-$current_stack" docker compose \
                -f docker/docker-compose.production.yml \
                down --remove-orphans
        fi
        
        log "SUCCESS" "Blue/Green deployment completed"
    else
        log "ERROR" "$new_stack stack health check failed"
        log "INFO" "Rolling back..."
        
        # Stop failed stack
        COMPOSE_PROJECT_NAME="plantd-$new_stack" docker compose \
            -f docker/docker-compose.production.yml \
            down --remove-orphans
        
        return 1
    fi
}

# Standard deployment
standard_deploy() {
    log "INFO" "Starting standard deployment..."
    
    cd "$PROJECT_ROOT"
    
    # Deploy services
    docker compose -f docker/docker-compose.production.yml \
        --env-file "$DEPLOY_CONFIG" \
        up -d --build --remove-orphans
    
    # Wait for services
    wait_for_services
    
    log "SUCCESS" "Standard deployment completed"
}

# Rollback function
rollback() {
    local backup_path="$1"
    
    log "INFO" "Starting rollback process..."
    
    if [[ ! -d "$backup_path" ]]; then
        log "ERROR" "Backup directory not found: $backup_path"
        return 1
    fi
    
    # Stop current services
    log "INFO" "Stopping current services..."
    docker compose -f docker/docker-compose.production.yml down --remove-orphans
    
    # Restore data volumes
    if [[ -f "$backup_path/state-data-primary.tar.gz" ]]; then
        log "INFO" "Restoring state data (primary)..."
        docker run --rm \
            -v plantd-production_state-data-primary:/target \
            -v "$backup_path":/backup \
            alpine sh -c "cd /target && tar xzf /backup/state-data-primary.tar.gz"
    fi
    
    if [[ -f "$backup_path/identity-db-primary.tar.gz" ]]; then
        log "INFO" "Restoring identity database (primary)..."
        docker run --rm \
            -v plantd-production_identity-db-primary:/target \
            -v "$backup_path":/backup \
            alpine sh -c "cd /target && tar xzf /backup/identity-db-primary.tar.gz"
    fi
    
    # Restore configuration
    if [[ -d "$backup_path/config" ]]; then
        log "INFO" "Restoring configuration files..."
        cp -r "$backup_path/config" "$PROJECT_ROOT/"
        cp -r "$backup_path/traefik" "$PROJECT_ROOT/"
    fi
    
    # Restart services
    log "INFO" "Restarting services..."
    docker compose -f docker/docker-compose.production.yml \
        --env-file "$backup_path/.env.production" \
        up -d
    
    wait_for_services
    
    log "SUCCESS" "Rollback completed"
}

# Deployment status
deployment_status() {
    log "INFO" "Checking deployment status..."
    
    echo "=== PlantD Production Status ==="
    echo
    
    # Docker Compose status
    echo "Services Status:"
    docker compose -f docker/docker-compose.production.yml ps
    echo
    
    # Health checks
    echo "Health Checks:"
    
    # Traefik
    if curl -f -s --max-time 5 "http://localhost:8080/ping" > /dev/null 2>&1; then
        echo "✅ Traefik Gateway: Healthy"
    else
        echo "❌ Traefik Gateway: Unhealthy"
    fi
    
    # State Service
    if curl -f -s --max-time 5 "http://localhost:8080/health" > /dev/null 2>&1; then
        echo "✅ State Service: Healthy"
    else
        echo "❌ State Service: Unhealthy"
    fi
    
    # Identity Service
    if curl -f -s --max-time 5 "http://localhost:8080/auth/health" > /dev/null 2>&1; then
        echo "✅ Identity Service: Healthy"
    else
        echo "❌ Identity Service: Unhealthy"
    fi
    
    # Resource usage
    echo
    echo "Resource Usage:"
    docker stats --no-stream --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.MemPerc}}"
}

# Main deployment function
main() {
    local command="${1:-deploy}"
    
    # Create log directory
    mkdir -p "$(dirname "$LOG_FILE")"
    
    log "INFO" "Starting PlantD production deployment"
    log "INFO" "Command: $command"
    log "INFO" "Log file: $LOG_FILE"
    
    case "$command" in
        "deploy")
            check_prerequisites
            load_config
            create_backup
            build_images
            
            if [[ "${BLUE_GREEN_ENABLED:-true}" == "true" ]]; then
                blue_green_deploy
            else
                standard_deploy
            fi
            ;;
        
        "rollback")
            local backup_path="${2:-}"
            if [[ -z "$backup_path" ]]; then
                log "ERROR" "Backup path required for rollback"
                echo "Usage: $0 rollback <backup_path>"
                exit 1
            fi
            rollback "$backup_path"
            ;;
        
        "status")
            deployment_status
            ;;
        
        "backup")
            load_config
            create_backup
            ;;
        
        "health")
            wait_for_services
            ;;
        
        "stop")
            log "INFO" "Stopping production services..."
            docker compose -f docker/docker-compose.production.yml down --remove-orphans
            log "SUCCESS" "Services stopped"
            ;;
        
        "logs")
            local service="${2:-}"
            if [[ -n "$service" ]]; then
                docker compose -f docker/docker-compose.production.yml logs -f "$service"
            else
                docker compose -f docker/docker-compose.production.yml logs -f
            fi
            ;;
        
        *)
            echo "Usage: $0 {deploy|rollback|status|backup|health|stop|logs}"
            echo
            echo "Commands:"
            echo "  deploy           - Deploy to production"
            echo "  rollback <path>  - Rollback to backup"
            echo "  status           - Show deployment status"
            echo "  backup           - Create backup"
            echo "  health           - Check service health"
            echo "  stop             - Stop all services"
            echo "  logs [service]   - Show logs"
            exit 1
            ;;
    esac
    
    log "SUCCESS" "Command '$command' completed successfully"
}

# Execute main function with all arguments
main "$@" 
