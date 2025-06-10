#!/bin/bash

# Identity Service Backup and Recovery Script
# This script handles backup and recovery operations for the plantd identity service

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
BACKUP_DIR="${BACKUP_DIR:-$PROJECT_DIR/backups}"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")

# Database configuration
DB_CONTAINER="plantd-identity-postgres"
DB_NAME="plantd_identity"
DB_USER="identity_user"

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

# Function to check if container is running
check_container() {
    local container_name="$1"
    
    if ! docker ps | grep -q "$container_name"; then
        log_error "Container $container_name is not running"
        log_info "Please start the identity service first using: ./scripts/deploy.sh start"
        exit 1
    fi
}

# Function to create backup directory
create_backup_dir() {
    if [[ ! -d "$BACKUP_DIR" ]]; then
        log_info "Creating backup directory: $BACKUP_DIR"
        mkdir -p "$BACKUP_DIR"
    fi
}

# Function to backup database
backup_database() {
    log_info "Creating database backup..."
    
    check_container "$DB_CONTAINER"
    create_backup_dir
    
    local backup_file="$BACKUP_DIR/identity_db_${TIMESTAMP}.sql"
    
    if docker exec "$DB_CONTAINER" pg_dump -U "$DB_USER" -d "$DB_NAME" > "$backup_file"; then
        log_success "Database backup created: $backup_file"
        
        # Compress the backup
        if gzip "$backup_file"; then
            log_success "Backup compressed: ${backup_file}.gz"
        fi
    else
        log_error "Failed to create database backup"
        exit 1
    fi
}

# Function to backup configuration
backup_config() {
    log_info "Creating configuration backup..."
    
    create_backup_dir
    
    local config_backup_dir="$BACKUP_DIR/config_${TIMESTAMP}"
    mkdir -p "$config_backup_dir"
    
    # Backup configuration files
    if [[ -f "$PROJECT_DIR/identity.yaml" ]]; then
        cp "$PROJECT_DIR/identity.yaml" "$config_backup_dir/"
    fi
    
    if [[ -f "$PROJECT_DIR/.env" ]]; then
        cp "$PROJECT_DIR/.env" "$config_backup_dir/"
    fi
    
    if [[ -f "$PROJECT_DIR/docker-compose.yml" ]]; then
        cp "$PROJECT_DIR/docker-compose.yml" "$config_backup_dir/"
    fi
    
    # Create archive
    local config_archive="$BACKUP_DIR/identity_config_${TIMESTAMP}.tar.gz"
    if tar -czf "$config_archive" -C "$BACKUP_DIR" "config_${TIMESTAMP}"; then
        log_success "Configuration backup created: $config_archive"
        rm -rf "$config_backup_dir"
    else
        log_error "Failed to create configuration backup"
        exit 1
    fi
}

# Function to backup Docker volumes
backup_volumes() {
    log_info "Creating volume backup..."
    
    create_backup_dir
    
    local volume_backup_file="$BACKUP_DIR/identity_volumes_${TIMESTAMP}.tar.gz"
    
    # Create a temporary container to backup volumes
    if docker run --rm \
        -v plantd-identity-data:/data \
        -v "$BACKUP_DIR:/backup" \
        alpine:latest \
        tar -czf "/backup/identity_volumes_${TIMESTAMP}.tar.gz" -C /data .; then
        log_success "Volume backup created: $volume_backup_file"
    else
        log_error "Failed to create volume backup"
        exit 1
    fi
}

# Function to create full backup
full_backup() {
    log_info "Creating full backup (database + config + volumes)..."
    
    backup_database
    backup_config
    backup_volumes
    
    log_success "Full backup completed"
    list_backups
}

# Function to restore database
restore_database() {
    local backup_file="$1"
    
    if [[ ! -f "$backup_file" ]]; then
        log_error "Backup file not found: $backup_file"
        exit 1
    fi
    
    log_info "Restoring database from: $backup_file"
    log_warning "This will overwrite the current database!"
    
    read -p "Are you sure? (y/N): " -n 1 -r
    echo
    
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log_info "Restore cancelled"
        return
    fi
    
    check_container "$DB_CONTAINER"
    
    # Determine if file is compressed
    local restore_command
    if [[ "$backup_file" == *.gz ]]; then
        restore_command="zcat '$backup_file' | docker exec -i '$DB_CONTAINER' psql -U '$DB_USER' -d '$DB_NAME'"
    else
        restore_command="docker exec -i '$DB_CONTAINER' psql -U '$DB_USER' -d '$DB_NAME' < '$backup_file'"
    fi
    
    if eval "$restore_command"; then
        log_success "Database restored successfully"
    else
        log_error "Failed to restore database"
        exit 1
    fi
}

