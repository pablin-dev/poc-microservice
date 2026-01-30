# Kafka SOAP Microservices E2E Test Example

This project demonstrates an end-to-end testing setup for a system involving two microservices interacting via Kafka and HTTP/SOAP, using Go with Ginkgo, Gomega, Testify, and GoMock (conceptually for unit tests). The entire system is orchestrated using Docker Compose.

## Project Structure

```
.
├── docker-compose.yaml
├── go.mod
├── go.sum
├── Makefile
├── README.md
├── services/
│   ├── consumer/
│   │   ├── consumer-service
│   │   ├── Dockerfile
│   │   ├── main_test.go
│   │   ├── main.go
│   │   ├── clients/
│   │   │   ├── consumer/
│   │   │   │   ├── consumer_test.go
│   │   │   │   └── consumer.go
│   │   │   ├── producer/
│   │   │   │   ├── producer_test.go
│   │   │   │   └── producer.go
│   │   │   └── soapclient/
│   │   │       ├── soapclient_test.go
│   │   │       └── soapclient.go
│   │   └── models/
│   │       └── models.go
│   └── providers/
│       └── kyc/
│           ├── Dockerfile
│           ├── helper.go
│           ├── kyc-service
│           ├── main_test.go
│           ├── main.go
│           ├── repository_test.go
│           └── repository.go
└── tests/
    ├── clients/
    │   ├── kafka.go
    │   └── kyc.go
    ├── e2e/
    │   ├── Dockerfile
    │   ├── main_test.go
    │   └── soap_matcher_test.go
    └── usecases/
        └── kyc/
            ├── user1.json
            ├── user2.json
            ├── user3.json
            ├── user4.json
            └── user5.json
```

## Services

1.  **KYC Provider Service (`services/providers/kyc`)**:
    *   A Go HTTP server that listens on port `8081`.
    *   It provides a SOAP endpoint (`/soap`) for CRUD operations on KYC (Know Your Customer) data.
    *   It expects an HTTP POST request with a SOAP/XML payload, supporting `KYCQuery`, `CreateKYC`, `UpdateKYC`, and `DeleteKYC` request types.
    *   Responds with a specific SOAP/XML message based on the incoming request type and operation result.
    *   Additionally, it exposes an Admin REST API (`/admin/v1/users` and `/admin/v1/actions`) for managing user data and simulating specific server behaviors (e.g., timeout, internal error, not found) for testing purposes.
    *   Initial user data can be loaded from JSON files located in `tests/usecases/kyc`.

2.  **Consumer Service (`services/consumer`)**:
    *   A Go application that consumes JSON messages from a Kafka topic named `Receive`.
    *   These messages contain a `type` field ("READ", "CREATE", "UPDATE", "DELETE") and a `ClientID` or `UserData` payload to determine the outgoing SOAP request to the KYC Provider Service.
    *   It utilizes internal packages (`clients/consumer`, `clients/producer`, `clients/soapclient`, `models`) to manage Kafka interactions and SOAP client calls.
    *   Upon receiving a message, it constructs and makes an HTTP POST call to the `kyc-provider-service` (e.g., `KYCQuery`, `CreateKYC`).
    *   Parses the SOAP/XML response from the `kyc-provider-service`.
    *   Transforms the relevant data from the SOAP response into a simpler JSON entity (`models.UserData`).
    *   Publishes this JSON entity (or an error representation if the SOAP operation failed) to another Kafka topic named `Response`.
    *   Includes robust startup checks to ensure Kafka and the KYC Provider Service are ready before processing messages.

## End-to-End Testing

