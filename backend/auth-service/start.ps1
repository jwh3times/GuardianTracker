# Start Auth Service with environment variables loaded from .env
#
# This script reads credentials from a local .env file (which is gitignored).
# Before first run: copy .env.example to .env and fill in your real values.
#   cp .env.example .env

# Change to script directory
Set-Location $PSScriptRoot

$envFile = Join-Path $PSScriptRoot ".env"
if (-not (Test-Path $envFile)) {
    Write-Host "ERROR: .env not found in $PSScriptRoot" -ForegroundColor Red
    Write-Host "Copy .env.example to .env and fill in your credentials:" -ForegroundColor Yellow
    Write-Host "  cp .env.example .env" -ForegroundColor Cyan
    exit 1
}

# Load KEY=VALUE pairs from .env into the process environment.
# Lines that are blank or start with '#' are ignored; surrounding quotes are stripped.
Get-Content $envFile | ForEach-Object {
    $line = $_.Trim()
    if ($line -and -not $line.StartsWith("#")) {
        $parts = $line -split "=", 2
        if ($parts.Count -eq 2) {
            $key = $parts[0].Trim()
            $value = $parts[1].Trim().Trim('"').Trim("'")
            Set-Item -Path "Env:$key" -Value $value
        }
    }
}

Write-Host "Starting Auth Service..." -ForegroundColor Green
Write-Host "Working Directory: $(Get-Location)" -ForegroundColor Cyan
Write-Host "PORT: $env:PORT" -ForegroundColor Yellow

go run .
