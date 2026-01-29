package e2e_test

import (
	"bytes"
	"fmt"
	"io"
	"kafka-soap-e2e-test/tests/clients"
	"log"
	"net/http"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Mountebank E2E Tests", Ordered, func() {
	var mountebankClient *clients.MountebankClient
	var mountebankBaseURL string

	BeforeAll(func() {
		defer GinkgoRecover()
		log.SetOutput(GinkgoWriter)

		mountebankBaseURL = os.Getenv("MOUNTEBANK_BASE_URL")
		if mountebankBaseURL == "" {
			mountebankBaseURL = "http://127.0.0.1:2525" // Default for local execution
		}

		mountebankClient = clients.NewMountebankClient(mountebankBaseURL)
		Expect(mountebankClient).NotTo(BeNil(), "Mountebank client should not be nil")

		err := mountebankClient.Init(30 * time.Second) // Use Init method which handles waiting and storing
		Expect(err).NotTo(HaveOccurred(), "Mountebank client did not initialize correctly")
	})

	Context("Hello World SOAP Imposter functionality", func() {
		const imposterPort = 4547

		// No BeforeEach/AfterEach for imposter creation/deletion
		// as it's assumed to be loaded by mb-config.ejs

		It("should respond to a HelloWorld SOAP request with the correct name", func() {
			soapRequest := `<?xml version="1.0" encoding="utf-8"?>
<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/" xmlns:tem="http://tempuri.org/">
  <soapenv:Header/>
  <soapenv:Body>
    <tem:HelloWorld>
      <tem:Name>Gopher</tem:Name>
    </tem:HelloWorld>
  </soapenv:Body>
</soapenv:Envelope>`
			soapAction := "http://tempuri.org/HelloWorld"
			targetURL := fmt.Sprintf("http://127.0.0.1:%d/soap", imposterPort)

			resp, err := sendSOAPRequest(targetURL, soapAction, soapRequest)
			Expect(err).NotTo(HaveOccurred(), "Failed to send SOAP request")
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			bodyBytes, err := io.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred(), "Failed to read response body")
			Expect(string(bodyBytes)).To(ContainSubstring("Hello Gopher from Mountebank!"))
		})
	})

	Context("Key-Value Store Impostor functionality", func() {
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
})

// sendSOAPRequest is a helper function to send a SOAP request.
func sendSOAPRequest(url, soapAction, requestBody string) (*http.Response, error) {
	req, err := http.NewRequest("POST", url, bytes.NewBufferString(requestBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	req.Header.Set("SOAPAction", soapAction)

	client := &http.Client{Timeout: 5 * time.Second}
	return client.Do(req)
}

// sendJSONPostRequest is a helper function to send a JSON POST request.
func sendJSONPostRequest(url, requestBody string) (*http.Response, error) {
	req, err := http.NewRequest("POST", url, bytes.NewBufferString(requestBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	return client.Do(req)
}

// sendJSONGetRequest is a helper function to send a JSON GET request.
func sendJSONGetRequest(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	return client.Do(req)
}
