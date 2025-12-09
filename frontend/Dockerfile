# Use the official Node.js image for building
FROM node:18-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy package.json and package-lock.json
COPY package*.json ./

# Install dependencies
RUN npm ci

# Copy the source code into the container
COPY . .

# Build the React application
RUN npm run build

# Use nginx unprivileged image (runs as non-root by default)
FROM nginxinc/nginx-unprivileged:1.25-alpine

# Copy the built application from the previous stage
COPY --from=builder /app/build /usr/share/nginx/html

# Copy nginx configuration
COPY nginx.conf /etc/nginx/conf.d/default.conf

# Expose port 8080 (nginx-unprivileged uses 8080 by default)
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/ || exit 1

# Start nginx (runs as non-root user 101)
CMD ["nginx", "-g", "daemon off;"]
