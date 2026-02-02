package main

import (
	"context" // Added for context.WithTimeout
	"encoding/json"
	"fmt" // Added for fmt.Errorf
	"log"
	"net/http" // Added for health check
	"os"
	"time"

	consumerPkg "kafka-soap-e2e-test/services/consumer/clients/consumer"
	producerPkg "kafka-soap-e2e-test/services/consumer/clients/producer"
	soapclientPkg "kafka-soap-e2e-test/services/consumer/clients/soapclient"
	"kafka-soap-e2e-test/services/consumer/models"

	"github.com/segmentio/kafka-go"


)

func main() {
	log.Println("Consumer Service: Starting up...")

	kafkaBootstrapServers := "kafka:9092"
	if s := os.Getenv("KAFKA_BOOTSTRAP_SERVERS"); s != "" {
		kafkaBootstrapServers = s
	}
	soapServiceURL := "http://soap-service:8081/soap"
	if s := os.Getenv("SOAP_SERVICE_URL"); s != "" {
		soapServiceURL = s
	}

	log.Printf("Consumer Service: KAFKA_BOOTSTRAP_SERVERS: %s", kafkaBootstrapServers)
	log.Printf("Consumer Service: SOAP_SERVICE_URL: %s", soapServiceURL)

	// Wait for Kafka connection to be established
	err := waitForKafkaConnection(kafkaBootstrapServers, 60*time.Second) // Give Kafka up to 60 seconds
	if err != nil {
		log.Println("Consumer Service: Shutting down due to Kafka not being ready.")
		log.Fatalf("Failed to wait for Kafka connection: %v", err)
	}

	// Kafka Consumer setup
	consumer, err := consumerPkg.NewConsumer(kafkaBootstrapServers, "consumer-group")
	if err != nil {
		log.Println("Consumer Service: Shutting down due to Kafka consumer creation error.")
		log.Fatalf("Failed to create Kafka consumer: %v", err)
	}
	defer consumer.Close()



	// Kafka Producer setup
	producer, err := producerPkg.NewProducer(kafkaBootstrapServers)
	if err != nil {
		log.Println("Consumer Service: Shutting down due to Kafka producer creation error.")
		log.Fatalf("Failed to create Kafka producer: %v", err)
	}
	defer producer.Close()

	// SOAP Client setup
	soapClient := soapclientPkg.NewSOAPClient(soapServiceURL)

	// Wait for SOAP service to be ready
	err = waitForSoapService(soapServiceURL, 60*time.Second) // Give SOAP service up to 60 seconds
	if err != nil {
		log.Println("Consumer Service: Shutting down due to SOAP service not being ready.")
		log.Fatalf("Failed to wait for SOAP service: %v", err)
	}

	log.Println("Consumer Service: Entering Kafka message processing loop.")
	for {
		msg, err := consumer.ReadMessage(context.Background()) // Use background context for ReadMessage
		if err == nil {
			log.Printf("Received message from Kafka topic %s: %s\n", msg.Topic, string(msg.Value))
			log.Printf("Consumer Service: Unmarshaling Kafka message. Content: %s", string(msg.Value))
			var kafkaMsg models.KafkaMessage
			if err := json.Unmarshal(msg.Value, &kafkaMsg); err != nil {
				log.Printf("Failed to unmarshal Kafka message: %v", err)
				continue
			}

			var processedEntity models.UserData
			var err error
			var deleteMessage string

			switch kafkaMsg.Type {
			case "READ":
				log.Printf("Consumer Service: Performing KYC Read for ClientID: %s", kafkaMsg.ClientID)
				processedEntity, err = soapClient.ReadKYC(kafkaMsg.ClientID)
			case "CREATE":
				log.Printf("Consumer Service: Performing KYC Create for ClientID: %s", kafkaMsg.UserData.ClientID)
				// Ensure ClientID is correctly set from KafkaMessage if not already in UserData
				if kafkaMsg.UserData.ClientID == "" {
					kafkaMsg.UserData.ClientID = kafkaMsg.ClientID
				}
				processedEntity, err = soapClient.CreateKYC(kafkaMsg.UserData)
			case "UPDATE":
				log.Printf("Consumer Service: Performing KYC Update for ClientID: %s", kafkaMsg.UserData.ClientID)
				// Ensure ClientID is correctly set from KafkaMessage if not already in UserData
				if kafkaMsg.UserData.ClientID == "" {
					kafkaMsg.UserData.ClientID = kafkaMsg.ClientID
				}
				processedEntity, err = soapClient.UpdateKYC(kafkaMsg.UserData)
			case "DELETE":
				log.Printf("Consumer Service: Performing KYC Delete for ClientID: %s", kafkaMsg.ClientID)
				deleteMessage, err = soapClient.DeleteKYC(kafkaMsg.ClientID)
				if err == nil {
					// For delete, we construct a dummy UserData for the response topic
					// to indicate success, as there's no UserData returned by DeleteKYC
					processedEntity = models.UserData{
						ClientID: kafkaMsg.ClientID,
						Status:   "Success",
						Message:  deleteMessage,
					}
				}
			default:
				log.Printf("Consumer Service: Unknown Kafka message type: %s", kafkaMsg.Type)
				err = fmt.Errorf("unknown Kafka message type: %s", kafkaMsg.Type)
			}

			if err != nil {
				log.Printf("Failed to perform SOAP operation for type %s: %v", kafkaMsg.Type, err)
				// Construct an error response to send back to Kafka
				errorEntity := models.UserData{
					ClientID: kafkaMsg.ClientID,
					Status:   "Error",
					Message:  fmt.Sprintf("Failed to perform %s operation: %v", kafkaMsg.Type, err),
				}
				entityBytes, marshalErr := json.Marshal(errorEntity)
				if marshalErr != nil {
					log.Printf("Failed to marshal error entity to JSON: %v", marshalErr)
					continue
				}
				// Produce error message to Kafka
				errorMsg := kafka.Message{
					Topic:   producerPkg.KafkaResponseTopic,
					Value:   entityBytes,
					Headers: []kafka.Header{{Key: "correlationId", Value: []byte(kafkaMsg.CorrelationID)}},
				}
				err = producer.Produce(errorMsg)
				if err != nil {
					log.Printf("Failed to produce error message to Kafka: %v", err)
				}
				continue
			}

			// For successful operations, processedEntity holds the result
			log.Printf("Consumer Service: Successfully performed %s operation. UserData ClientID: %s, Status: %s, Message: %s, Risk: %f",
				kafkaMsg.Type, processedEntity.ClientID, processedEntity.Status, processedEntity.Message, processedEntity.Risk)

			// Publish to Response topic
			entityBytes, err := json.Marshal(processedEntity)
			if err != nil {
				log.Printf("Failed to marshal processed entity to JSON: %v", err)
				continue
			}

			log.Printf("Consumer Service: Producing message to Kafka topic: %s. Entity: %s", producerPkg.KafkaResponseTopic, string(entityBytes))
			successMsg := kafka.Message{
				Topic:   producerPkg.KafkaResponseTopic,
				Value:   entityBytes,
				Headers: []kafka.Header{{Key: "correlationId", Value: []byte(kafkaMsg.CorrelationID)}},
			}
			err = producer.Produce(successMsg)
			if err != nil {
				log.Printf("Failed to produce message to Kafka: %v", err)
			} else {
				log.Printf("Consumer Service: Successfully produced message to Kafka topic: %s", producerPkg.KafkaResponseTopic)
			}
		} else if err != context.DeadlineExceeded && err != context.Canceled { // context.DeadlineExceeded for timeout from ReadMessage, context.Canceled if the context is cancelled
			log.Printf("Consumer error: %v\n", err)
		}
	}
}

