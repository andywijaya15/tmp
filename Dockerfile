# ===============================
# Stage 1 — Build the Go binary
# ===============================
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install git for go get
RUN apk add --no-cache git

# Copy go.mod dan go.sum
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN go build -o tmp-files main.go

# ===============================
# Stage 2 — Minimal runtime
# ===============================
FROM alpine:latest

WORKDIR /app

# Install CA certificates (for HTTPS calls if needed)
RUN apk add --no-cache ca-certificates

# Copy binary dan static files
COPY --from=builder /app/tmp-files .
COPY --from=builder /app/static ./static

# Create tmp folder for uploads
RUN mkdir -p /app/tmp

# Expose port
EXPOSE 3003

# Run the binary
CMD ["./tmp-files"]
