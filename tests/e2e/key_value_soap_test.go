package e2e_test

import (
	"fmt"
	"io"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// NOTE: This file is part of the Mountebank E2E test suite.
// The BeforeAll setup and helper functions are defined in mountebank_suite_test.go.

var _ = Context("Key-Value Store SOAP Impostor functionality on port 4546", func() {
	var targetURL string
	const imposterPort = 4546

	BeforeEach(func() {
		targetURL = fmt.Sprintf("http://127.0.0.1:%d/ws", imposterPort)
	})

	It("should successfully store an XML key-value pair via SOAP", func() {
		By("Sending SOAP POST request to store XML data")
		soapAction := "http://tempuri.org/StoreXML" // Changed SOAPAction
		key := "soapTestKey1"
		xmlValue := `<data><item id="1">value1</item><item id="2">value2</item></data>` // XML content
		soapRequest := generateStoreXMLSOAPRequest(key, xmlValue)                       // Use new helper

		resp, err := sendSOAPRequest(testFramework.MountebankClient.HTTPClient, targetURL, soapAction, soapRequest) // Use testFramework.MountebankClient.HTTPClient
		Expect(err).NotTo(HaveOccurred(), "Failed to send SOAP StoreXML request")
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		bodyBytes, err := io.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred(), "Failed to read SOAP StoreXML response body")
		Expect(string(bodyBytes)).To(ContainSOAPElementWithValue("//tem:StoreXMLResponse/tem:Result", "XML Data stored successfully")) // Updated expected response
	})

	It("should successfully retrieve a stored XML value via SOAP", func() {
		By("Storing an XML key-value pair first for retrieval test")
		soapActionStore := "http://tempuri.org/StoreXML" // Changed SOAPAction
		key := "soapTestKey2"
		xmlValue := `<data><name>TestUser</name><email>test@example.com</email></data>` // XML content
		soapRequestStore := generateStoreXMLSOAPRequest(key, xmlValue)                  // Use new helper

		respStore, err := sendSOAPRequest(testFramework.MountebankClient.HTTPClient, targetURL, soapActionStore, soapRequestStore)
		Expect(err).NotTo(HaveOccurred(), "Failed to send initial SOAP StoreXML request for retrieval")
		Expect(respStore.StatusCode).To(Equal(http.StatusOK))

		By("Sending SOAP POST request to retrieve data")
		soapActionRetrieve := "http://tempuri.org/Retrieve"
		soapRequestRetrieve := generateRetrieveSOAPRequest(key)

		respRetrieve, err := sendSOAPRequest(testFramework.MountebankClient.HTTPClient, targetURL, soapActionRetrieve, soapRequestRetrieve)
		Expect(err).NotTo(HaveOccurred(), "Failed to send SOAP Retrieve request")
		Expect(respRetrieve.StatusCode).To(Equal(http.StatusOK))

		bodyBytes, err := io.ReadAll(respRetrieve.Body)
		Expect(err).NotTo(HaveOccurred(), "Failed to read SOAP Retrieve response body")
		// The assertion now expects the full XML value
		Expect(string(bodyBytes)).To(ContainSOAPElementWithValue("//tem:RetrieveResponse/tem:Result", xmlValue))
	})

	It("should return 404 for a non-existent key during SOAP retrieval", func() {
		By("Sending SOAP POST request to retrieve a non-existent key")
		soapAction := "http://tempuri.org/Retrieve"
		key := "nonExistentSoapKey"
		soapRequest := generateRetrieveSOAPRequest(key)

		resp, err := sendSOAPRequest(testFramework.MountebankClient.HTTPClient, targetURL, soapAction, soapRequest)
		Expect(err).NotTo(HaveOccurred(), "Failed to send SOAP Retrieve request for non-existent key")
		Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

		bodyBytes, err := io.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred(), "Failed to read error response body")
		Expect(string(bodyBytes)).To(ContainSOAPElementWithValue("//tem:ErrorResponse/tem:Message", fmt.Sprintf("Key not found: %s", key)))
	})

	It("should return 400 for a SOAP StoreXML request with missing Key or XML Data", func() {
		By("Sending SOAP StoreXML request with missing XML Data")
		soapAction := "http://tempuri.org/StoreXML" // Changed SOAPAction
		key := "incompleteSoapKey"
		soapRequestMissingData := generateMissingValueStoreSOAPRequest(key) // This now generates a request with missing Data

		resp, err := sendSOAPRequest(testFramework.MountebankClient.HTTPClient, targetURL, soapAction, soapRequestMissingData)
		Expect(err).NotTo(HaveOccurred(), "Failed to send incomplete SOAP StoreXML request (missing Data)")
		Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))

		bodyBytes, err := io.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred(), "Failed to read error response body")
		Expect(string(bodyBytes)).To(ContainSOAPElementWithValue("//tem:ErrorResponse/tem:Message", "Missing Key or XML Data in SOAP request body")) // Updated error message

		By("Sending SOAP StoreXML request with missing Key")
		xmlData := "<data>someValue</data>"                                  // Provide some XML data
		soapRequestMissingKey := generateMissingKeyStoreSOAPRequest(xmlData) // This now generates a request with missing Key

		resp, err = sendSOAPRequest(testFramework.MountebankClient.HTTPClient, targetURL, soapAction, soapRequestMissingKey)
		Expect(err).NotTo(HaveOccurred(), "Failed to send incomplete SOAP StoreXML request (missing Key)")
		Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))

		bodyBytes, err = io.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred(), "Failed to read error response body")
		Expect(string(bodyBytes)).To(ContainSOAPElementWithValue("//tem:ErrorResponse/tem:Message", "Missing Key or XML Data in SOAP request body")) // Updated error message
	})

	It("should return 400 for a SOAP Retrieve request with missing Key", func() {
		By("Sending SOAP POST request to retrieve with missing Key")
		soapAction := "http://tempuri.org/Retrieve"
		soapRequestMissingKey := generateMissingKeyRetrieveSOAPRequest()

		resp, err := sendSOAPRequest(testFramework.MountebankClient.HTTPClient, targetURL, soapAction, soapRequestMissingKey)
		Expect(err).NotTo(HaveOccurred(), "Failed to send incomplete SOAP Retrieve request (missing Key)")
		Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))

		bodyBytes, err := io.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred(), "Failed to read error response body")
		Expect(string(bodyBytes)).To(ContainSOAPElementWithValue("//tem:ErrorResponse/tem:Message", "Missing Key in SOAP request body"))
	})

	It("should return 404 for a request that doesn't match any SOAP predicate", func() {
		By("Sending a SOAP request with an unknown SOAPAction")
		soapAction := "http://tempuri.org/UnknownAction"
		soapRequest := generateUnknownActionSOAPRequest()

		resp, err := sendSOAPRequest(testFramework.MountebankClient.HTTPClient, targetURL, soapAction, soapRequest)
		Expect(err).NotTo(HaveOccurred(), "Failed to send SOAP request with unknown action")
		Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

		bodyBytes, err := io.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred(), "Failed to read error response body")
		Expect(string(bodyBytes)).To(ContainSOAPElementWithValue("//tem:ErrorResponse/tem:Message", "No matching SOAP service found for the request."))
	})
})
