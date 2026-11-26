<#
.SYNOPSIS
    Nexus RMM Deployment Script for Windows RDP

.DESCRIPTION
    Deploys Nexus RMM on Windows Server using Docker Desktop or Docker Engine.
    Configures for domain rmm.mezfashion.pro with SSL.

.PARAMETER Environment
    Deployment environment: Development, Staging, Production (default: Production)

.PARAMETER Domain
    Domain name for SSL certificates (default: rmm.mezfashion.pro)

.PARAMETER Email
    Email for Let's Encrypt certificates

.PARAMETER SkipBuild
    Skip building Docker images (use existing)

.PARAMETER SkipCert
    Skip SSL certificate generation (use existing)

.EXAMPLE
    .\deploy-windows.ps1 -Environment Production -Domain "rmm.mezfashion.pro" -Email "admin@mezfashion.pro"

.EXAMPLE
    .\deploy-windows.ps1 -Environment Development
#>

[CmdletBinding()]
param(
    [ValidateSet("Development", "Staging", "Production")]
    [string]$Environment = "Production",

    [string]$Domain = "rmm.mezfashion.pro",

    [string]$Email = "admin@mezfashion.pro",

    [switch]$SkipBuild,

    [switch]$SkipCert
)

# Check for Administrator privileges
if (-NOT ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] "Administrator")) {
    Write-Error "This script requires Administrator privileges. Please run PowerShell as Administrator."
    exit 1
}

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

Write-Host "==========================================" -ForegroundColor Cyan
Write-Host "  Nexus RMM Windows Deployment" -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan
Write-Host "Environment: $Environment" -ForegroundColor Yellow
Write-Host "Domain: $Domain" -ForegroundColor Yellow
Write-Host ""

# Get script directory
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Definition
$ProjectRoot = Split-Path -Parent $ScriptDir
$DockerDir = Join-Path $ProjectRoot "docker"

# Check requirements
Write-Host "Checking requirements..." -ForegroundColor Blue

$missing = @()
$tools = @("docker", "docker-compose", "openssl")
foreach ($tool in $tools) {
    if (-not (Get-Command $tool -ErrorAction SilentlyContinue)) {
        $missing += $tool
    }
}

if ($missing.Count -gt 0) {
    Write-Error "Missing required tools: $($missing -join ', ')"
    Write-Host "Please install Docker Desktop for Windows: https://docs.docker.com/desktop/install/windows-install/" -ForegroundColor Yellow
    exit 1
}

# Check Docker daemon
try {
    docker info | Out-Null
    Write-Host "Docker daemon is running" -ForegroundColor Green
} catch {
    Write-Error "Docker daemon is not running. Please start Docker Desktop."
    exit 1
}

# Change to docker directory
Set-Location $DockerDir

# Setup environment file
Write-Host "Setting up environment..." -ForegroundColor Blue

$envFile = Join-Path $DockerDir ".env"
$envExample = Join-Path $DockerDir ".env.example"

if (-not (Test-Path $envFile)) {
    if (Test-Path $envExample) {
        Copy-Item $envExample $envFile
        Write-Host "Created .env from .env.example" -ForegroundColor Yellow
    } else {
        Write-Error ".env.example not found"
        exit 1
    }
}

# Generate secure passwords if needed
$envContent = Get-Content $envFile -Raw

if ($envContent -match "your-secure-postgres-password-here") {
    $pgPass = [System.Convert]::ToBase64String((1..32 | ForEach-Object { Get-Random -Maximum 256 })) -replace '[=+/]', '' -replace '.{32}.*', '$&'
    $envContent = $envContent -replace "your-secure-postgres-password-here", $pgPass
    Write-Host "Generated PostgreSQL password" -ForegroundColor Green
}

if ($envContent -match "your-secure-redis-password-here") {
    $redisPass = [System.Convert]::ToBase64String((1..32 | ForEach-Object { Get-Random -Maximum 256 })) -replace '[=+/]', '' -replace '.{32}.*', '$&'
    $envContent = $envContent -replace "your-secure-redis-password-here", $redisPass
    Write-Host "Generated Redis password" -ForegroundColor Green
}

if ($envContent -match "your-256-bit-jwt-secret-here") {
    $jwtSecret = [System.Convert]::ToBase64String((1..32 | ForEach-Object { Get-Random -Maximum 256 }))
    $envContent = $envContent -replace "your-256-bit-jwt-secret-here", $jwtSecret
    Write-Host "Generated JWT secret" -ForegroundColor Green
}

# Update domain and email
$envContent = $envContent -replace "rmm\.mezfashion\.pro", $Domain
$envContent = $envContent -replace "admin@mezfashion\.pro", $Email

Set-Content -Path $envFile -Value $envContent -Encoding UTF8

# Create cert directories
$certDirs = @("certbot/www", "certbot/conf", "certs")
foreach ($dir in $certDirs) {
    $fullPath = Join-Path $DockerDir $dir
    if (-not (Test-Path $fullPath)) {
        New-Item -ItemType Directory -Path $fullPath -Force | Out-Null
    }
}