The E2E tests are implemented using [Ginkgo](https://onsi.github.io/ginkgo/) and [Gomega](https://onsi.github.io/gomega/) and are located in `tests/e2e`.

*   **Custom Gomega Matcher (`tests/e2e/soap_matcher_test.go`)**: Includes a custom Gomega matcher `ContainSOAPElement` that allows for assertions against XML content using XPath expressions (though less relevant now with structured JSON responses from consumer).
*   **Test Clients (`tests/clients`)**: Contains Go packages for interacting with Kafka (`kafka.go` for producer, consumer, and admin operations) and the KYC Provider Service's Admin API (`kyc.go` for managing test data and simulating actions).
*   **Test Data (`tests/usecases/kyc`)**: JSON files (e.g., `user1.json`) define initial user data used by the KYC Provider Service and for setting up test scenarios via the Admin API.
*   **Table-Driven Tests (`tests/e2e/main_test.go`)**: The test suite produces JSON messages (specifying the SOAP request type and user data) to the `Receive` Kafka topic, waits for the `consumer-service` to process them, and then consumes the resulting JSON entity from the `Response` Kafka topic to perform assertions. This includes testing various CRUD flows (CREATE, READ, UPDATE, DELETE) and error simulation for KYC data. These tests leverage the `tests/clients` for setup and teardown of test data and simulated service behavior.

## GoMock

While GoMock was part of the initial request, it's important to note its role in this E2E context:

*   For the E2E tests themselves, direct mocking of Kafka or the HTTP client is generally avoided to ensure the entire system's integration is validated. The E2E tests interact with real Kafka and deployed microservices.
*   GoMock would be primarily beneficial for **unit testing** individual components (e.g., mocking the Kafka client within the `consumer-service`'s unit tests, or mocking an external HTTP client for the `consumer-service` if the SOAP service were external to the integration being tested).

## Setup and Running the Tests

### Prerequisites

*   [Docker](https://www.docker.com/get-started/)
*   [Docker Compose](https://docs.docker.com/compose/install/)

### Quick Start (Recommended)

1.  **Clone this repository** (or ensure you are in the project root directory).
2.  **Run the E2E tests**:
    ```bash
    make all
    ```
    This command will:
    *   Build all Docker images for the services and the test runner.
    *   Start Zookeeper, Kafka, `kyc-provider-service`, and `consumer-service`.
    *   Execute the E2E tests in a dedicated container.
    *   Automatically tear down all Docker Compose services upon completion.

### Individual Commands

You can also run the steps individually using the provided `Makefile` targets:

*   **Build Docker Images**:
    ```bash
    make build-images
    ```

*   **Start Services (Kafka, Zookeeper, KYC Provider, Consumer)**:
    ```bash
    make up
    ```
    (Run `docker compose logs -f` to view service logs)

*   **Run E2E Tests**:
    ```bash
    make test-e2e
    ```
    This command will also bring down the services after the tests run.

*   **Stop and Remove Services**:
    ```bash
    make down
    ```

*   **Clean Up (Remove images and artifacts)**:
    ```bash
    make clean
    ```

## Testing Mountebank Impostors

Once the Mountebank service is running (e.g., via `make up` if `mb-config.ejs` is loaded with all impostors), you can test the various impostor endpoints using `curl`.

### Key-Value Store SOAP Impostor (Port 4546, `imposter2.ejs`)

**1. Store an XML Key-Value Pair (SOAP):**

```bash
curl -X POST http://localhost:4546/ws -H 'Content-Type: application/soap+xml' -H 'SOAPAction: http://tempuri.org/StoreXML' -d '<?xml version="1.0" encoding="utf-8"?><soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/" xmlns:tem="http://tempuri.org/"><soapenv:Header/><soapenv:Body><tem:StoreXML><tem:Key>soapXMLKey</tem:Key><tem:Data><customer><id>CUST001</id><name>Jane Doe</name><email>jane.doe@example.com</email></customer></tem:Data></tem:StoreXML></soapenv:Body></soapenv:Envelope>'
```

**2. Retrieve a Key's XML Value (SOAP):**

```bash
curl -X POST http://localhost:4546/ws -H 'Content-Type: application/soap+xml' -H 'SOAPAction: http://tempuri.org/Retrieve' -d '<?xml version="1.0" encoding="utf-8"?><soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/" xmlns:tem="http://tempuri.org/"><soapenv:Header/><soapenv:Body><tem:Retrieve><tem:Key>soapXMLKey</tem:Key></tem:Retrieve></soapenv:Body></soapenv:Envelope>'
# Expected response body will contain the stored XML within <tem:Result>
```

### Key-Value Store Impostor (Port 4545, `impostor.ejs`)

**1. Store a Key-Value Pair (JSON):**

```bash
curl -X POST http://localhost:4545/store -H 'Content-Type: application/json' -d '{"key": "jsonKey", "value": "jsonValue"}'
```

**2. Retrieve a Key's Value (JSON):**

```bash
curl -X GET "http://localhost:4545/retrieve?key=jsonKey"
```

### Hello World SOAP Impostor (Port 4547, `hello.ejs`)

**1. Say Hello (SOAP):**

```bash
curl -X POST http://localhost:4547/soap -H 'Content-Type: application/soap+xml' -H 'SOAPAction: http://tempuri.org/HelloWorld' -d '<?xml version="1.0" encoding="utf-8"?><soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/" xmlns:tem="http://tempuri.org/"><soapenv:Header/><soapenv:Body><tem:HelloWorld><tem:Name>MountebankUser</tem:Name></tem:HelloWorld></soapenv:Body></soapenv:Envelope>'
```

### Mountebank Management Endpoints

These commands interact with the Mountebank server itself, typically on its main administration port (2525).

**1. Get All Impostors:**

```bash
curl http://localhost:2525/imposters
```

**2. Delete All Impostors:**

```bash
curl -X DELETE http://localhost:2525/imposters
```

**3. Get Mountebank Configuration:**

```bash
curl http://localhost:2525/config
```

**4. Get Mountebank Logs:**

```bash
curl http://localhost:2525/logs
```

## Docker Compose Details

The `docker-compose.yaml` defines the following services:

*   **`zookeeper`**: Confluentinc Zookeeper for Kafka.
*   **`kafka`**: Confluentinc Kafka broker. It's configured to auto-create `Receive` and `Response` topics.
*   **`kyc-provider-service`**: Builds and runs the Go KYC Provider microservice. Exposes port `8081`.
*   **`consumer-service`**: Builds and runs the Go Kafka consumer microservice. It's dependent on `kafka` and `soap-service`.
*   **`e2e-tests`**: Builds and runs the Ginkgo E2E test suite. It's dependent on `consumer-service` and connects to the Kafka broker within the Docker network.
