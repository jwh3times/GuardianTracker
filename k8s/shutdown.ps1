# Guardian Tracker - Full Shutdown Script
# This script stops all services, cleans up Kubernetes resources, and stops Minikube

Write-Host "=== Guardian Tracker Shutdown Script ===" -ForegroundColor Red
Write-Host "Starting shutdown process..." -ForegroundColor Yellow

# Ensure we're in the correct directory
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $scriptDir
Write-Host "Working from directory: $scriptDir" -ForegroundColor Gray

# Step 1: Stop port forwarding
Write-Host "`n1. Stopping port forwarding..." -ForegroundColor Cyan

# Stop any kubectl port-forward processes
Write-Host "   Stopping port-forward processes..." -ForegroundColor Yellow
$portForwardProcesses = Get-Process | Where-Object {$_.ProcessName -eq "kubectl" -and $_.CommandLine -like "*port-forward*"}
if ($portForwardProcesses) {
    $portForwardProcesses | ForEach-Object {
        Write-Host "   Stopping port-forward process (PID: $($_.Id))" -ForegroundColor Yellow
        Stop-Process -Id $_.Id -Force -ErrorAction SilentlyContinue
    }
} else {
    Write-Host "   No port-forward processes found" -ForegroundColor Gray
}

# Stop any background jobs
$portForwardJobs = Get-Job | Where-Object {$_.Command -like "*port-forward*"}
if ($portForwardJobs) {
    $portForwardJobs | ForEach-Object {
        Write-Host "   Stopping port-forward job (ID: $($_.Id))" -ForegroundColor Yellow
        Stop-Job -Id $_.Id -ErrorAction SilentlyContinue
        Remove-Job -Id $_.Id -ErrorAction SilentlyContinue
    }
} else {
    Write-Host "   No port-forward jobs found" -ForegroundColor Gray
}

Write-Host "   Port forwarding stopped" -ForegroundColor Green

# Step 2: Delete Kubernetes resources
Write-Host "`n2. Deleting Kubernetes resources..." -ForegroundColor Cyan

$manifests = @(
    "frontend.yaml",
    "graphql-service.yaml", 
    "bungie-service.yaml",
    "auth-service.yaml",
    "auth-service-configmap.yaml"
)

foreach ($manifest in $manifests) {
    if (Test-Path $manifest) {
        Write-Host "   Deleting resources from $manifest..." -ForegroundColor Yellow
        kubectl delete -f $manifest --ignore-not-found=true
        if ($LASTEXITCODE -ne 0) {
            Write-Host "   WARNING: Failed to delete $manifest (may not exist)" -ForegroundColor Yellow
        } else {
            Write-Host "   Successfully deleted $manifest" -ForegroundColor Green
        }
    } else {
        Write-Host "   Skipping $manifest (file not found)" -ForegroundColor Yellow
    }
}

# Wait for pods to terminate
Write-Host "   Waiting for pods to terminate..." -ForegroundColor Yellow
$timeout = 60
$elapsed = 0
do {
    $pods = kubectl get pods --no-headers 2>$null | Where-Object {$_ -notmatch "No resources found"}
    if (-not $pods) {
        Write-Host "   All pods terminated" -ForegroundColor Green
        break
    }
    Start-Sleep -Seconds 2
    $elapsed += 2
    if ($elapsed % 10 -eq 0) {
        Write-Host "   Still waiting for pods to terminate... ($elapsed seconds)" -ForegroundColor Yellow
    }
} while ($elapsed -lt $timeout)

if ($elapsed -ge $timeout) {
    Write-Host "   WARNING: Some pods may still be terminating" -ForegroundColor Yellow
}

Write-Host "   Kubernetes resources cleaned up" -ForegroundColor Green

# Step 3: Clean up Docker images (optional)
Write-Host "`n3. Docker cleanup..." -ForegroundColor Cyan

$cleanupChoice = Read-Host "   Do you want to remove Guardian Tracker Docker images? (y/N)"
if ($cleanupChoice -eq 'y' -or $cleanupChoice -eq 'Y') {
    Write-Host "   Setting Docker environment to Minikube..." -ForegroundColor Yellow
    try {
        & minikube docker-env --shell powershell | Invoke-Expression
        Write-Host "   Docker environment set to Minikube" -ForegroundColor Green
    } catch {
        Write-Host "   WARNING: Failed to set Docker environment. Minikube may not be running." -ForegroundColor Yellow
        Write-Host "   Skipping Docker cleanup." -ForegroundColor Yellow
        return
    }
    
    Write-Host "   Removing Guardian Tracker Docker images..." -ForegroundColor Yellow
    
    $images = @(
        "guardian-tracker/auth-service:latest",
        "guardian-tracker/bungie-service:latest", 
        "guardian-tracker/graphql-service:latest",
        "guardian-tracker/frontend:v2",
        "guardian-tracker/frontend:latest"
    )
    
    foreach ($image in $images) {
        docker rmi $image 2>$null
        if ($LASTEXITCODE -eq 0) {
            Write-Host "   Removed $image" -ForegroundColor Green
        } else {
            Write-Host "   $image not found (already removed)" -ForegroundColor Yellow
        }
    }
    
    # Clean up dangling images
    Write-Host "   Cleaning up dangling images..." -ForegroundColor Yellow
    docker image prune -f >$null 2>&1
    
    Write-Host "   Docker cleanup complete" -ForegroundColor Green
} else {
    Write-Host "   Skipping Docker image cleanup" -ForegroundColor Yellow
}

# Step 4: Stop Minikube
Write-Host "`n4. Stopping Minikube..." -ForegroundColor Cyan

$minikubeChoice = Read-Host "   Do you want to stop Minikube cluster? (Y/n)"
if ($minikubeChoice -ne 'n' -and $minikubeChoice -ne 'N') {
    Write-Host "   Stopping Minikube cluster..." -ForegroundColor Yellow
    minikube stop
    if ($LASTEXITCODE -eq 0) {
        Write-Host "   Minikube stopped successfully" -ForegroundColor Green
    } else {
        Write-Host "   WARNING: Failed to stop Minikube or it was already stopped" -ForegroundColor Yellow
    }
} else {
    Write-Host "   Leaving Minikube running" -ForegroundColor Yellow
}

# Step 5: Final status
Write-Host "`n=== Shutdown Complete ===" -ForegroundColor Red
Write-Host ""

# Check if anything is still running
Write-Host "Remaining Kubernetes resources:" -ForegroundColor Cyan
$remainingPods = kubectl get pods --no-headers 2>$null
if ($remainingPods -and $remainingPods -notmatch "No resources found") {
    kubectl get pods
} else {
    Write-Host "  No pods running" -ForegroundColor Green
}

Write-Host "`nMinikube status:" -ForegroundColor Cyan
try {
    $minikubeStatus = minikube status 2>$null
    if ($LASTEXITCODE -eq 0) {
        Write-Host "  $minikubeStatus" -ForegroundColor White
    } else {
        Write-Host "  Minikube is stopped" -ForegroundColor Green
    }
} catch {
    Write-Host "  Minikube is stopped" -ForegroundColor Green
}

Write-Host "`nNext Steps:" -ForegroundColor Cyan
Write-Host "  - Run startup.ps1 or startup.bat to restart Guardian Tracker" -ForegroundColor White
Write-Host "  - Run 'minikube delete' to completely remove the cluster" -ForegroundColor White
Write-Host "  - Double-click startup.bat for easy restart" -ForegroundColor White

Write-Host "`n=== Guardian Tracker Shutdown Complete ===" -ForegroundColor Red
