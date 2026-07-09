# ADR 0004: Local Development and Minikube Scope

**Status:** Accepted
**Date:** 2026-07-08

## Context

Guardian Tracker has Docker Compose and Kubernetes manifests. The Kubernetes
path is useful for validating containers and manifests, but it does not currently
provide production parity.

## Decision

Docker Compose is the recommended local full-stack development path. Minikube is
a local validation environment for Kubernetes manifests and startup scripts. It
runs in development mode and should not be documented as production-equivalent.

Production deployment planning remains deferred until a target platform is
accepted and implemented.

## Consequences

- `SETUP.md` treats Compose as the default setup path.
- `k8s/README.md` remains focused on Minikube validation.
- Production-specific deployment runbooks belong in `private/` until public
  deployment automation exists.
