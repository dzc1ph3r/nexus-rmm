<#
.SYNOPSIS
    Nexus RMM Agent Installer for Windows

.DESCRIPTION
    Downloads, installs, and configures the Nexus RMM agent as a Windows service.
    Requires Administrator privileges.

.PARAMETER ServerUrl
    The Nexus RMM server URL (e.g., https://rmm.mezfashion.pro)

.PARAMETER EnrollmentToken
    The enrollment token from the Nexus RMM web console

.PARAMETER InstallPath
    Installation directory (default: C:\Program Files\NexusRMM)

.PARAMETER ServiceName
    Windows service name (default: NexusRMMAgent)

.PARAMETER StartService
    Start the service after installation (default: $true)

.EXAMPLE
    .\install-agent.ps1 -ServerUrl "https://rmm.mezfashion.pro" -EnrollmentToken "abc123..."

.EXAMPLE
    .\install-agent.ps1 -ServerUrl "https://rmm.mezfashion.pro" -EnrollmentToken "abc123..." -InstallPath "D:\NexusRMM"
#>

[CmdletBinding()]
param(
    [Parameter(Mandatory=$true)]
    [string]$ServerUrl,

    [Parameter(Mandatory=$true)]
    [string]$EnrollmentToken,

    [string]$InstallPath = "C:\Program Files\NexusRMM",

    [string]$ServiceName = "NexusRMMAgent",

    [switch]$StartService = $true
)

# Check for Administrator privileges
if (-NOT ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] "Administrator")) {
    Write-Error "This script requires Administrator privileges. Please run PowerShell as Administrator."
    exit 1
}

# Set strict mode
Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

Write-Host "==========================================" -ForegroundColor Cyan
Write-Host "  Nexus RMM Agent Installer for Windows" -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan
Write-Host ""

# Validate inputs
if (-not $ServerUrl.StartsWith("http")) {
    Write-Error "ServerUrl must start with http:// or https://"
    exit 1
}

if ($EnrollmentToken.Length -lt 10) {
    Write-Error "EnrollmentToken appears to be invalid"
    exit 1
}

# Determine architecture
$is64bit = [Environment]::Is64BitOperatingSystem
$arch = if ($is64bit) { "amd64" } else { "386" }
Write-Host "Detected architecture: $arch" -ForegroundColor Green

# Create installation directory
Write-Host "Creating installation directory: $InstallPath" -ForegroundColor Yellow
if (-not (Test-Path $InstallPath)) {
    New-Item -ItemType Directory -Path $InstallPath -Force | Out-Null
}

# Download agent binary
$agentUrl = "$ServerUrl/api/v1/agent/download/windows/$arch"
$agentPath = Join-Path $InstallPath "nexus-agent.exe"

Write-Host "Downloading agent from: $agentUrl" -ForegroundColor Yellow
try {
    $wc = New-Object System.Net.WebClient
    $wc.Headers.Add("User-Agent", "NexusRMMAgentInstaller/1.0")
    $wc.DownloadFile($agentUrl, $agentPath)
    Write-Host "Download complete" -ForegroundColor Green
} catch {
    Write-Error "Failed to download agent: $_"
    exit 1
}

# Verify download
if (-not (Test-Path $agentPath)) {
    Write-Error "Agent binary not found after download"
    exit 1
}

# Create configuration file
$configPath = Join-Path $InstallPath "agent.yaml"
Write-Host "Creating configuration: $configPath" -ForegroundColor Yellow

$config = @"
server_url: "$ServerUrl"
enrollment_token: "$EnrollmentToken"
install_path: "$InstallPath"
log_level: "info"
log_file: "C:\ProgramData\NexusRMM\agent.log"
heartbeat_interval: 30s
command_timeout: 5m
max_concurrent_tasks: 5
auto_update: true
update_check_interval: 1h
"@

Set-Content -Path $configPath -Value $config -Encoding UTF8

# Create log directory
$logDir = "C:\ProgramData\NexusRMM"
if (-not (Test-Path $logDir)) {
    New-Item -ItemType Directory -Path $logDir -Force | Out-Null
}

# Install as Windows service
Write-Host "Installing Windows service: $ServiceName" -ForegroundColor Yellow

$serviceArgs = @(
    "create",
    $ServiceName,
    "binPath= `"$agentPath`" --config `"$configPath`" run-service",
    "DisplayName= Nexus RMM Agent",
    "Description= Nexus RMM Endpoint Agent",
    "start= auto",
    "depend= Tcpip"
)

$sc = & sc.exe @serviceArgs
if ($LASTEXITCODE -ne 0) {
    Write-Error "Failed to create service: $sc"
    exit 1
}

Write-Host "Service created successfully" -ForegroundColor Green

# Configure service recovery
Write-Host "Configuring service recovery..." -ForegroundColor Yellow
& sc.exe failure $ServiceName reset= 86400 actions= restart/60000/restart/120000/restart/180000
& sc.exe failureflag $ServiceName 1

# Set service description with PowerShell
$service = Get-WmiObject -Class Win32_Service -Filter "Name='$ServiceName'"
if ($service) {
    $service.Change($null, $null, $null, $null, $null, $null, "Nexus RMM Endpoint Agent", $null, $null, $null, $null)
}

# Start service if requested
if ($StartService) {
    Write-Host "Starting service..." -ForegroundColor Yellow
    $startResult = Start-Service -Name $ServiceName -ErrorAction SilentlyContinue
    if ($?) {
        Write-Host "Service started successfully" -ForegroundColor Green
    } else {
        Write-Warning "Service created but failed to start. Check logs at $logDir\agent.log"
    }
}

# Verify service status
Write-Host "" -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan
Write-Host "  Installation Complete!" -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "Service Name: $ServiceName" -ForegroundColor Green
Write-Host "Install Path: $InstallPath" -ForegroundColor Green
Write-Host "Config File:  $configPath" -ForegroundColor Green
Write-Host "Log File:     $logDir\agent.log" -ForegroundColor Green
Write-Host ""
Write-Host "Useful commands:" -ForegroundColor Yellow
Write-Host "  Start service:   Start-Service -Name $ServiceName" -ForegroundColor Gray
Write-Host "  Stop service:    Stop-Service -Name $ServiceName" -ForegroundColor Gray
Write-Host "  Restart service: Restart-Service -Name $ServiceName" -ForegroundColor Gray
Write-Host "  View logs:       Get-Content $logDir\agent.log -Wait" -ForegroundColor Gray
Write-Host "  Uninstall:       sc.exe delete $ServiceName" -ForegroundColor Gray
Write-Host ""
Write-Host "The agent will now enroll with the server and appear in the Nexus RMM console." -ForegroundColor Cyan