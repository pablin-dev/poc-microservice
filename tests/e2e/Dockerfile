FROM golang:1.25-bookworm AS builder

WORKDIR /src

COPY go.mod go.sum ./

# Install necessary packages for building, including CGO dependencies for librdkafka
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    librdkafka-dev \
    git \
    && rm -rf /var/lib/apt/lists/* \
    && go mod tidy

COPY . .

# Build the test executable
# Ginkgo typically runs tests directly, but for Docker Compose, it's often better
# to have the ginkgo binary available and then run tests.
RUN go install github.com/onsi/ginkgo/v2/ginkgo@latest
# This will copy the ginkgo binary to /go/bin/ginkgo

# Final image for running tests
FROM golang:1.25-bookworm

# Create the /app directory
RUN mkdir -p /app
# Set the working directory
WORKDIR /app

# Copy only the necessary binaries and files
COPY --from=builder /go/bin/ginkgo /usr/local/bin/ginkgo
# Copy the entire /src directory (which contains go.mod, services, tests/e2e etc.) from the builder stage
COPY --from=builder /src /app

# Set the entrypoint to ginkgo, command will be the test arguments
ENTRYPOINT ["ginkgo"]
