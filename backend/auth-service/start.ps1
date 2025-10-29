# Start Auth Service with Environment Variables

# Change to script directory
Set-Location $PSScriptRoot

$env:JWT_SECRET = "dev_jwt_secret_minimum_32_characters_change_in_production_12345"
$env:BUNGIE_CLIENT_ID = "30139"
$env:BUNGIE_API_KEY = "38bb3aae6c2c466b8f9f0d20dd13c90c"
$env:BUNGIE_CLIENT_SECRET = "your_bungie_client_secret_here"
$env:PORT = "8081"
$env:ENVIRONMENT = "development"

Write-Host "Starting Auth Service..." -ForegroundColor Green
Write-Host "Working Directory: $(Get-Location)" -ForegroundColor Cyan
Write-Host "JWT_SECRET: $($env:JWT_SECRET.Substring(0, 20))..." -ForegroundColor Yellow
Write-Host "PORT: $env:PORT" -ForegroundColor Yellow

go run .
