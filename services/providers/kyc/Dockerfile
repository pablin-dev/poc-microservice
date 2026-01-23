FROM golang:1.25-bookworm AS builder

WORKDIR /src

COPY go.mod go.sum ./

# Install necessary packages for building, including CGO dependencies for librdkafka (even if not directly used here)
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    librdkafka-dev \
    && rm -rf /var/lib/apt/lists/* \
    && go clean --modcache \
    && go mod download

COPY . .

# Build the Go application for the KYC service
RUN go build -o /kyc-service ./services/providers/kyc

# Use a minimal image to run the application
FROM debian:bookworm-slim

# Set the working directory
WORKDIR /root/ 

# Copy the pre-built binary from the builder stage
COPY --from=builder /kyc-service .

# Expose port 8081 for the KYC service (assuming it uses the same port as the original SOAP service)
EXPOSE 8081

# Command to run the executable
CMD ["./kyc-service"]
