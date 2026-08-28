# Minikube Manifest Validation

This directory validates Guardian Tracker's Kubernetes manifests and container
builds in local Minikube. It is not a production deployment template: the API
runs with `GO_ENV: development`, without Postgres persistence, database-backed
JWT revocation, or durable wishlist/preferences storage.

## Prerequisites

- Minikube, kubectl, Docker Desktop, and Windows PowerShell or PowerShell 7
- A Bungie API key and public OAuth client ID
- An ignored `api-service-secret.yaml`, prepared as described below

## Prepare the Local Secret Manifest

`startup.ps1` applies `api-service-secret.yaml`, but that file is deliberately
gitignored. The root setup helper creates it without overwriting an existing
file:

```powershell
cd ..
./setup.ps1
```

Alternatively, copy `api-service-secret.yaml.example` to
`api-service-secret.yaml`. Replace all three placeholders before startup.
Authorized maintainers may restore the `k8s` target with
`scripts/restore-private-secrets.ps1` before running `setup.ps1`. The helpers
preserve an existing file.

`BUNGIE_CLIENT_ID` is a public identifier; Guardian Tracker does not use a Bungie
client secret. Keep development credentials in this local manifest and never
use production values in the Minikube validation stack.

## Start and Stop

From this directory:

```powershell
./startup.ps1
./shutdown.ps1
```

The `.bat` wrappers run the same scripts for users who prefer Explorer or
Windows Command Prompt. Startup:

1. starts Minikube when needed;
2. selects Minikube's Docker daemon and builds both application images with
   `--pull --no-cache`;
3. applies the config map, ignored local secret, deployments, services, PVC,
   and frontend disruption budget;
4. restarts only an existing deployment whose applied pod template did not
   already trigger a rollout; and
5. forwards <http://localhost:5273> to the frontend service.

Shutdown removes the Kubernetes resources and port forwarding, then optionally
removes the local images or stops Minikube.

## OAuth Through ngrok

Tunnel the forwarded frontend port:

```powershell
ngrok http 5273
```

For every new tunnel URL, update all three matching locations:

1. the Bungie application callback URL;
2. `AUTH_REDIRECT_URI` in `api-service-configmap.yaml`, including
   `/auth/callback`; and
3. `CORS_ALLOWED_ORIGINS` in `api-service-configmap.yaml`, using the origin only.

Apply the changed config and restart the API:

```powershell
kubectl apply -f api-service-configmap.yaml
kubectl rollout restart deployment/api-service
kubectl rollout status deployment/api-service --timeout=300s
```

## Manual Validation

The scripts are the canonical workflow. These commands are useful when isolating
a failed step:

```powershell
minikube start
& minikube docker-env --shell powershell | Invoke-Expression

# Run these builds from the repository root.
docker build --pull --no-cache -t guardian-tracker/api-service:latest backend/api-service/
docker build --pull --no-cache -t guardian-tracker/frontend:v2 frontend/

kubectl apply -f k8s/api-service-configmap.yaml
kubectl apply -f k8s/api-service-secret.yaml
kubectl apply -f k8s/api-service.yaml
kubectl apply -f k8s/frontend.yaml

kubectl rollout status deployment/api-service --timeout=300s
kubectl rollout status deployment/frontend --timeout=300s
kubectl port-forward service/frontend 5273:80
```

Both deployments use `imagePullPolicy: Never` because their images exist only
inside Minikube's Docker daemon. After rebuilding a reused tag, restart an
existing deployment only when applying the manifest did not already change its
pod template.

## Troubleshooting

- Missing-secret errors: confirm `api-service-secret.yaml` exists in this
  directory and contains all three keys from the committed example.
- Image pull or `ImagePullBackOff`: rerun `minikube docker-env` in the current
  shell, rebuild, and confirm the deployment uses `imagePullPolicy: Never`.
- Port-forward failure: check whether 5273 is already in use, then run
  `kubectl port-forward service/frontend 5273:80` manually.
- OAuth failure: confirm the tunnel origin and callback match both the config map
  and Bungie application exactly, then restart the API deployment.
- Pod failure: use `kubectl get pods`, `kubectl describe pod <name>`, and
  `kubectl logs <name> --previous`.

The manifests and scripts are authoritative for topology and image tags. See
[SETUP.md](../SETUP.md#5-validate-kubernetes-manifests) for the higher-level
development options and port map.
