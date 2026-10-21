# Build stage
FROM node:26.12.0-alpine3.24@sha256:6f08a523443ddedf52941874c203475b0ad503e859ad70cb26ab8be9a517dbd5 AS builder

WORKDIR /app

COPY package*.json ./
RUN npm ci

COPY . .
# Development-only tunnel OAuth callback handoff (issue #317). Env files are
# excluded from the build context, so it arrives as a build argument; empty by
# default, which leaves the callback unchanged.
ARG VITE_OAUTH_COMPLETION_ORIGIN=
ENV VITE_OAUTH_COMPLETION_ORIGIN=${VITE_OAUTH_COMPLETION_ORIGIN}
RUN npm run build

# Production stage — nginx serves the static dist/
FROM nginxinc/nginx-unprivileged:1.31.6-alpine3.24@sha256:884e0b7f42f303641b0f4e78755aea647521dca0b89ca3be7992ad78b5c07434

COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://127.0.0.1:8080/ || exit 1

CMD ["nginx", "-g", "daemon off;"]
