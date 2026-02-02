package e2e

import (
	"fmt"
	"io"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// NOTE: This file is part of the Mountebank E2E test suite.
// The BeforeAll setup and helper functions are defined in mountebank_suite_test.go.

var _ = Context("Hello World SOAP Imposter functionality", func() {
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

		resp, err := sendSOAPRequest(testFramework.MountebankClient.HTTPClient, targetURL, soapAction, soapRequest)
		Expect(err).NotTo(HaveOccurred(), "Failed to send SOAP request")
		Expect(resp.StatusCode).To(Equal(http.StatusOK))

		bodyBytes, err := io.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred(), "Failed to read response body")
		Expect(string(bodyBytes)).To(ContainSubstring("Hello Gopher from Mountebank!"))
	})
})
