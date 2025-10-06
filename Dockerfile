# Build stage - Backend
FROM golang:1.23-alpine AS backend-builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git gcc musl-dev sqlite-dev

# Copy go mod files
COPY go.mod go.sum ./
RUN GOTOOLCHAIN=auto go mod download

# Install git-sizer for repository analysis
RUN go install github.com/github/git-sizer@latest

# Copy source code
COPY . .

# Build binaries
RUN GOTOOLCHAIN=auto CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o server cmd/server/main.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates sqlite-libs git git-lfs

WORKDIR /app

# Copy binaries from backend builder
COPY --from=backend-builder /app/server .
COPY --from=backend-builder /go/bin/git-sizer /usr/local/bin/

# Copy configs
COPY configs ./configs

# Create data and logs directories
RUN mkdir -p /app/data /app/logs

# Expose port
EXPOSE 8080

# Run server
CMD ["./server"]
