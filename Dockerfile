# Build stage - Frontend
FROM node:20-alpine AS frontend-builder

WORKDIR /app/web

# Copy package files
COPY web/package*.json ./

# Install dependencies
RUN npm ci

# Copy frontend source
COPY web/ ./

# Build frontend
RUN npm run build

# Build stage - Backend (using Debian-based Go image for glibc compatibility)
FROM golang:1.25-bookworm AS backend-builder

WORKDIR /app

# Install dependencies (including curl and unzip for downloading binaries)
RUN apt-get update && apt-get install -y --no-install-recommends \
    git \
    gcc \
    libsqlite3-dev \
    curl \
    unzip \
    && rm -rf /var/lib/apt/lists/*

# Copy go mod files
COPY go.mod go.sum ./
RUN GOTOOLCHAIN=auto go mod download

# Copy source code and scripts
COPY . .

# Download git-sizer binaries for embedding
RUN chmod +x scripts/download-git-sizer.sh && \
    ./scripts/download-git-sizer.sh

# Build binaries with embedded git-sizer
RUN GOTOOLCHAIN=auto CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o server cmd/server/main.go

# Final stage - Debian slim for glibc compatibility with Copilot CLI
FROM debian:bookworm-slim

# Install runtime dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    git \
    git-lfs \
    curl \
    unzip \
    nodejs \
    npm \
    && rm -rf /var/lib/apt/lists/*

# Install git-sizer for repository analysis (releases are .zip format)
RUN ARCH=$(dpkg --print-architecture) && \
    if [ "$ARCH" = "amd64" ]; then \
        curl -fsSL -o /tmp/git-sizer.zip https://github.com/github/git-sizer/releases/download/v1.5.0/git-sizer-1.5.0-linux-amd64.zip && \
        unzip -j /tmp/git-sizer.zip git-sizer -d /usr/local/bin && \
        rm /tmp/git-sizer.zip && \
        chmod +x /usr/local/bin/git-sizer; \
    elif [ "$ARCH" = "arm64" ]; then \
        apt-get update && apt-get install -y --no-install-recommends golang-go && \
        go install github.com/github/git-sizer@v1.5.0 && \
        mv /root/go/bin/git-sizer /usr/local/bin/ && \
        apt-get purge -y golang-go && apt-get autoremove -y && \
        rm -rf /var/lib/apt/lists/* /root/go; \
    fi

# Install GitHub Copilot CLI globally
# The CLI is required for the Copilot SDK to function
RUN npm install -g @github/copilot && \
    copilot --version

WORKDIR /app

# Copy server binary from backend builder (git-sizer is now embedded)
COPY --from=backend-builder /app/server .

# Copy configs
COPY configs ./configs

# Use config_template.yml as the default config.yaml (required by the application)
# Environment variables will override values from this file
RUN cp configs/config_template.yml configs/config.yaml || true

# Copy frontend static files from frontend builder
COPY --from=frontend-builder /app/web/dist ./web/dist

# Create data and logs directories
RUN mkdir -p /app/data /app/logs

# Expose port
EXPOSE 8080

# Run server
CMD ["./server"]
