# Use the official Go image to build the application
FROM golang:1.25-bookworm AS builder

WORKDIR /src

COPY go.mod go.sum ./

# Install necessary packages for building, including CGO dependencies for librdkafka
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    librdkafka-dev \
    git \
    && rm -rf /var/lib/apt/lists/* \
    && go clean --modcache \
    && go mod download

COPY . .

# Build the Go application
# On Debian, confluent-kafka-go should pick up librdkafka-dev correctly without explicit ldflags
RUN go build -o /consumer-service ./services/consumer/main.go

# Use a minimal image to run the application
FROM debian:bookworm-slim

# Set the working directory
WORKDIR /root/

# Copy the pre-built binary from the builder stage
COPY --from=builder /consumer-service .

# Command to run the executable
CMD ["./consumer-service"]
