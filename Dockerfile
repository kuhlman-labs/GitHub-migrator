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

# Build stage - Backend
FROM golang:1.25-alpine AS backend-builder

WORKDIR /app

# Install dependencies (including curl for downloading binaries)
RUN apk add --no-cache git gcc musl-dev sqlite-dev curl bash

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

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates sqlite-libs git git-lfs

WORKDIR /app

# Copy server binary from backend builder (git-sizer is now embedded)
COPY --from=backend-builder /app/server .

# Copy configs
COPY configs ./configs

# Use docker.yaml as the default config.yaml (required by the application)
# Environment variables will override values from this file
RUN cp configs/docker.yaml configs/config.yaml || true

# Copy frontend static files from frontend builder
COPY --from=frontend-builder /app/web/dist ./web/dist

# Create data and logs directories
RUN mkdir -p /app/data /app/logs

# Expose port
EXPOSE 8080

# Run server
CMD ["./server"]
