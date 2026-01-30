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

	It("should successfully store a key-value pair via SOAP", func() {
		By("Sending SOAP POST request to store data")
		soapAction := "http://tempuri.org/Store"
		key := "soapTestKey1"
		value := "soapTestValue1"
		soapRequest := generateStoreSOAPRequest(key, value)

		resp, err := sendSOAPRequest(targetURL, soapAction, soapRequest)
		Expect(err).NotTo(HaveOccurred(), "Failed to send SOAP Store request")
		Expect(resp.StatusCode).To(Equal(http.StatusOK))

		bodyBytes, err := io.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred(), "Failed to read SOAP Store response body")
		Expect(string(bodyBytes)).To(ContainSOAPElementWithValue("//tem:StoreResponse/tem:Result", "Stored successfully"))
	})

	It("should successfully retrieve a stored value via SOAP", func() {
		By("Storing a key-value pair first for retrieval test")
		soapActionStore := "http://tempuri.org/Store"
		key := "soapTestKey2"
		value := "soapTestValue2"
		soapRequestStore := generateStoreSOAPRequest(key, value)

		respStore, err := sendSOAPRequest(targetURL, soapActionStore, soapRequestStore)
		Expect(err).NotTo(HaveOccurred(), "Failed to send initial SOAP Store request for retrieval")
		Expect(respStore.StatusCode).To(Equal(http.StatusOK))

		By("Sending SOAP POST request to retrieve data")
		soapActionRetrieve := "http://tempuri.org/Retrieve"
		soapRequestRetrieve := generateRetrieveSOAPRequest(key)

		respRetrieve, err := sendSOAPRequest(targetURL, soapActionRetrieve, soapRequestRetrieve)
		Expect(err).NotTo(HaveOccurred(), "Failed to send SOAP Retrieve request")
		Expect(respRetrieve.StatusCode).To(Equal(http.StatusOK))

		bodyBytes, err := io.ReadAll(respRetrieve.Body)
		Expect(err).NotTo(HaveOccurred(), "Failed to read SOAP Retrieve response body")
		Expect(string(bodyBytes)).To(ContainSOAPElementWithValue("//tem:RetrieveResponse/tem:Result", value))
	})

	It("should return 404 for a non-existent key during SOAP retrieval", func() {
		By("Sending SOAP POST request to retrieve a non-existent key")
		soapAction := "http://tempuri.org/Retrieve"
		key := "nonExistentSoapKey"
		soapRequest := generateRetrieveSOAPRequest(key)

		resp, err := sendSOAPRequest(targetURL, soapAction, soapRequest)
		Expect(err).NotTo(HaveOccurred(), "Failed to send SOAP Retrieve request for non-existent key")
		Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

		bodyBytes, err := io.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred(), "Failed to read error response body")
		Expect(string(bodyBytes)).To(ContainSOAPElementWithValue("//tem:ErrorResponse/tem:Message", fmt.Sprintf("Key not found: %s", key)))
	})

	It("should return 400 for a SOAP Store request with missing Key or Value", func() {
		By("Sending SOAP Store request with missing Value")
		soapAction := "http://tempuri.org/Store"
		key := "incompleteSoapKey"
		soapRequestMissingValue := generateMissingValueStoreSOAPRequest(key)

		resp, err := sendSOAPRequest(targetURL, soapAction, soapRequestMissingValue)
		Expect(err).NotTo(HaveOccurred(), "Failed to send incomplete SOAP Store request (missing Value)")
		Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))

		bodyBytes, err := io.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred(), "Failed to read error response body")
		Expect(string(bodyBytes)).To(ContainSOAPElementWithValue("//tem:ErrorResponse/tem:Message", "Missing Key or Value in SOAP request body"))

		By("Sending SOAP Store request with missing Key")
		value := "incompleteSoapValue"
		soapRequestMissingKey := generateMissingKeyStoreSOAPRequest(value)

		resp, err = sendSOAPRequest(targetURL, soapAction, soapRequestMissingKey)
		Expect(err).NotTo(HaveOccurred(), "Failed to send incomplete SOAP Store request (missing Key)")
		Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))

		bodyBytes, err = io.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred(), "Failed to read error response body")
		Expect(string(bodyBytes)).To(ContainSOAPElementWithValue("//tem:ErrorResponse/tem:Message", "Missing Key or Value in SOAP request body"))
	})

	It("should return 400 for a SOAP Retrieve request with missing Key", func() {
		By("Sending SOAP POST request to retrieve with missing Key")
		soapAction := "http://tempuri.org/Retrieve"
		soapRequestMissingKey := generateMissingKeyRetrieveSOAPRequest()

		resp, err := sendSOAPRequest(targetURL, soapAction, soapRequestMissingKey)
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

		resp, err := sendSOAPRequest(targetURL, soapAction, soapRequest)
		Expect(err).NotTo(HaveOccurred(), "Failed to send SOAP request with unknown action")
		Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

		bodyBytes, err := io.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred(), "Failed to read error response body")
		Expect(string(bodyBytes)).To(ContainSOAPElementWithValue("//tem:ErrorResponse/tem:Message", "No matching SOAP service found for the request."))
	})
})