// waitForKafkaConnection polls Kafka until it's ready to serve requests or a timeout occurs.
func waitForKafkaConnection(bootstrapServers string, timeout time.Duration) error {
	log.Printf("Consumer Service: Waiting for Kafka at %s to be ready for %s", bootstrapServers, timeout)
	endTime := time.Now().Add(timeout)
	for time.Now().Before(endTime) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel() // Ensure context is cancelled even if check succeeds

		conn, err := kafka.DialContext(ctx, "tcp", bootstrapServers)
		if err != nil {
			log.Printf("Consumer Service: Kafka not yet ready (Dial failed): %v. Retrying...", err)
			time.Sleep(1 * time.Second)
			continue
		}
		defer conn.Close() // Close the connection after each check

		// Optionally, check controller to ensure it's healthy
		_, err = conn.Controller()
		if err != nil {
			log.Printf("Consumer Service: Kafka not yet ready (Controller failed): %v. Retrying...", err)
			time.Sleep(1 * time.Second)
			continue
		}

		// Read partitions to verify cluster metadata can be fetched
		_, err = conn.ReadPartitions()
		if err == nil {
			log.Println("Consumer Service: Kafka connection established and cluster metadata fetched.")
			return nil
		}

		log.Printf("Consumer Service: Kafka not yet ready (ReadPartitions failed): %v. Retrying...", err)
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("timed out waiting for Kafka to be ready")
}

// waitForSoapService polls the SOAP service URL until it's reachable or a timeout occurs.
func waitForSoapService(url string, timeout time.Duration) error {
	log.Printf("Consumer Service: Waiting for SOAP service at %s to be ready for %s", url, timeout)
	endTime := time.Now().Add(timeout)
	for time.Now().Before(endTime) {
		req, errReq := http.NewRequest(http.MethodHead, url, nil)
		if errReq != nil {
			log.Printf("Consumer Service: Failed to create HTTP request for SOAP service: %v. Retrying...", errReq)
			time.Sleep(1 * time.Second)
			continue
		}

		resp, err := http.DefaultClient.Do(req)
		if err == nil { // Request was successful (no network error)
			defer resp.Body.Close() // Close body immediately after checking status

			if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusMethodNotAllowed {
				log.Printf("Consumer Service: SOAP service is ready (status: %d).", resp.StatusCode)
				return nil
			}
			// Log non-200/405 status code
			log.Printf("Consumer Service: SOAP service not yet ready (status: %d). Retrying...", resp.StatusCode)
		} else { // Request failed (network error, timeout, etc.)
			log.Printf("Consumer Service: Failed to reach SOAP service: %v. Retrying...", err)
		}
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("timed out waiting for SOAP service to be ready")
}

// Helper to get a string pointer for Kafka topic
// This helper is now part of the consumer/producer packages
// func KafkaStringPtr(s string) *string { return &s }
