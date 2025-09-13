# Use the official Golang image as the builder
FROM golang:1.24.3 AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy the source code
COPY . .

RUN go mod tidy

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/test-rps main.go
# Set execute permissions in the builder stage
RUN chmod 755 /app/bin/test-rps

# Use a distroless image for the final stage
FROM gcr.io/distroless/base-debian11

# Set the working directory inside the container
WORKDIR /srv

# Copy the binary from the builder stage (already executable)
COPY --from=builder /app/bin/test-rps /srv/test-rps

EXPOSE 8080

# Command to run the binary
CMD ["/srv/test-rps"]
