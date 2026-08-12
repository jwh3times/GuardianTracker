# Build stage
FROM node:26.7.0-alpine3.24@sha256:aadf416b2cdce311a8811ba3f0608a61b77dbf997500e2eafe781b51f6a0b019 AS builder

WORKDIR /app

COPY package*.json ./
RUN npm ci

COPY . .
RUN npm run build

# Production stage — nginx serves the static dist/
FROM nginxinc/nginx-unprivileged:1.31.3-alpine3.24@sha256:334d92979f15aaecd5dd50af5105e1230e2bb70765d45b1e2f964e7c5eda81c3

COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://127.0.0.1:8080/ || exit 1

CMD ["nginx", "-g", "daemon off;"]
