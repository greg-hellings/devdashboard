# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download
RUN go mod verify

# Copy source code
COPY . .

# Build the binary
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.version=${VERSION}" \
    -a -installsuffix cgo \
    -o devdashboard \
    ./cmd/devdashboard

# Runtime stage
FROM scratch

# Copy CA certificates for HTTPS
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy the binary
COPY --from=builder /build/devdashboard /usr/local/bin/devdashboard

# Create non-root user
USER 65534:65534

# Set entrypoint
ENTRYPOINT ["/usr/local/bin/devdashboard"]

# Default command
CMD ["help"]

# Metadata
LABEL org.opencontainers.image.title="DevDashboard"
LABEL org.opencontainers.image.description="Repository management and dependency analysis tool"
LABEL org.opencontainers.image.source="https://github.com/greg-hellings/devdashboard"
LABEL org.opencontainers.image.licenses="MIT"
