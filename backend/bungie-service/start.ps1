# Start Bungie Service with Environment Variables

# Change to script directory
Set-Location $PSScriptRoot

$env:BUNGIE_API_KEY = "38bb3aae6c2c466b8f9f0d20dd13c90c"
$env:BUNGIE_CLIENT_ID = "30139"
$env:BUNGIE_CLIENT_SECRET = "your_bungie_client_secret_here"
$env:PORT = "8082"
$env:ENVIRONMENT = "development"

Write-Host "Starting Bungie Service..." -ForegroundColor Green
Write-Host "Working Directory: $(Get-Location)" -ForegroundColor Cyan
Write-Host "PORT: $env:PORT" -ForegroundColor Yellow

go run .
