# Guardian Tracker - Full Startup Script
# This script starts Minikube, builds Docker images, deploys all services, and sets up port forwarding

Write-Host "=== Guardian Tracker Startup Script ===" -ForegroundColor Green
Write-Host "Starting full deployment process..." -ForegroundColor Yellow

# Step 1: Start Minikube
Write-Host "`n1. Starting Minikube..." -ForegroundColor Cyan
try {
    $minikubeStatus = minikube status 2>$null
    if ($LASTEXITCODE -eq 0 -and $minikubeStatus -match "Running") {
        Write-Host "   Minikube is already running" -ForegroundColor Green
    } else {
        Write-Host "   Starting Minikube cluster..." -ForegroundColor Yellow
        minikube start
        if ($LASTEXITCODE -ne 0) {
            Write-Host "   ERROR: Failed to start Minikube" -ForegroundColor Red
            exit 1
        }
        Write-Host "   Minikube started successfully" -ForegroundColor Green
    }
} catch {
    Write-Host "   ERROR: Failed to check/start Minikube: $_" -ForegroundColor Red
    exit 1
}

# Step 2: Build Docker Images
Write-Host "`n2. Building Docker images..." -ForegroundColor Cyan

# Set Docker environment to use Minikube's Docker daemon
Write-Host "   Setting Docker environment to Minikube..." -ForegroundColor Yellow
& minikube docker-env --shell powershell | Invoke-Expression

$services = @(
    @{Name="api-service"; Path="../backend/api-service"; Tag="guardian-tracker/api-service:latest"},
    @{Name="frontend"; Path="../frontend"; Tag="guardian-tracker/frontend:v2"}
)

foreach ($service in $services) {
    Write-Host "   Building $($service.Name)..." -ForegroundColor Yellow
    Push-Location $service.Path
    try {
        docker build -t $service.Tag .
        if ($LASTEXITCODE -ne 0) {
            Write-Host "   ERROR: Failed to build $($service.Name)" -ForegroundColor Red
            Pop-Location
            exit 1
        }
        Write-Host "   $($service.Name) built successfully" -ForegroundColor Green
    } catch {
        Write-Host "   ERROR: Failed to build $($service.Name): $_" -ForegroundColor Red
        Pop-Location
        exit 1
    }
    Pop-Location
}

# Step 3: Deploy Kubernetes Services
Write-Host "`n3. Deploying Kubernetes services..." -ForegroundColor Cyan

$manifests = @(
    "api-service-configmap.yaml",
    "api-service-secret.yaml",
    "api-service.yaml",
    "frontend.yaml"
)

foreach ($manifest in $manifests) {
    Write-Host "   Applying $manifest..." -ForegroundColor Yellow
    kubectl apply -f $manifest
    if ($LASTEXITCODE -ne 0) {
        Write-Host "   ERROR: Failed to apply $manifest" -ForegroundColor Red
        exit 1
    }
}

Write-Host "   All manifests applied successfully" -ForegroundColor Green

# Step 4: Wait for deployments to be ready
Write-Host "`n4. Waiting for deployments to be ready..." -ForegroundColor Cyan

$deployments = @("api-service", "frontend")

foreach ($deployment in $deployments) {
    Write-Host "   Waiting for $deployment..." -ForegroundColor Yellow
    kubectl rollout status deployment/$deployment --timeout=300s
    if ($LASTEXITCODE -ne 0) {
        Write-Host "   ERROR: $deployment failed to deploy" -ForegroundColor Red
        exit 1
    }
    Write-Host "   $deployment is ready" -ForegroundColor Green
}

# Step 5: Set up port forwarding
Write-Host "`n5. Setting up port forwarding..." -ForegroundColor Cyan

# Kill any existing port forwarding on port 3000
Write-Host "   Cleaning up existing port forwards..." -ForegroundColor Yellow
Get-Process | Where-Object {$_.ProcessName -eq "kubectl" -and $_.CommandLine -like "*port-forward*"} | Stop-Process -Force -ErrorAction SilentlyContinue

# Wait a moment for cleanup
Start-Sleep -Seconds 2

# Start new port forwarding in background
Write-Host "   Starting port forward to frontend (localhost:3000 -> frontend:80)..." -ForegroundColor Yellow
$portForwardJob = Start-Job -ScriptBlock {
    kubectl port-forward service/frontend 3000:80
}

# Wait a moment for port forwarding to establish
Start-Sleep -Seconds 3

# Test if port forwarding is working
try {
    $response = Invoke-WebRequest -Uri "http://localhost:3000" -Method Head -TimeoutSec 5 -ErrorAction Stop
    Write-Host "   Port forwarding established successfully" -ForegroundColor Green
} catch {
    Write-Host "   WARNING: Port forwarding may not be ready yet. You can test manually." -ForegroundColor Yellow
}

# Step 6: Display status and next steps
Write-Host "`n=== Deployment Complete ===" -ForegroundColor Green
Write-Host ""
Write-Host "Guardian Tracker is now running!" -ForegroundColor Green
Write-Host ""
Write-Host "Services Status:" -ForegroundColor Cyan
kubectl get pods

Write-Host "`nService URLs:" -ForegroundColor Cyan
Write-Host "  Frontend: http://localhost:3000" -ForegroundColor White
Write-Host "  (Port forwarding job ID: $($portForwardJob.Id))" -ForegroundColor Gray

Write-Host "`nNext Steps:" -ForegroundColor Cyan
Write-Host "  1. Open http://localhost:3000 in your browser" -ForegroundColor White
Write-Host "  2. Update your ngrok tunnel if needed for OAuth" -ForegroundColor White
Write-Host "  3. Use shutdown.ps1 to stop all services when done" -ForegroundColor White

Write-Host "`nUseful Commands:" -ForegroundColor Cyan
Write-Host "  Check pods: kubectl get pods" -ForegroundColor White
Write-Host "  Check services: kubectl get services" -ForegroundColor White
Write-Host "  View logs: kubectl logs <pod-name>" -ForegroundColor White
Write-Host "  Stop port forwarding: Stop-Job $($portForwardJob.Id)" -ForegroundColor White

Write-Host "`n=== Startup Complete ===" -ForegroundColor Green
