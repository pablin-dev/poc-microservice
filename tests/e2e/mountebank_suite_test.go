package e2e_test

import (
	"bytes"
	"fmt"
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
})

// Helper functions for generating SOAP request bodies
func generateStoreSOAPRequest(key, value string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/" xmlns:tem="http://tempuri.org/">
  <soapenv:Header/>
  <soapenv:Body>
    <tem:Store>
      <tem:Key>%s</tem:Key>
      <tem:Value>%s</tem:Value>
    </tem:Store>
  </soapenv:Body>
</soapenv:Envelope>`, key, value)
}

func generateRetrieveSOAPRequest(key string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/" xmlns:tem="http://tempuri.org/">
  <soapenv:Header/>
  <soapenv:Body>
    <tem:Retrieve>
      <tem:Key>%s</tem:Key>
    </tem:Retrieve>
  </soapenv:Body>
</soapenv:Envelope>`, key)
}

func generateMissingKeyStoreSOAPRequest(value string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/" xmlns:tem="http://tempuri.org/">
  <soapenv:Header/>
  <soapenv:Body>
    <tem:Store>
      <!-- tem:Key is missing -->
      <tem:Value>%s</tem:Value>
    </tem:Store>
  </soapenv:Body>
</soapenv:Envelope>`, value)
}

func generateMissingValueStoreSOAPRequest(key string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/" xmlns:tem="http://tempuri.org/">
  <soapenv:Header/>
  <soapenv:Body>
    <tem:Store>
      <tem:Key>%s</tem:Key>
      <!-- tem:Value is missing -->
    </tem:Store>
  </soapenv:Body>
</soapenv:Envelope>`, key)
}

func generateMissingKeyRetrieveSOAPRequest() string {
	return `<?xml version="1.0" encoding="utf-8"?>
<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/" xmlns:tem="http://tempuri.org/">
  <soapenv:Header/>
  <soapenv:Body>
    <tem:Retrieve>
      <!-- tem:Key is missing -->
    </tem:Retrieve>
  </soapenv:Body>
</soapenv:Envelope>`
}

func generateUnknownActionSOAPRequest() string {
	return `<?xml version="1.0" encoding="utf-8"?>
<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/" xmlns:tem="http://tempuri.org/">
  <soapenv:Header/>
  <soapenv:Body>
    <tem:UnknownRequest>
      <tem:Data>someData</tem:Data>
    </tem:UnknownRequest>
  </soapenv:Body>
</soapenv:Envelope>`
}

// sendSOAPRequest is a helper function to send a SOAP request.

func sendSOAPRequest(url, soapAction, requestBody string) (*http.Response, error) {
	req, err := http.NewRequest("POST", url, bytes.NewBufferString(requestBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/soap+xml")
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
