package e2e

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"net/http"

	. "github.com/onsi/gomega"
)

// SoapEnvelope defines the structure for a SOAP 1.1 Envelope
type SoapEnvelope struct {
	XMLName      xml.Name    `xml:"soapenv:Envelope"`
	XmlnsSoapEnv string      `xml:"xmlns:soapenv,attr"`
	XmlnsTem     string      `xml:"xmlns:tem,attr"`
	Header       *SoapHeader `xml:"soapenv:Header"`
	Body         SoapBody    `xml:"soapenv:Body"`
}

// SoapHeader defines the structure for a SOAP Header
type SoapHeader struct {
	XMLName xml.Name `xml:"soapenv:Header"`
	// Header can be empty or contain specific SOAP header elements
}

// SoapBody defines the structure for a SOAP Body
type SoapBody struct {
	XMLName    xml.Name           `xml:"soapenv:Body"`
	StoreOp    *StoreOperation    `xml:"tem:Store,omitempty"`
	RetrieveOp *RetrieveOperation `xml:"tem:Retrieve,omitempty"`
	StoreXMLOp *StoreXMLOperation `xml:"tem:StoreXML,omitempty"` // New field for storing XML
}

// StoreOperation defines the structure for the <tem:Store> element
type StoreOperation struct {
	XMLName xml.Name `xml:"tem:Store"`
	Key     string   `xml:"tem:Key"`
	Value   string   `xml:"tem:Value"`
}

// StoreXMLOperation defines the structure for the <tem:StoreXML> element
type StoreXMLOperation struct {
	XMLName xml.Name `xml:"tem:StoreXML"`
	Key     string   `xml:"tem:Key"`
	Data    string   `xml:"tem:Data"` // This will hold the raw XML string
}

// RetrieveOperation defines the structure for the <tem:Retrieve> element
type RetrieveOperation struct {
	XMLName xml.Name `xml:"tem:Retrieve"`
	Key     string   `xml:"tem:Key"`
}

// generateStoreXMLSOAPRequest generates a SOAP request for the StoreXML operation.

func generateStoreXMLSOAPRequest(key, xmlData string) string {
	envelope := SoapEnvelope{
		XmlnsSoapEnv: "http://schemas.xmlsoap.org/soap/envelope/",

		XmlnsTem: "http://tempuri.org/",

		Header: &SoapHeader{},

		Body: SoapBody{
			StoreXMLOp: &StoreXMLOperation{
				Key: key,

				Data: xmlData,
			},
		},
	}

	output, err := xml.MarshalIndent(envelope, "", "  ")

	Expect(err).NotTo(HaveOccurred(), "Failed to marshal StoreXML SOAP request")

	return fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>

%s`, string(output))
}

func generateRetrieveSOAPRequest(key string) string {
	envelope := SoapEnvelope{
		XmlnsSoapEnv: "http://schemas.xmlsoap.org/soap/envelope/",
		XmlnsTem:     "http://tempuri.org/",
		Header:       &SoapHeader{},
		Body: SoapBody{
			RetrieveOp: &RetrieveOperation{
				Key: key,
			},
		},
	}

	output, err := xml.MarshalIndent(envelope, "", "  ")
	Expect(err).NotTo(HaveOccurred(), "Failed to marshal Retrieve SOAP request")

	return fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
%s`, string(output))
}

func generateMissingKeyStoreSOAPRequest(xmlData string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/" xmlns:tem="http://tempuri.org/">
  <soapenv:Header/>
  <soapenv:Body>
    <tem:StoreXML>
      <!-- tem:Key is missing -->
      <tem:Data>%s</tem:Data>
    </tem:StoreXML>
  </soapenv:Body>
</soapenv:Envelope>`, xmlData)
}

func generateMissingValueStoreSOAPRequest(key string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/" xmlns:tem="http://tempuri.org/">
  <soapenv:Header/>
  <soapenv:Body>
    <tem:StoreXML>
      <tem:Key>%s</tem:Key>
      <!-- tem:Data is missing -->
    </tem:StoreXML>
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

func sendSOAPRequest(httpClient *http.Client, url, soapAction, requestBody string) (*http.Response, error) {
	req, err := http.NewRequest("POST", url, bytes.NewBufferString(requestBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/soap+xml")

	req.Header.Set("SOAPAction", soapAction)

	return httpClient.Do(req)
}

// sendJSONPostRequestHelper is a helper function to send a JSON POST request.

func sendJSONPostRequestHelper(httpClient *http.Client, url, requestBody string) (*http.Response, error) {
	req, err := http.NewRequest("POST", url, bytes.NewBufferString(requestBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	return httpClient.Do(req)
}

// sendJSONGetRequestHelper is a helper function to send a JSON GET request.

func sendJSONGetRequestHelper(httpClient *http.Client, url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")

	return httpClient.Do(req)
}
