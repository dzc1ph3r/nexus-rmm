#!/bin/bash
#
# Nexus RMM Deployment Script
#
# Usage: ./deploy.sh [environment]
# Environment: development, staging, production (default: production)

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

ENVIRONMENT="${1:-production}"
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DOCKER_DIR="$PROJECT_ROOT/docker"

print_header() {
    echo -e "${CYAN}=========================================="
    echo -e "  Nexus RMM Deployment Script"
    echo -e "  Environment: $ENVIRONMENT"
    echo -e "==========================================${NC}"
    echo
}

print_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
print_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
print_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
print_error() { echo -e "${RED}[ERROR]${NC} $1"; }

check_requirements() {
    print_info "Checking requirements..."

    local missing=()

    command -v docker >/dev/null 2>&1 || missing+=("docker")
    command -v docker-compose >/dev/null 2>&1 || missing+=("docker-compose")
    command -v openssl >/dev/null 2>&1 || missing+=("openssl")

    if [[ ${#missing[@]} -gt 0 ]]; then
        print_error "Missing required tools: ${missing[*]}"
        exit 1
    fi

    # Check Docker daemon
    if ! docker info >/dev/null 2>&1; then
        print_error "Docker daemon is not running"
        exit 1
    fi

    print_success "All requirements met"
}

setup_environment() {
    print_info "Setting up environment..."

    cd "$DOCKER_DIR"

    # Create .env if it doesn't exist
    if [[ ! -f .env ]]; then
        if [[ -f .env.example ]]; then
            cp .env.example .env
            print_warning "Created .env from .env.example - PLEASE EDIT IT WITH YOUR VALUES"
        else
            print_error ".env.example not found"
            exit 1
        fi
    fi

    # Generate secure passwords if not set
    if grep -q "your-secure-postgres-password-here" .env 2>/dev/null; then
        local pg_pass=$(openssl rand -base64 32 | tr -d "=+/" | cut -c1-32)
        sed -i "s/your-secure-postgres-password-here/$pg_pass/" .env
        print_info "Generated PostgreSQL password"
    fi

    if grep -q "your-secure-redis-password-here" .env 2>/dev/null; then
        local redis_pass=$(openssl rand -base64 32 | tr -d "=+/" | cut -c1-32)
        sed -i "s/your-secure-redis-password-here/$redis_pass/" .env
        print_info "Generated Redis password"
    fi

    if grep -q "your-256-bit-jwt-secret-here" .env 2>/dev/null; then
        local jwt_secret=$(openssl rand -base64 32)
        sed -i "s/your-256-bit-jwt-secret-here/$jwt_secret/" .env
        print_info "Generated JWT secret"
    fi

    # Create cert directories
    mkdir -p certbot/www certbot/conf certs

    print_success "Environment configured"
}

generate_self_signed_cert() {
    print_info "Generating self-signed certificate for development..."

    mkdir -p certs

    if [[ ! -f certs/fullchain.pem ]] || [[ ! -f certs/privkey.pem ]]; then
        openssl req -x509 -nodes -newkey rsa:2048 -days 365 \
            -keyout certs/privkey.pem \
            -out certs/fullchain.pem \
            -subj "/CN=localhost" \
            -addext "subjectAltName=DNS:localhost,DNS:rmm.mezfashion.pro,IP:127.0.0.1" \
            2>/dev/null
        print_success "Self-signed certificate generated"
    else
        print_info "Certificate already exists"
    fi
}

build_images() {
    print_info "Building Docker images..."

    cd "$DOCKER_DIR"

    if [[ "$ENVIRONMENT" == "production" ]]; then
        docker-compose -f docker-compose.prod.yml build --no-cache
    else
        docker-compose build --no-cache
    fi

    print_success "Images built successfully"
}

start_services() {
    print_info "Starting services..."

    cd "$DOCKER_DIR"

    if [[ "$ENVIRONMENT" == "production" ]]; then
        docker-compose -f docker-compose.prod.yml up -d
    else
        docker-compose up -d
    fi

    print_success "Services started"
}

wait_for_health() {
    print_info "Waiting for services to be healthy..."

    local max_attempts=60
    local attempt=0

    while [[ $attempt -lt $max_attempts ]]; do
        local healthy=true

        # Check each service using docker inspect
        for service in postgres redis backend gateway frontend; do
            local container_name="nexus-rmm-$service"
            local status
            status=$(docker inspect --format='{{.State.Health.Status}}' "$container_name" 2>/dev/null || echo "unknown")

            if [[ "$status" != "healthy" ]]; then
                healthy=false
                break
            fi
        done

        if [[ "$healthy" == "true" ]]; then
            print_success "All services are healthy"
            return 0
        fi

        attempt=$((attempt + 1))
        echo -n "."
        sleep 5
    done

    echo
    print_warning "Timeout waiting for services to be healthy"
    print_info "Check logs with: docker-compose logs -f"
    return 1
}

run_migrations() {
    print_info "Running database migrations..."

    cd "$DOCKER_DIR"

    if [[ "$ENVIRONMENT" == "production" ]]; then
        docker-compose -f docker-compose.prod.yml exec -T backend ./nexus-rmm migrate up
    else
        docker-compose exec -T backend ./nexus-rmm migrate up
    fi

    print_success "Migrations completed"
}

show_summary() {
    echo
    echo -e "${CYAN}=========================================="
    echo -e "  Deployment Complete!"
    echo -e "==========================================${NC}"
    echo

    if [[ "$ENVIRONMENT" == "production" ]]; then
        echo -e "Web Console: ${GREEN}https://rmm.mezfashion.pro${NC}"
        echo -e "API:         ${GREEN}https://rmm.mezfashion.pro/api${NC}"
        echo -e "WebSocket:   ${GREEN}wss://rmm.mezfashion.pro/ws${NC}"
    else
        echo -e "Web Console: ${GREEN}http://localhost${NC}"
        echo -e "API:         ${GREEN}http://localhost:8080/api${NC}"
        echo -e "WebSocket:   ${GREEN}ws://localhost:8443/ws${NC}"
    fi

    echo
    echo -e "${YELLOW}Default Credentials:${NC}"
    echo -e "  Email:    admin@nexus-rmm.local"
    echo -e "  Password: admin123"
    echo
    echo -e "${RED}IMPORTANT: Change the default password immediately after first login!${NC}"
    echo
    echo -e "${YELLOW}Useful commands:${NC}"
    if [[ "$ENVIRONMENT" == "production" ]]; then
        echo -e "  View logs:     ${GRAY}docker-compose -f docker-compose.prod.yml logs -f${NC}"
        echo -e "  Stop:          ${GRAY}docker-compose -f docker-compose.prod.yml down${NC}"
        echo -e "  Restart:       ${GRAY}docker-compose -f docker-compose.prod.yml restart${NC}"
    else
        echo -e "  View logs:     ${GRAY}docker-compose logs -f${NC}"
        echo -e "  Stop:          ${GRAY}docker-compose down${NC}"
        echo -e "  Restart:       ${GRAY}docker-compose restart${NC}"
    fi
    echo
}

# Main
main() {
    print_header
    check_requirements
    setup_environment

    if [[ "$ENVIRONMENT" != "production" ]]; then
        generate_self_signed_cert
    fi

    build_images
    start_services
    wait_for_health
    run_migrations
    show_summary
}

main "$@"