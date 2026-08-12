# Guardian Tracker - Kubernetes Deployment Scripts

This directory contains scripts to start and stop the Guardian Tracker application on Minikube.

> **Dev-validation environment only.** This stack runs with `GO_ENV: development` and has
> **no Postgres**, so the api-service runs in degraded mode: in-memory Bungie token store,
> no wishlist/preferences persistence, no JWT revocation. It exists to validate the
> Kubernetes manifests and container builds — it is **not** a production deployment
> template. No production runtime has been selected; production deployment planning
> remains private and deferred.

## Quick Start

### Option 1: Using Batch Files (Easiest)

- **Start:** Double-click `startup.bat`
- **Stop:** Double-click `shutdown.bat`

### Option 2: Using PowerShell Scripts Directly

- **Start:** `powershell.exe -ExecutionPolicy Bypass -File startup.ps1`
- **Stop:** `powershell.exe -ExecutionPolicy Bypass -File shutdown.ps1`

## What the Startup Script Does

1. **Starts Minikube** (if not already running)
2. **Builds Docker Images** for both services with fresh pinned base images
   (`--pull --no-cache`):
   - `guardian-tracker/api-service:latest`
   - `guardian-tracker/frontend:v2`
3. **Deploys Kubernetes manifests**
4. **Activates rebuilt images** by restarting a Deployment whose pod template
   stayed unchanged; newly created Deployments and pod-template changes already
   have a rollout in progress and are not restarted again. Then it waits for
   readiness.
5. **Sets up port forwarding** (localhost:5273 → frontend)
6. **Shows status** and provides next steps

After startup completes:

- Frontend available at: <http://localhost:5273>
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

- Remember to update your ngrok tunnel URL in `api-service-configmap.yaml` if needed
- The current OAuth redirect URI must match your Bungie.net application settings

### Prerequisites

- Minikube installed and configured
- kubectl installed and configured
- Docker Desktop running
- PowerShell execution policy allows script execution

### Troubleshooting

#### If startup fails

1. Check if Minikube is running: `minikube status`
2. Check if Docker is running: `docker version`
3. Look for error messages in the script output
4. Try running individual commands manually

#### If port forwarding fails

1. Check if port 5273 is already in use
2. Kill existing kubectl processes: `Get-Process kubectl | Stop-Process`
3. Restart the port forwarding manually: `kubectl port-forward service/frontend 5273:80`

#### If OAuth doesn't work

1. Update the ngrok tunnel URL in `api-service-configmap.yaml`
2. Restart the api-service: `kubectl rollout restart deployment/api-service`
3. Check that the redirect URI matches in your Bungie.net app settings

### Manual Commands

```powershell
# Start Minikube
minikube start

# Point Docker at Minikube, then build fresh images (from the repository root)
& minikube docker-env --shell powershell | Invoke-Expression
docker build --pull --no-cache -t guardian-tracker/api-service:latest backend/api-service/
docker build --pull --no-cache -t guardian-tracker/frontend:v2 frontend/

# Deploy services
kubectl apply -f k8s/api-service-configmap.yaml
kubectl apply -f k8s/api-service-secret.yaml
kubectl apply -f k8s/api-service.yaml
kubectl apply -f k8s/frontend.yaml

# If applying these manifests did not change the pod template, consume the
# rebuilt local tags. Skip restarts for new Deployments or pod-template changes.
kubectl rollout restart deployment/api-service deployment/frontend
kubectl rollout status deployment/api-service --timeout=300s
kubectl rollout status deployment/frontend --timeout=300s

# Port forward
kubectl port-forward service/frontend 5273:80

# Check status
kubectl get pods
kubectl get services
```

### File Structure

```text
k8s/
├── startup.ps1               # Main startup script
├── startup.bat               # Batch wrapper for startup
├── shutdown.ps1              # Main shutdown script
├── shutdown.bat              # Batch wrapper for shutdown
├── README.md                 # This file
├── api-service-configmap.yaml
├── api-service-secret.yaml
└── api-service.yaml
└── frontend.yaml
```

## Support

If you encounter issues:

1. Check the script output for specific error messages
2. Verify all prerequisites are installed and running
3. Try running individual kubectl/docker commands manually
4. Check Minikube logs: `minikube logs`
