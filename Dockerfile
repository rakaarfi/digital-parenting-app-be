# Dockerfile

# --- Build Stage ---
FROM golang:1.21-alpine AS builder

# Set necessary environment variables for build
ENV CGO_ENABLED=0 GOOS=linux
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./
# Download dependencies
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the application
# -ldflags="-w -s" reduces the size of the binary by removing debug information.
# Adjust the main package path if necessary (./cmd/api/main.go)
RUN go build -ldflags="-w -s" -o /app/server ./cmd/api/main.go

# --- Final Stage ---
# Use a minimal alpine image for the final stage
FROM alpine:latest

WORKDIR /app

# Copy only the compiled binary from the builder stage
COPY --from=builder /app/server /app/server

# Copy migrations directory (optional, if needed inside the container for startup checks or runs)
# If migrations are run externally before deployment, this COPY can be removed.
COPY migrations ./migrations

# Expose the port the app runs on (make sure it matches APP_PORT in .env)
# This should be the port *inside* the container.
EXPOSE 3001

# Command to run the executable
ENTRYPOINT ["/app/server"]

# Optional: Add a non-root user for security
# RUN addgroup -S appgroup && adduser -S appuser -G appgroup
# USER appuser