# Build script for bestdefense-device-monitor (Windows / PowerShell)
# Usage: .\scripts\build.ps1 [-Version "1.0.0"]

param(
    [string]$Version = "dev"
)

$ErrorActionPreference = "Stop"

# Detect git commit
$Commit = "unknown"
try {
    $Commit = (git rev-parse --short HEAD 2>$null)
} catch {}

$BuildDate = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")

$env:GOOS = "windows"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "0"

$OutputPath = "dist\bestdefense-device-monitor.exe"
New-Item -ItemType Directory -Force -Path "dist" | Out-Null

Write-Host "Building bestdefense-device-monitor $Version (commit: $Commit)..."

$LdFlags = "-s -w -H windowsgui " +
           "-X main.Version=$Version " +
           "-X main.BuildCommit=$Commit " +
           "-X `"main.BuildDate=$BuildDate`""

go build `
    -ldflags $LdFlags `
    -o $OutputPath `
    .\cmd\bestdefense-device-monitor

if ($LASTEXITCODE -ne 0) {
    Write-Error "Build failed"
    exit 1
}

$Size = (Get-Item $OutputPath).Length / 1MB
Write-Host "Built: $OutputPath ($([math]::Round($Size, 1)) MB)"
Write-Host ""
Write-Host "To install on this machine (requires elevation):"
Write-Host "  .\$OutputPath install --key YOUR_REGISTRATION_KEY"
Write-Host ""
Write-Host "To audit what data is collected (no install required):"
Write-Host "  .\$OutputPath check"
