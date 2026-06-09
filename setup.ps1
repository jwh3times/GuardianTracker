#!/usr/bin/env pwsh
# Guardian Tracker - Environment Setup Script

Write-Host "Guardian Tracker - Environment Setup" -ForegroundColor Cyan
Write-Host "=====================================" -ForegroundColor Cyan
Write-Host ""

$ErrorActionPreference = "Stop"

function Setup-EnvFile {
    param(
        [string]$Path,
        [string]$SourceFile,
        [string]$DestFile,
        [string]$ServiceName
    )

    $src = Join-Path $Path $SourceFile
    $dst = Join-Path $Path $DestFile

    if (Test-Path $src) {
        if (Test-Path $dst) {
            Write-Host "  $ServiceName $DestFile already exists" -ForegroundColor Green
        } else {
            Copy-Item $src $dst
            Write-Host "  Created $ServiceName $DestFile" -ForegroundColor Green
        }
    } else {
        Write-Host "  Warning: $ServiceName $SourceFile not found at $src" -ForegroundColor Yellow
    }
}

# Root (docker-compose env)
Write-Host "Setting up root configuration..." -ForegroundColor Yellow
Setup-EnvFile -Path "." -SourceFile ".env.example" -DestFile ".env" -ServiceName "Root"

# API Service
Write-Host "Setting up API Service..." -ForegroundColor Yellow
Setup-EnvFile -Path "backend/api-service" -SourceFile ".env.example" -DestFile ".env" -ServiceName "API Service"

# Frontend
Write-Host "Setting up Frontend..." -ForegroundColor Yellow
Setup-EnvFile -Path "frontend" -SourceFile ".env.example" -DestFile ".env.local" -ServiceName "Frontend"

Write-Host ""
Write-Host "=====================================" -ForegroundColor Cyan
Write-Host "Environment setup complete!" -ForegroundColor Green
Write-Host ""
Write-Host "NEXT STEPS:" -ForegroundColor Yellow
Write-Host ""
Write-Host "1. Get Bungie API credentials at https://www.bungie.net/en/Application" -ForegroundColor White
Write-Host "   Copy: API Key, Client ID, Client Secret" -ForegroundColor Cyan
Write-Host ""
Write-Host "2. Fill in credentials in:" -ForegroundColor White
Write-Host "   -> .env (root, used by docker-compose)" -ForegroundColor Cyan
Write-Host "   -> backend/api-service/.env (for running the service individually)" -ForegroundColor Cyan
Write-Host "   -> frontend/.env.local (VITE_API_URL etc.)" -ForegroundColor Cyan
Write-Host ""
Write-Host "3. Generate a secure JWT secret (minimum 32 characters):" -ForegroundColor White
Write-Host "   openssl rand -base64 32" -ForegroundColor Cyan
Write-Host ""
Write-Host "4. Start the full stack:" -ForegroundColor White
Write-Host "   docker compose up --build" -ForegroundColor Cyan
Write-Host ""
