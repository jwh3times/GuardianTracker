# Production Dockerfile for GraphQL Server
FROM node:26-alpine AS builder

WORKDIR /app

# Copy package files
COPY package*.json ./

# Install all dependencies (including dev for build)
RUN npm ci

# Copy source code
COPY . .

# Build the application
RUN npm run build

# Production stage
FROM node:26-alpine

# Security: Install dumb-init for proper signal handling (node user already exists at uid/gid 1000)
RUN apk --no-cache add dumb-init

# Set working directory
WORKDIR /app

# Copy built application from builder stage
COPY --from=builder --chown=node:node /app/dist ./dist
COPY --from=builder --chown=node:node /app/node_modules ./node_modules
COPY --from=builder --chown=node:node /app/package*.json ./

# Set environment to production
ENV NODE_ENV=production

# Switch to non-root user
USER node

# Expose port
EXPOSE 4000

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:4000/health || exit 1

# Use dumb-init to handle signals properly
CMD ["dumb-init", "node", "dist/server.js"]

