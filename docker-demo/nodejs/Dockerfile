# ==========================================
# STAGE 1: Build Environment
# ==========================================
FROM node:20-alpine AS builder

WORKDIR /usr/src/app

# Copy package.json and package-lock.json first to leverage Docker cache for node_modules
COPY package*.json ./

# Use npm ci (clean install) instead of npm install:
# - Extremely fast, highly reliable, and fully deterministic
# - Installs dependencies exactly as specified in package-lock.json
# - Ideal for CI/CD pipelines and Docker builds
RUN npm ci

# Copy the remaining source files
COPY . .

# ==========================================
# STAGE 2: Production Runtime
# ==========================================
FROM node:20-alpine

# Install 'tini' to act as the init process (PID 1) to handle kernel signals (SIGTERM, SIGINT).
# By default, Node.js does not handle OS signals properly at PID 1 (e.g., zombie process reaping, signal propagation).
RUN apk add --no-cache tini

WORKDIR /usr/src/app

# Set Production Environment variables
ENV NODE_ENV=production \
    PORT=8081

# Copy package.json to trigger npm scripts if needed
COPY --from=builder /usr/src/app/package*.json ./
# Copy node_modules and source files from the builder stage
COPY --from=builder /usr/src/app/node_modules ./node_modules
COPY --from=builder /usr/src/app/server.js ./server.js

# Use the predefined 'node' user from node-alpine image instead of root (Security best practice)
# Ensure the application directory belongs to the node user
RUN chown -R node:node /usr/src/app
USER node

# Expose port
EXPOSE 8081

# Use tini as the entrypoint and start the application via node
ENTRYPOINT ["/sbin/tini", "--"]
CMD ["node", "server.js"]