# Function to restore volumes
restore_volumes() {
    local backup_file="$1"
    
    if [[ ! -f "$backup_file" ]]; then
        log_error "Volume backup file not found: $backup_file"
        exit 1
    fi
    
    log_info "Restoring volumes from: $backup_file"
    log_warning "This will overwrite current volume data!"
    
    read -p "Are you sure? (y/N): " -n 1 -r
    echo
    
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log_info "Restore cancelled"
        return
    fi
    
    # Stop services before restoring volumes
    log_info "Stopping services for volume restore..."
    cd "$PROJECT_DIR"
    docker-compose down
    
    # Restore volumes
    if docker run --rm \
        -v plantd-identity-data:/data \
        -v "$(dirname "$backup_file"):/backup" \
        alpine:latest \
        sh -c "cd /data && tar -xzf /backup/$(basename "$backup_file")"; then
        log_success "Volumes restored successfully"
        
        # Restart services
        log_info "Restarting services..."
        docker-compose up -d
    else
        log_error "Failed to restore volumes"
        exit 1
    fi
}

# Function to list available backups
list_backups() {
    log_info "Available backups in $BACKUP_DIR:"
    
    if [[ ! -d "$BACKUP_DIR" ]] || [[ -z "$(ls -A "$BACKUP_DIR" 2>/dev/null)" ]]; then
        log_warning "No backups found"
        return
    fi
    
    echo ""
    echo "Database backups:"
    ls -lh "$BACKUP_DIR"/identity_db_*.sql* 2>/dev/null || echo "  None found"
    
    echo ""
    echo "Configuration backups:"
    ls -lh "$BACKUP_DIR"/identity_config_*.tar.gz 2>/dev/null || echo "  None found"
    
    echo ""
    echo "Volume backups:"
    ls -lh "$BACKUP_DIR"/identity_volumes_*.tar.gz 2>/dev/null || echo "  None found"
    echo ""
}

# Function to clean old backups
clean_backups() {
    local days="${1:-7}"
    
    log_info "Cleaning backups older than $days days..."
    
    if [[ ! -d "$BACKUP_DIR" ]]; then
        log_warning "Backup directory not found"
        return
    fi
    
    local count=0
    count=$(find "$BACKUP_DIR" -name "identity_*" -type f -mtime +"$days" | wc -l)
    
    if [[ $count -gt 0 ]]; then
        find "$BACKUP_DIR" -name "identity_*" -type f -mtime +"$days" -delete
        log_success "Removed $count old backup files"
    else
        log_info "No old backups to clean"
    fi
}

# Function to show help
show_help() {
    echo "Identity Service Backup and Recovery Script"
    echo ""
    echo "Usage: $0 [COMMAND] [OPTIONS]"
    echo ""
    echo "Commands:"
    echo "  backup-db           Create database backup"
    echo "  backup-config       Create configuration backup"
    echo "  backup-volumes      Create volume backup"
    echo "  backup-full         Create full backup (default)"
    echo "  restore-db FILE     Restore database from backup file"
    echo "  restore-volumes FILE Restore volumes from backup file"
    echo "  list                List available backups"
    echo "  clean [DAYS]        Clean backups older than DAYS (default: 7)"
    echo "  help                Show this help message"
    echo ""
    echo "Environment Variables:"
    echo "  BACKUP_DIR          Backup directory (default: ./backups)"
    echo ""
    echo "Examples:"
    echo "  $0 backup-full                           # Create full backup"
    echo "  $0 restore-db backups/identity_db_20240101_120000.sql.gz"
    echo "  $0 clean 30                              # Clean backups older than 30 days"
}

# Main function
main() {
    local command="${1:-backup-full}"
    
    case "$command" in
        "backup-db")
            backup_database
            ;;
        "backup-config")
            backup_config
            ;;
        "backup-volumes")
            backup_volumes
            ;;
        "backup-full")
            full_backup
            ;;
        "restore-db")
            if [[ -z "${2:-}" ]]; then
                log_error "Please specify backup file"
                echo "Usage: $0 restore-db <backup_file>"
                exit 1
            fi
            restore_database "$2"
            ;;
        "restore-volumes")
            if [[ -z "${2:-}" ]]; then
                log_error "Please specify backup file"
                echo "Usage: $0 restore-volumes <backup_file>"
                exit 1
            fi
            restore_volumes "$2"
            ;;
        "list")
            list_backups
            ;;
        "clean")
            clean_backups "${2:-7}"
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
