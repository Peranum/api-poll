# Use the official Golang image as the builder
FROM golang:1.24.3 AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy go mod and sum files first for better caching
COPY go.mod go.sum ./

# Download dependencies with cache mount - this persists across builds
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy the source code
COPY . .

# Build the Go app with cache mounts for both module and build cache
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -o /app/bin/upbit-api-poll main.go

# Set execute permissions in the builder stage
RUN chmod 755 /app/bin/upbit-api-poll

# Use a distroless image for the final stage
FROM gcr.io/distroless/base-debian11

# Set the working directory inside the container
WORKDIR /srv

# Copy the binary from the builder stage (already executable)
COPY --from=builder /app/bin/upbit-api-poll /srv/upbit-api-poll

# Copy the .env and configs folder
COPY .env /srv/.env
COPY configs /srv/configs

EXPOSE 8080

# Command to run the binary
CMD ["/srv/upbit-api-poll", "--config", "/srv/configs/config.yaml"]
