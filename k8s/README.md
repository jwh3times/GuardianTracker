# Guardian Tracker - Deployment Scripts

This directory contains scripts to easily start and stop the complete Guardian Tracker application on Minikube.

## Quick Start

### Option 1: Using Batch Files (Easiest)

- **Start:** Double-click `startup.bat`
- **Stop:** Double-click `shutdown.bat`

### Option 2: Using PowerShell Scripts Directly

- **Start:** `powershell.exe -ExecutionPolicy Bypass -File startup.ps1`
- **Stop:** `powershell.exe -ExecutionPolicy Bypass -File shutdown.ps1`

## What the Startup Script Does

1. **Starts Minikube** (if not already running)
2. **Builds Docker Images** for all services:
   - auth-service
   - bungie-service
   - graphql-service
   - frontend (with nginx configuration)
3. **Deploys Kubernetes Services** from manifests
4. **Waits for deployments** to be ready
5. **Sets up port forwarding** (localhost:3000 → frontend)
6. **Shows status** and provides next steps

After startup completes:

- Frontend available at: http://localhost:3000
- All services running in Minikube cluster
- Port forwarding active in background

## What the Shutdown Script Does

1. **Stops port forwarding** processes and jobs
2. **Deletes Kubernetes resources** (pods, services, deployments)
3. **Optionally removes Docker images** (saves disk space)
4. **Optionally stops Minikube** (preserves cluster by default)
5. **Shows final status**

## Important Notes

### OAuth Configuration

- Remember to update your ngrok tunnel URL in `auth-service-configmap.yaml` if needed
- The current OAuth redirect URI is: `https://51467bc2b8ce.ngrok-free.app/auth/callback`

### Prerequisites

- Minikube installed and configured
- kubectl installed and configured
- Docker Desktop running
- PowerShell execution policy allows script execution

### Troubleshooting

#### If startup fails:

1. Check if Minikube is running: `minikube status`
2. Check if Docker is running: `docker version`
3. Look for error messages in the script output
4. Try running individual commands manually

#### If port forwarding fails:

1. Check if port 3000 is already in use
2. Kill existing kubectl processes: `Get-Process kubectl | Stop-Process`
3. Restart the port forwarding manually: `kubectl port-forward service/frontend 3000:80`

#### If OAuth doesn't work:

1. Update the ngrok tunnel URL in auth-service-configmap.yaml
2. Restart the auth-service: `kubectl rollout restart deployment/auth-service`
3. Check that the redirect URI matches in your Bungie.net app settings

### Manual Commands

If you prefer to run commands manually:

```powershell
# Start Minikube
minikube start

# Build images (from each service directory)
docker build -t guardian-tracker/auth-service:latest .
docker build -t guardian-tracker/bungie-service:latest .
docker build -t guardian-tracker/graphql-service:latest .
docker build -t guardian-tracker/frontend:v2 .

# Deploy services
kubectl apply -f auth-service-configmap.yaml
kubectl apply -f auth-service.yaml
kubectl apply -f bungie-service.yaml
kubectl apply -f graphql-service.yaml
kubectl apply -f frontend.yaml

# Port forward
kubectl port-forward service/frontend 3000:80

# Check status
kubectl get pods
kubectl get services
```

### File Structure

```
k8s/
├── startup.ps1           # Main startup script
├── startup.bat           # Batch wrapper for startup
├── shutdown.ps1          # Main shutdown script
├── shutdown.bat          # Batch wrapper for shutdown
├── README.md             # This file
├── auth-service-configmap.yaml
├── auth-service.yaml
├── bungie-service.yaml
├── graphql-service.yaml
└── frontend.yaml
```

## Support

If you encounter issues:

1. Check the script output for specific error messages
2. Verify all prerequisites are installed and running
3. Try running individual kubectl/docker commands manually
4. Check Minikube logs: `minikube logs`
