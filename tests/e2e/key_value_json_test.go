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

var _ = Context("Key-Value Store Impostor functionality", func() {
	const imposterPort = 4545
	const storePath = "/store"
	const retrievePath = "/retrieve"

	It("should successfully store a key-value pair", func() {
		By("Sending POST request to store data")
		storeRequestBody := `{"key": "testKey", "value": "testValue"}`
		targetURL := fmt.Sprintf("http://127.0.0.1:%d%s", imposterPort, storePath)

		resp, err := sendJSONPostRequest(targetURL, storeRequestBody)
		Expect(err).NotTo(HaveOccurred(), "Failed to send store POST request")
		Expect(resp.StatusCode).To(Equal(http.StatusOK))

		bodyBytes, err := io.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred(), "Failed to read store response body")
		Expect(string(bodyBytes)).To(MatchJSON(`{"message":"Stored successfully","key":"testKey","value":"testValue"}`))
	})

	It("should successfully retrieve a stored value", func() {
		By("Storing a key-value pair first")
		storeRequestBody := `{"key": "anotherKey", "value": "anotherValue"}`
		storeURL := fmt.Sprintf("http://127.0.0.1:%d%s", imposterPort, storePath)
		storeResp, err := sendJSONPostRequest(storeURL, storeRequestBody)
		Expect(err).NotTo(HaveOccurred(), "Failed to send initial store POST request")
		Expect(storeResp.StatusCode).To(Equal(http.StatusOK))

		By("Sending GET request to retrieve data")
		retrieveURL := fmt.Sprintf("http://127.0.0.1:%d%s?key=anotherKey", imposterPort, retrievePath)
		retrieveResp, err := sendJSONGetRequest(retrieveURL)
		Expect(err).NotTo(HaveOccurred(), "Failed to send retrieve GET request")
		Expect(retrieveResp.StatusCode).To(Equal(http.StatusOK))

		bodyBytes, err := io.ReadAll(retrieveResp.Body)
		Expect(err).NotTo(HaveOccurred(), "Failed to read retrieve response body")
		Expect(string(bodyBytes)).To(MatchJSON(`{"key":"anotherKey","value":"anotherValue"}`))
	})

	It("should return 404 for a non-existent key during retrieval", func() {
		By("Sending GET request for a non-existent key")
		retrieveURL := fmt.Sprintf("http://127.0.0.1:%d%s?key=nonExistentKey", imposterPort, retrievePath)
		retrieveResp, err := sendJSONGetRequest(retrieveURL)
		Expect(err).NotTo(HaveOccurred(), "Failed to send retrieve GET request for non-existent key")
		Expect(retrieveResp.StatusCode).To(Equal(http.StatusNotFound))

		bodyBytes, err := io.ReadAll(retrieveResp.Body)
		Expect(err).NotTo(HaveOccurred(), "Failed to read error response body")
		Expect(string(bodyBytes)).To(ContainSubstring(`"error":"Key not found: nonExistentKey"`))
	})

	It("should return 400 for a store request with missing key or value", func() {
		By("Sending POST request with missing value")
		storeRequestBody := `{"key": "incompleteKey"}`
		targetURL := fmt.Sprintf("http://127.0.0.1:%d%s", imposterPort, storePath)

		resp, err := sendJSONPostRequest(targetURL, storeRequestBody)
		Expect(err).NotTo(HaveOccurred(), "Failed to send incomplete store POST request")
		Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))

		bodyBytes, err := io.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred(), "Failed to read error response body")
		Expect(string(bodyBytes)).To(ContainSubstring(`"error":"Missing key or value in request body"`))

		By("Sending POST request with missing key")
		storeRequestBody = `{"value": "incompleteValue"}`
		resp, err = sendJSONPostRequest(targetURL, storeRequestBody)
		Expect(err).NotTo(HaveOccurred(), "Failed to send incomplete store POST request")
		Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))

		bodyBytes, err = io.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred(), "Failed to read error response body")
		Expect(string(bodyBytes)).To(ContainSubstring(`"error":"Missing key or value in request body"`))
	})

	It("should return 400 for a retrieve request with missing key parameter", func() {
		By("Sending GET request with missing key parameter")
		retrieveURL := fmt.Sprintf("http://127.0.0.1:%d%s", imposterPort, retrievePath) // No query parameter
		retrieveResp, err := sendJSONGetRequest(retrieveURL)
		Expect(err).NotTo(HaveOccurred(), "Failed to send retrieve GET request with missing key")
		Expect(retrieveResp.StatusCode).To(Equal(http.StatusBadRequest))

		bodyBytes, err := io.ReadAll(retrieveResp.Body)
		Expect(err).NotTo(HaveOccurred(), "Failed to read error response body")
		Expect(string(bodyBytes)).To(ContainSubstring(`"error":"Missing key query parameter"`))
	})
})
