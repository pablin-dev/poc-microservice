package e2e_test

import (
	"encoding/json"
	"fmt"
	"kafka-soap-e2e-test/services/consumer/models"
	"log"
	"math/rand"
	"strings"
	"testing"
	"time"

	adminClient "kafka-soap-e2e-test/tests/clients" // Import the clients package with an alias
	framework "kafka-soap-e2e-test/tests/framework" // Import the framework package

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2E Suite")
}

const (
	kafkaReceiveTopic = "Receive"

	kafkaResponseTopic = "Response"
)

var rng *rand.Rand // Declare rng here

var _ = Describe("Kafka SOAP E2E Test Suite", Ordered, func() {
	BeforeAll(func() {
		defer GinkgoRecover()
		log.SetOutput(GinkgoWriter) // Redirect standard logger to GinkgoWriter

		var err error
		testFramework, err = framework.NewFramework("../config.yaml") // Corrected path to config.yaml
		Expect(err).NotTo(HaveOccurred(), "Failed to initialize test framework")

		// Wait for Kafka to be ready using the new client
		err = testFramework.KafkaClient.WaitForKafka(60 * time.Second) // Give Kafka up to 60 seconds
		Expect(err).NotTo(HaveOccurred(), "Kafka did not become ready")

		// A small additional sleep after Kafka is ready, just in case services need to catch up
		time.Sleep(5 * time.Second)

		// Seed the random number generator for unique correlation IDs
		rng = rand.New(rand.NewSource(time.Now().UnixNano())) // Initialize rng here

		// Initialize Admin API Client is now handled by framework.NewFramework
		Expect(testFramework.KycClient).NotTo(BeNil(), "Admin API client should not be nil after initialization")
		fmt.Println("DEBUG: Starting Kafka topic creation process.")
		// Ensure topics are clean before recreation
		log.Printf("Attempting to delete Kafka topic: %s", kafkaReceiveTopic)
		err = testFramework.KafkaClient.DeleteKafkaTopic(kafkaReceiveTopic)
		if err != nil {
			log.Printf("Warning: Error deleting Kafka topic %s: %v. It might not have existed, proceeding.", kafkaReceiveTopic, err)
		} else {
			log.Printf("Successfully deleted Kafka topic %s.", kafkaReceiveTopic)
		}

		log.Printf("Attempting to delete Kafka topic: %s", kafkaResponseTopic)
		err = testFramework.KafkaClient.DeleteKafkaTopic(kafkaResponseTopic)
		if err != nil {
			log.Printf("Warning: Error deleting Kafka topic %s: %v. It might not have existed, proceeding.", kafkaResponseTopic, err)
		} else {
			log.Printf("Successfully deleted Kafka topic %s.", kafkaResponseTopic)
		}

		// Give Kafka a moment to fully propagate topic deletion before creation
		time.Sleep(2 * time.Second)

		// Create Kafka topics if they don't exist
		log.Printf("Attempting to create Kafka topic: %s", kafkaReceiveTopic)
		err = testFramework.KafkaClient.CreateKafkaTopic(kafkaReceiveTopic, 1) // Use new client method
		if err != nil {
			log.Printf("Error creating Kafka topic %s: %v", kafkaReceiveTopic, err)
		}
		Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to create Kafka topic %s", kafkaReceiveTopic))
		log.Printf("Successfully handled Kafka topic %s creation/existence.", kafkaReceiveTopic)

		log.Printf("Attempting to create Kafka topic: %s", kafkaResponseTopic)
		err = testFramework.KafkaClient.CreateKafkaTopic(kafkaResponseTopic, 1) // Use new client method
		if err != nil {
			log.Printf("Error creating Kafka topic %s: %v", kafkaResponseTopic, err)
		}
		Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to create Kafka topic %s", kafkaResponseTopic))
		log.Printf("Successfully handled Kafka topic %s creation/existence.", kafkaResponseTopic)

		// Give Kafka a moment to fully propagate topic creation
		time.Sleep(1 * time.Second)

		fmt.Println("DEBUG: About to initialize Kafka Producer")

		// Subscribe the Kafka consumer to the response topic
		err = testFramework.KafkaClient.Consumer.SubscribeTopics([]string{kafkaResponseTopic}, nil) // Use testFramework.KafkaClient.Consumer
		Expect(err).NotTo(HaveOccurred(), "Failed to subscribe to Kafka response topic")
	})

	Context("SOAP message processing via Kafka", func() {
		type TestCase struct {
			TestName         string // Added for better logging in setup/teardown
			KafkaMessageType string
			KafkaClientID    string
			KafkaUserData    models.UserData // For CREATE/UPDATE operations

			ExpectedClientID string
			ExpectedStatus   string
			ExpectedMessage  string
			ExpectedRisk     float64
		}

		DescribeTable("should process Kafka message, call SOAP service, and publish response",
			func(tc TestCase) {
				var dummyKafkaMessage []byte
				var err error

				// --- Admin Setup: Ensure user exists for READ/UPDATE/DELETE, or doesn't exist for CREATE ---
				By(fmt.Sprintf("Admin Setup for '%s' test (ClientID: %s)", tc.TestName, tc.KafkaClientID))
				// Ensure the client exists for READ, UPDATE, DELETE scenarios, or if it's the error test case
				if tc.KafkaMessageType == "READ" || tc.KafkaMessageType == "UPDATE" || tc.KafkaMessageType == "DELETE" || tc.TestName == "READ clientError InternalError" || tc.TestName == "READ clientTimeout Timeout" {
					_, readErr := testFramework.KycClient.ReadUser(tc.KafkaClientID)
					if readErr != nil && strings.Contains(readErr.Error(), "status: 404") {
						defaultUserData := models.UserData{
							ClientID: tc.KafkaClientID,
							Risk:     0.1,
							Status:   "Initial",
							Message:  "Created by test setup",
						}
						Expect(testFramework.KycClient.CreateUser(defaultUserData)).NotTo(HaveOccurred(), "Admin API: Failed to create user for test setup")
						log.Printf("Admin API: Created client '%s' for test setup.", tc.KafkaClientID)
					} else {
						Expect(readErr).NotTo(HaveOccurred(), "Admin API: Failed to read user for test setup")
						log.Printf("Admin API: Client '%s' exists for test setup.", tc.KafkaClientID)
					}
				} else if tc.KafkaMessageType == "CREATE" {
					// Ensure the client does not exist before creating
					err = testFramework.KycClient.DeleteUser(tc.KafkaClientID)
					if err != nil && !strings.Contains(err.Error(), "status: 404") {
						Expect(err).NotTo(HaveOccurred(), "Admin API: Failed to delete user before create test")
					}
					log.Printf("Admin API: Ensured client '%s' does not exist.", tc.KafkaClientID)
				}

				// Apply specific action for error test case
				if tc.TestName == "READ clientError InternalError" {
					Expect(testFramework.KycClient.SetAction(tc.KafkaClientID, "InternalError")).NotTo(HaveOccurred(), "Admin API: Failed to set InternalError action")
					log.Printf("Admin API: Set InternalError action for client '%s'.", tc.KafkaClientID)
				} else if tc.TestName == "READ clientTimeout Timeout" {
					Expect(testFramework.KycClient.SetAction(tc.KafkaClientID, "Timeout")).NotTo(HaveOccurred(), "Admin API: Failed to set Timeout action")
					log.Printf("Admin API: Set Timeout action for client '%s'.", tc.KafkaClientID)
				}

				// Clear any *general* previous action overrides for this client before this specific test's setup,
				// unless this test case itself is setting an action.
				if tc.TestName != "READ clientError InternalError" && tc.TestName != "READ clientTimeout Timeout" {
					Expect(testFramework.KycClient.ClearAction(tc.KafkaClientID)).NotTo(HaveOccurred(), "Admin API: Failed to clear action before test")
					log.Printf("Admin API: Cleared actions for client '%s'.", tc.KafkaClientID)
				}

				// --- Send Kafka Message ---
				// Ensure CorrelationID is unique for each message send, including retries within the loop
				kafkaMsgToSend := models.KafkaMessage{
					Type:          tc.KafkaMessageType,
					CorrelationID: fmt.Sprintf("dummy-req-%s-%d", tc.KafkaClientID, rng.Intn(100000)), // More unique CorrelationID
					ClientID:      tc.KafkaClientID,
				}

				// Populate UserData only if provided in TestCase (for CREATE/UPDATE)
				if tc.KafkaUserData.ClientID != "" || tc.KafkaUserData.Risk != 0 || tc.KafkaUserData.Status != "" || tc.KafkaUserData.Message != "" {
					kafkaMsgToSend.UserData = tc.KafkaUserData
				}

				dummyKafkaMessage, err = json.Marshal(kafkaMsgToSend)
				Expect(err).NotTo(HaveOccurred(), "Failed to marshal dummy Kafka message")

				By(fmt.Sprintf("Sending Kafka message type '%s' to '%s' topic (ClientID: %s, CorrelationID: %s)", tc.KafkaMessageType, kafkaReceiveTopic, tc.KafkaClientID, kafkaMsgToSend.CorrelationID))
				// Produce message to 'Receive' topic
				err = testFramework.KafkaClient.Producer.Produce(&kafka.Message{ // Use testFramework.KafkaClient.Producer
					TopicPartition: kafka.TopicPartition{Topic: adminClient.KafkaStringPtr(kafkaReceiveTopic), Partition: kafka.PartitionAny}, // Use testKafkaClient.KafkaStringPtr
					Value:          dummyKafkaMessage,
					Headers:        []kafka.Header{{Key: "correlationId", Value: []byte(kafkaMsgToSend.CorrelationID)}},
				}, nil)
				Expect(err).NotTo(HaveOccurred(), "Failed to produce message to Receive topic")

				// Wait for message delivery
				ev := testFramework.KafkaClient.Producer.Flush(10 * 1000) // Use testFramework.KafkaClient.Producer
				Expect(ev).To(BeNumerically(">", 0), "Producer did not flush any messages")

				By(fmt.Sprintf("Waiting for response on '%s' topic for triggered SOAP call (Expected CorrelationID: %s)", kafkaResponseTopic, kafkaMsgToSend.CorrelationID))

				// Consume message from 'Response' topic
				var receivedEntity models.UserData
				startTime := time.Now()
				for time.Since(startTime) < 60*time.Second {
					ev := testFramework.KafkaClient.Consumer.Poll(100) // Use testFramework.KafkaClient.Consumer
					if ev == nil {
						continue // No event received, try again after a small delay
					}

					switch e := ev.(type) {
					case kafka.Error:
						Fail(fmt.Sprintf("Kafka consumer error: %v", e))
					case *kafka.Message:
						// Extract correlationId from header
						var receivedCorrelationID string
						for _, header := range e.Headers {
							if header.Key == "correlationId" {
								receivedCorrelationID = string(header.Value)
								break
							}
						}

						// If correlationId header is missing, log and treat as not matching
						if receivedCorrelationID == "" {
							log.Printf("Received message with missing correlationId header. Value: %s. Continuing to poll...", string(e.Value))
							continue
						}

						// Verify the correlation ID matches the one sent for this test case
						if receivedCorrelationID != kafkaMsgToSend.CorrelationID {
							log.Printf("Received message with unexpected CorrelationID: %s (Expected: %s). Value: %s. Continuing to poll...", receivedCorrelationID, kafkaMsgToSend.CorrelationID, string(e.Value))
							continue // Not the message we are looking for, continue polling
						}

						log.Printf("Received response message (CorrelationID: %s): %s", receivedCorrelationID, string(e.Value))
						err := json.Unmarshal(e.Value, &receivedEntity)
						Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal received entity")

						// Verify the transformed message
						Expect(receivedEntity.ClientID).To(Equal(tc.ExpectedClientID))
						Expect(receivedEntity.Status).To(Equal(tc.ExpectedStatus))
						Expect(receivedEntity.Message).To(Equal(tc.ExpectedMessage))
						Expect(receivedEntity.Risk).To(Equal(tc.ExpectedRisk))

						// --- Test Teardown (Admin API) ---
						By(fmt.Sprintf("Admin Teardown for '%s' test (ClientID: %s)", tc.TestName, tc.KafkaClientID))
						// Clean up created users or revert updated users
						switch tc.KafkaMessageType {
						case "CREATE":
							Expect(testFramework.KycClient.DeleteUser(tc.KafkaClientID)).NotTo(HaveOccurred(), "Admin API: Failed to delete user after create test")
							log.Printf("Admin API: Deleted client '%s' after create test.", tc.KafkaClientID)
						case "UPDATE":
							// Revert clientA123 to its original state (from user1.json)
							originalClientA123 := models.UserData{
								ClientID: "clientA123",
								Risk:     0.15,
								Status:   "Approved",
								Message:  "Initial KYC check passed",
							}
							Expect(testFramework.KycClient.UpdateUser(originalClientA123)).NotTo(HaveOccurred(), "Admin API: Failed to revert clientA123 after update test")
							log.Printf("Admin API: Reverted client '%s' after update test.", tc.KafkaClientID)
						case "DELETE":
							// No special teardown needed as it's already deleted
							log.Printf("Admin API: Client '%s' already deleted.", tc.KafkaClientID)
						}
						// Always clear the action set for error injection, if applicable
						if tc.TestName == "READ clientError InternalError" {
							Expect(testFramework.KycClient.ClearAction(tc.KafkaClientID)).NotTo(HaveOccurred(), "Admin API: Failed to clear InternalError action during teardown")
							log.Printf("Admin API: Cleared InternalError action for client '%s' during teardown.", tc.KafkaClientID)
						} else if tc.TestName == "READ clientTimeout Timeout" {
							Expect(testFramework.KycClient.ClearAction(tc.KafkaClientID)).NotTo(HaveOccurred(), "Admin API: Failed to clear Timeout action during teardown")
							log.Printf("Admin API: Cleared Timeout action for client '%s' during teardown.", tc.KafkaClientID)
						}
						return // Message processed, exit the test case
					}
				}
				Fail("Timed out waiting for Kafka response message") // If loop finishes without receiving a message
			},

			Entry("KYCQuery for clientA123 (READ)", TestCase{
				TestName:         "READ clientA123",
				KafkaMessageType: "READ",
				KafkaClientID:    "clientA123",
				ExpectedClientID: "clientA123",
				ExpectedStatus:   "Success",
				ExpectedMessage:  "User data retrieved",
				ExpectedRisk:     0.15,
			}),

			Entry("KYCCreate for newClientXYZ (CREATE)", TestCase{
				TestName:         "CREATE newClientXYZ",
				KafkaMessageType: "CREATE",
				KafkaClientID:    "newClientXYZ",
				KafkaUserData: models.UserData{
					ClientID: "newClientXYZ",
					Risk:     0.5,
					Status:   "Pending",
					Message:  "Initial creation",
				},
				ExpectedClientID: "newClientXYZ",
				ExpectedStatus:   "Success",
				ExpectedMessage:  "User created",
				ExpectedRisk:     0.5,
			}),

			Entry("KYCUpdate for clientA123 (UPDATE)", TestCase{
				TestName:         "UPDATE clientA123",
				KafkaMessageType: "UPDATE",
				KafkaClientID:    "clientA123",
				KafkaUserData: models.UserData{
					ClientID: "clientA123",
					Risk:     0.75, // Updated risk
					Status:   "Approved",
					Message:  "Risk updated",
				},
				ExpectedClientID: "clientA123",
				ExpectedStatus:   "Success",
				ExpectedMessage:  "User updated",
				ExpectedRisk:     0.75,
			}),

			Entry("KYCDelete for newClientXYZ (DELETE)", TestCase{
				TestName:         "DELETE newClientXYZ",
				KafkaMessageType: "DELETE",
				KafkaClientID:    "newClientXYZ",
				ExpectedClientID: "newClientXYZ",
				ExpectedStatus:   "Success",
				ExpectedMessage:  "User deleted",
				ExpectedRisk:     0.0, // Default for delete response
			}),

			Entry("KYCQuery for clientError (READ) with simulated InternalError", TestCase{
				TestName:         "READ clientError InternalError",
				KafkaMessageType: "READ",
				KafkaClientID:    "clientError",
				ExpectedClientID: "clientError",
				ExpectedStatus:   "Error",
				ExpectedMessage:  "Failed to perform READ operation: SOAP service returned non-OK status: 500",
				ExpectedRisk:     0.0, // Default for error response
			}),

			Entry("KYCQuery for clientTimeout (READ) with simulated Timeout", TestCase{
				TestName:         "READ clientTimeout Timeout",
				KafkaMessageType: "READ",
				KafkaClientID:    "clientTimeout",
				ExpectedClientID: "clientTimeout",
				ExpectedStatus:   "Error",
				ExpectedMessage:  "Failed to perform READ operation: SOAP service returned non-OK status: 408", // Expected 408 from kyc-service
				ExpectedRisk:     0.0,                                                                          // Default for error response
			}),
		)
	})
})
