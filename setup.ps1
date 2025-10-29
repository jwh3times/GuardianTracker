#!/usr/bin/env pwsh
# Guardian Tracker - Environment Setup Script
# This script helps set up environment files for all services

Write-Host "🚀 Guardian Tracker - Environment Setup" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

$ErrorActionPreference = "Stop"

# Function to copy .env.example to .env if it doesn't exist
function Setup-EnvFile {
    param(
        [string]$Path,
        [string]$ServiceName
    )
    
    $envExample = Join-Path $Path ".env.example"
    $envFile = Join-Path $Path ".env"
    
    if (Test-Path $envExample) {
        if (Test-Path $envFile) {
            Write-Host "✓ $ServiceName .env already exists" -ForegroundColor Green
        } else {
            Copy-Item $envExample $envFile
            Write-Host "✓ Created $ServiceName .env file" -ForegroundColor Green
        }
    } else {
        Write-Host "⚠ Warning: $ServiceName .env.example not found at $envExample" -ForegroundColor Yellow
    }
}

# Root directory
Write-Host "Setting up root configuration..." -ForegroundColor Yellow
Setup-EnvFile -Path "." -ServiceName "Root"

# Auth Service
Write-Host "Setting up Auth Service..." -ForegroundColor Yellow
Setup-EnvFile -Path "backend/auth-service" -ServiceName "Auth Service"

# Bungie Service
Write-Host "Setting up Bungie Service..." -ForegroundColor Yellow
Setup-EnvFile -Path "backend/bungie-service" -ServiceName "Bungie Service"

# GraphQL Service
Write-Host "Setting up GraphQL Service..." -ForegroundColor Yellow
Setup-EnvFile -Path "backend/graphql-service" -ServiceName "GraphQL Service"

# Frontend (use .env.local for React)
Write-Host "Setting up Frontend..." -ForegroundColor Yellow
$frontendEnvExample = "frontend/.env.example"
$frontendEnvLocal = "frontend/.env.local"

if (Test-Path $frontendEnvExample) {
    if (Test-Path $frontendEnvLocal) {
        Write-Host "✓ Frontend .env.local already exists" -ForegroundColor Green
    } else {
        Copy-Item $frontendEnvExample $frontendEnvLocal
        Write-Host "✓ Created Frontend .env.local file" -ForegroundColor Green
    }
} else {
    Write-Host "⚠ Warning: Frontend .env.example not found" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "✅ Environment setup complete!" -ForegroundColor Green
Write-Host ""
Write-Host "📝 IMPORTANT NEXT STEPS:" -ForegroundColor Yellow
Write-Host ""
Write-Host "1. Get your Bungie API credentials:" -ForegroundColor White
Write-Host "   → Visit: https://www.bungie.net/en/Application" -ForegroundColor Cyan
Write-Host "   → Create a new application" -ForegroundColor Cyan
Write-Host "   → Copy: API Key, Client ID, and Client Secret" -ForegroundColor Cyan
Write-Host ""
Write-Host "2. Update the following .env files with your credentials:" -ForegroundColor White
Write-Host "   → backend/auth-service/.env" -ForegroundColor Cyan
Write-Host "   → backend/bungie-service/.env" -ForegroundColor Cyan
Write-Host "   → backend/graphql-service/.env" -ForegroundColor Cyan
Write-Host "   → frontend/.env.local" -ForegroundColor Cyan
Write-Host ""
Write-Host "3. Generate a secure JWT secret (minimum 32 characters)" -ForegroundColor White
Write-Host "   → Use the same secret in auth-service and graphql-service" -ForegroundColor Cyan
Write-Host ""
Write-Host "4. Review the full setup guide:" -ForegroundColor White
Write-Host "   → Read: ENVIRONMENT_SETUP.md" -ForegroundColor Cyan
Write-Host ""
Write-Host "5. After configuration, start the services:" -ForegroundColor White
Write-Host "   → Run: docker-compose up" -ForegroundColor Cyan
Write-Host "   → Or run each service individually" -ForegroundColor Cyan
Write-Host ""
Write-Host "Need help? Check ENVIRONMENT_SETUP.md for detailed instructions!" -ForegroundColor Yellow
Write-Host ""