# Generate self-signed cert for development
if ($Environment -ne "Production" -and -not $SkipCert) {
    Write-Host "Generating self-signed certificate..." -ForegroundColor Blue
    $certPath = Join-Path $DockerDir "certs"
    $keyFile = Join-Path $certPath "privkey.pem"
    $crtFile = Join-Path $certPath "fullchain.pem"

    if (-not (Test-Path $keyFile) -or -not (Test-Path $crtFile)) {
        & openssl req -x509 -nodes -newkey rsa:2048 -days 365 `
            -keyout $keyFile `
            -out $crtFile `
            -subj "/CN=localhost" `
            -addext "subjectAltName=DNS:localhost,DNS:$Domain,IP:127.0.0.1" 2>$null
        Write-Host "Self-signed certificate generated" -ForegroundColor Green
    } else {
        Write-Host "Certificate already exists" -ForegroundColor Yellow
    }
}

# Build images
if (-not $SkipBuild) {
    Write-Host "Building Docker images..." -ForegroundColor Blue

    $composeFile = if ($Environment -eq "Production") { "docker-compose.prod.yml" } else { "docker-compose.yml" }
    $composePath = Join-Path $DockerDir $composeFile

    if (-not (Test-Path $composePath)) {
        Write-Error "Compose file not found: $composePath"
        exit 1
    }

    docker-compose -f $composePath build --no-cache
    Write-Host "Images built successfully" -ForegroundColor Green
}

# Start services
Write-Host "Starting services..." -ForegroundColor Blue

$composeFile = if ($Environment -eq "Production") { "docker-compose.prod.yml" } else { "docker-compose.yml" }
$composePath = Join-Path $DockerDir $composeFile

docker-compose -f $composePath up -d

Write-Host "Services started" -ForegroundColor Green

# Wait for health
Write-Host "Waiting for services to be healthy..." -ForegroundColor Blue

$maxAttempts = 60
$attempt = 0
$healthy = $false

while ($attempt -lt $maxAttempts) {
    $allHealthy = $true

    $services = @("postgres", "redis", "backend", "gateway", "frontend")
    foreach ($service in $services) {
        $containerName = "nexus-rmm-$service"
        # Check container state - first check if running, then check health if available
        $containerState = docker inspect --format='{{.State.Status}} {{.State.Health.Status}}' $containerName 2>$null
        if (-not $containerState) {
            $allHealthy = $false
            break
        }
        $parts = $containerState.Split(' ')
        $status = $parts[0]
        $healthStatus = if ($parts.Count -gt 1) { $parts[1] } else { "none" }

        if ($status -ne "running") {
            $allHealthy = $false
            break
        }
        # If health check exists, require it to be healthy
        if ($healthStatus -ne "none" -and $healthStatus -ne "healthy") {
            $allHealthy = $false
            break
        }
    }

    if ($allHealthy) {
        $healthy = $true
        break
    }

    $attempt++
    Write-Host "." -NoNewline
    Start-Sleep -Seconds 5
}

Write-Host ""

if ($healthy) {
    Write-Host "All services are healthy/running" -ForegroundColor Green
} else {
    Write-Warning "Timeout waiting for services to be healthy"
    Write-Host "Check logs with: docker-compose -f $composeFile logs -f" -ForegroundColor Yellow
}

# Run migrations
Write-Host "Running database migrations..." -ForegroundColor Blue

$containerName = "nexus-rmm-backend"
docker exec $containerName ./nexus-rmm migrate up

Write-Host "Migrations completed" -ForegroundColor Green

# Show summary
Write-Host ""
Write-Host "==========================================" -ForegroundColor Cyan
Write-Host "  Deployment Complete!" -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan
Write-Host ""

if ($Environment -eq "Production") {
    Write-Host "Web Console: https://$Domain" -ForegroundColor Green
    Write-Host "API:         https://$Domain/api" -ForegroundColor Green
    Write-Host "WebSocket:   wss://$Domain/ws" -ForegroundColor Green
} else {
    Write-Host "Web Console: http://localhost" -ForegroundColor Green
    Write-Host "API:         http://localhost:8080/api" -ForegroundColor Green
    Write-Host "WebSocket:   ws://localhost:8443/ws" -ForegroundColor Green
}

Write-Host ""
Write-Host "Default Credentials:" -ForegroundColor Yellow
Write-Host "  Email:    admin@nexus-rmm.local" -ForegroundColor Gray
Write-Host "  Password: admin123" -ForegroundColor Gray
Write-Host ""
Write-Host "IMPORTANT: Change the default password immediately after first login!" -ForegroundColor Red
Write-Host ""
Write-Host "Useful commands:" -ForegroundColor Yellow
Write-Host "  View logs:   docker-compose -f $composeFile logs -f" -ForegroundColor Gray
Write-Host "  Stop:        docker-compose -f $composeFile down" -ForegroundColor Gray
Write-Host "  Restart:     docker-compose -f $composeFile restart" -ForegroundColor Gray
Write-Host ""