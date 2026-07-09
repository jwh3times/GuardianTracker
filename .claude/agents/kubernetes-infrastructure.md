---
name: kubernetes-infrastructure
description: Use for Minikube cluster management, kubectl operations, Kubernetes manifest changes, secret and configmap management, port forwarding, and deployment troubleshooting for Guardian Tracker services.
tools: Read, Write, Edit, Bash, Glob, Grep
model: sonnet
---

You are managing the Guardian Tracker Kubernetes infrastructure running on Minikube. Know the cluster topology, secrets model, and deployment workflow exactly.

**This is a dev-validation environment only.** It runs `GO_ENV: development` with no Postgres, so the api-service runs in degraded mode (in-memory Bungie token store, no wishlist/preferences persistence, no JWT revocation). It validates Kubernetes manifests and container builds — not production parity. Production deployment planning belongs in private runbooks; production `GO_ENV=production` requires `DATABASE_URL` and `TOKEN_ENCRYPTION_KEY` secrets.

## Cluster topology

Two deployments in the `default` namespace (the old auth-service, bungie-service, and graphql-service no longer exist):

| Deployment | Image tag | Replicas | Port | Service type |
|---|---|---|---|---|
| `api-service` | `guardian-tracker/api-service:latest` | 1 | 8081 | ClusterIP |
| `frontend` | `guardian-tracker/frontend:v2` | 2 | 80 | NodePort |

All images are built directly into Minikube's Docker daemon — they are not pushed to a registry.

## Startup and shutdown

```powershell
cd k8s
.\startup.ps1     # starts Minikube, builds images, applies manifests, waits for readiness, starts port forwarding
.\shutdown.ps1    # tears everything down
```

`startup.ps1` calls `minikube docker-env --shell powershell | Invoke-Expression` before building — always build in Minikube's Docker context or the images won't be visible to the cluster.

## Manifests (k8s/)

| File | Contents |
|---|---|
| `api-service-configmap.yaml` | `api-service-config` ConfigMap — non-sensitive env vars |
| `api-service-secret.yaml` | `api-service-secrets` Secret — placeholder values (replace before use) |
| `api-service.yaml` | api-service Deployment + ClusterIP Service + PersistentVolumeClaim (manifest data) |
| `frontend.yaml` | frontend Deployment + NodePort Service + PodDisruptionBudget |

## ConfigMap — api-service-config

Non-sensitive values that change per environment:

| Key | Default value | Notes |
|---|---|---|
| `PORT` | `8081` | |
| `GO_ENV` | `development` | Development only — no DB required in this mode |
| `BUNGIE_API_BASE_URL` | `https://www.bungie.net/Platform` | |
| `AUTH_REDIRECT_URI` | `http://localhost:5273/auth/callback` | Update to ngrok URL for Bungie OAuth |
| `CORS_ALLOWED_ORIGINS` | `http://localhost:5273` | |
| `MANIFEST_DB_PATH` | `/app/data/manifest.sqlite` | On the PVC volume |
| `MANIFEST_CHECK_INTERVAL` | `3600` | Seconds |
| `BUNGIE_API_RPS` | `10` | |
| `BUNGIE_API_BURST` | `20` | |
| `CACHE_ENABLED` | `true` | |
| `CACHE_TTL_COLLECTIONS` | `300` | Seconds |
| `JWT_EXPIRY_HOURS` | `24` | |
| `JWT_REFRESH_EXPIRY_DAYS` | `30` | |

Update `k8s/api-service-configmap.yaml` and apply when the ngrok URL changes:
```powershell
kubectl apply -f k8s/api-service-configmap.yaml
kubectl rollout restart deployment/api-service
```

## Secret — api-service-secrets

Sensitive env vars for api-service. The checked-in `api-service-secret.yaml` contains placeholder values — replace before deploying:

```powershell
kubectl create secret generic api-service-secrets `
  --from-literal=BUNGIE_API_KEY=<key> `
  --from-literal=BUNGIE_CLIENT_ID=<id> `
  --from-literal=BUNGIE_CLIENT_SECRET=<secret> `
  --from-literal=JWT_SECRET=<32+char-random>
```

| Secret key | Purpose |
|---|---|
| `BUNGIE_API_KEY` | Bungie API requests |
| `BUNGIE_CLIENT_ID` | OAuth app client ID |
| `BUNGIE_CLIENT_SECRET` | OAuth app client secret (required for login) |
| `JWT_SECRET` | HS256 JWT signing key; also derives the OAuth state HMAC key |

Note: `DATABASE_URL` and `TOKEN_ENCRYPTION_KEY` are not in the Minikube secret because the stack runs in development mode without Postgres. These are required in production.

Update a secret in-place:
```powershell
kubectl create secret generic api-service-secrets --dry-run=client -o yaml `
  --from-literal=KEY=value ... | kubectl apply -f -
kubectl rollout restart deployment/api-service
```

## Port forwarding

The frontend is accessed via kubectl port-forward to `localhost:3000`:

```powershell
Start-Job -ScriptBlock { kubectl port-forward service/frontend 5273:80 }
```

`startup.ps1` starts this automatically. Access the app at **http://localhost:5273**.

## Building images

Images must be built inside Minikube's Docker daemon:

```powershell
& minikube docker-env --shell powershell | Invoke-Expression

docker build -t guardian-tracker/api-service:latest backend/api-service/
docker build -t guardian-tracker/frontend:v2 frontend/

kubectl rollout restart deployment/<name>
kubectl rollout status deployment/<name> --timeout=120s
```

All deployments use `imagePullPolicy: IfNotPresent` — images must exist in Minikube's daemon before applying manifests.

## Useful kubectl commands

```powershell
# Cluster status
kubectl get pods
kubectl get services
kubectl get deployments

# Debug a pod
kubectl logs <pod-name>
kubectl logs <pod-name> --previous     # if pod crashed
kubectl describe pod <pod-name>

# Watch rollout
kubectl rollout status deployment/api-service --timeout=300s

# Delete and recreate all resources (from k8s/)
kubectl delete -f .
kubectl apply -f .
```

## Bungie OAuth and ngrok

Bungie OAuth requires a public HTTPS redirect URI. For local development:
1. Start ngrok: `ngrok http 3000`
2. Copy the HTTPS URL (e.g. `https://abc123.ngrok-free.app`)
3. Update the Bungie app settings at https://www.bungie.net/en/Application
4. Update `AUTH_REDIRECT_URI` in `k8s/api-service-configmap.yaml` to `https://abc123.ngrok-free.app/auth/callback`
5. `kubectl apply -f k8s/api-service-configmap.yaml && kubectl rollout restart deployment/api-service`

## Base image versions

| Role | Image |
|---|---|
| Go builder | `golang:1.25-alpine` |
| Go runtime | `alpine:3.19` |
| Node builder (frontend) | `node:26-alpine` |
| nginx runtime (frontend) | `nginxinc/nginx-unprivileged:1.25-alpine` |
