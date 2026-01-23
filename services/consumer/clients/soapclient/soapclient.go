package soapclient

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"

	"time"

	"kafka-soap-e2e-test/services/consumer/models"
)

// SOAPClient holds the HTTP client and SOAP service URL
type SOAPClient struct {
	client *http.Client
	url    string
}

// NewSOAPClient initializes a new SOAPClient
func NewSOAPClient(url string) *SOAPClient {
	log.Printf("Consumer Service: Initializing SOAP client with URL: %s", url)
	return &SOAPClient{
		client: &http.Client{Timeout: 10 * time.Second},
		url:    url,
	}
}

// doSOAPRequest is a helper to send SOAP requests and return the raw response body
func (sc *SOAPClient) doSOAPRequest(soapAction string, requestBody []byte) ([]byte, error) {
	req, err := http.NewRequest("POST", sc.url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create SOAP request: %w", err)
	}
	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	req.Header.Set("SOAPAction", soapAction)

	log.Printf("Consumer Service: Sending HTTP request to SOAP service. URL: %s, SOAPAction: %s", sc.url, soapAction)
	resp, err := sc.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call SOAP service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		log.Printf("Consumer Service: SOAP service returned non-OK status: %d. Response: %s", resp.StatusCode, string(respBody))
		return nil, fmt.Errorf("SOAP service returned non-OK status: %d", resp.StatusCode)
	}

	soapResponseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read SOAP response body: %w", err)
	}
	log.Printf("Consumer Service: Received SOAP response. Body length: %d", len(soapResponseBody))
	return soapResponseBody, nil
}

// ReadKYC performs a KYCQuery operation
func (sc *SOAPClient) ReadKYC(clientID string) (models.UserData, error) {
	requestTemplate := `<?xml version="1.0" encoding="UTF-8"?>
<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/" xmlns="http://example.com/kyc">
  <soapenv:Header>
    <KYCRequest xmlns="http://example.com/kyc"/>
  </soapenv:Header>
  <soapenv:Body>
    <KYCQuery xmlns="http://example.com/kyc">
      <ClientID>%s</ClientID>
    </KYCQuery>
  </soapenv:Body>
</soapenv:Envelope>`
	requestBody := fmt.Sprintf(requestTemplate, clientID)
	soapAction := "http://example.com/kyc/KYCQuery"

	respBody, err := sc.doSOAPRequest(soapAction, []byte(requestBody))
	if err != nil {
		return models.UserData{}, err
	}

	var envelope models.StandardKYCResponseEnvelope
	if err := xml.Unmarshal(respBody, &envelope); err != nil {
		log.Printf("Consumer Service: Failed to unmarshal KYC Read response: %v", err)
		return models.UserData{}, fmt.Errorf("failed to parse SOAP response: %w", err)
	}

	// Initialize userData with status and message from the response, regardless of UserData presence
	userData := models.UserData{
		Status:  envelope.Body.KYCResult.Status,
		Message: envelope.Body.KYCResult.Message,
	}

	// If UserData is present in the response, populate it
	if envelope.Body.KYCResult.UserData != nil {
		userData.ClientID = envelope.Body.KYCResult.UserData.ClientID
		userData.Risk = envelope.Body.KYCResult.UserData.Risk
	} else {
		// If no UserData is returned, but the status is "Error", it's a valid scenario.
		// We still need to populate the ClientID from the request if it's an error response
		// and the clientID was the subject of the query.
		if userData.Status == "Error" {
			userData.ClientID = clientID // Assign the requested clientID to the error response
		}
	}
	return userData, nil
}

// CreateKYC performs a CreateKYC operation
func (sc *SOAPClient) CreateKYC(userData models.UserData) (models.UserData, error) {
	requestTemplate := `<?xml version="1.0" encoding="UTF-8"?>
<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/" xmlns="http://example.com/kyc">
  <soapenv:Header>
    <KYCRequest xmlns="http://example.com/kyc"/>
  </soapenv:Header>
  <soapenv:Body>
    <CreateKYC xmlns="http://example.com/kyc">
      <UserData>
        <ClientID>%s</ClientID>
        <Risk>%f</Risk>
        <!-- Include other UserData fields as needed for creation -->
      </UserData>
    </CreateKYC>
  </soapenv:Body>
</soapenv:Envelope>`
	// Note: You might want to marshal userData directly to XML for more robust handling of all fields.
	// For simplicity, using Sprintf for ClientID and Risk here.
	requestBody := fmt.Sprintf(requestTemplate, userData.ClientID, userData.Risk)
	soapAction := "http://example.com/kyc/CreateKYC"

	respBody, err := sc.doSOAPRequest(soapAction, []byte(requestBody))
	if err != nil {
		return models.UserData{}, err
	}

	var envelope models.StandardKYCResponseEnvelope
	if err := xml.Unmarshal(respBody, &envelope); err != nil {
		log.Printf("Consumer Service: Failed to unmarshal KYC Create response: %v", err)
		return models.UserData{}, fmt.Errorf("failed to parse SOAP response: %w", err)
	}

	if envelope.Body.KYCResult.UserData == nil {
		return models.UserData{}, fmt.Errorf("no UserData found in KYC Create response")
	}

	createdUserData := *envelope.Body.KYCResult.UserData
	createdUserData.Status = envelope.Body.KYCResult.Status
	createdUserData.Message = envelope.Body.KYCResult.Message
	return createdUserData, nil
}

// UpdateKYC performs an UpdateKYC operation
func (sc *SOAPClient) UpdateKYC(userData models.UserData) (models.UserData, error) {
	requestTemplate := `<?xml version="1.0" encoding="UTF-8"?>
<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/" xmlns="http://example.com/kyc">
  <soapenv:Header>
    <KYCRequest xmlns="http://example.com/kyc"/>
  </soapenv:Header>
  <soapenv:Body>
    <UpdateKYC xmlns="http://example.com/kyc">
      <UserData>
        <ClientID>%s</ClientID>
        <Risk>%f</Risk>
        <!-- Include other UserData fields as needed for update -->
      </UserData>
    </UpdateKYC>
  </soapenv:Body>
</soapenv:Envelope>`
	requestBody := fmt.Sprintf(requestTemplate, userData.ClientID, userData.Risk)
	soapAction := "http://example.com/kyc/UpdateKYC"

	respBody, err := sc.doSOAPRequest(soapAction, []byte(requestBody))
	if err != nil {
		return models.UserData{}, err
	}

	var envelope models.StandardKYCResponseEnvelope
	if err := xml.Unmarshal(respBody, &envelope); err != nil {
		log.Printf("Consumer Service: Failed to unmarshal KYC Update response: %v", err)
		return models.UserData{}, fmt.Errorf("failed to parse SOAP response: %w", err)
	}

	if envelope.Body.KYCResult.UserData == nil {
		return models.UserData{}, fmt.Errorf("no UserData found in KYC Update response")
	}

	updatedUserData := *envelope.Body.KYCResult.UserData
	updatedUserData.Status = envelope.Body.KYCResult.Status
	updatedUserData.Message = envelope.Body.KYCResult.Message
	return updatedUserData, nil
}

// DeleteKYC performs a DeleteKYC operation
func (sc *SOAPClient) DeleteKYC(clientID string) (string, error) {
	requestTemplate := `<?xml version="1.0" encoding="UTF-8"?>
<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/" xmlns="http://example.com/kyc">
  <soapenv:Header>
    <KYCRequest xmlns="http://example.com/kyc"/>
  </soapenv:Header>
  <soapenv:Body>
    <DeleteKYC xmlns="http://example.com/kyc">
      <ClientID>%s</ClientID>
    </DeleteKYC>
  </soapenv:Body>
</soapenv:Envelope>`
	requestBody := fmt.Sprintf(requestTemplate, clientID)
	soapAction := "http://example.com/kyc/DeleteKYC"

	respBody, err := sc.doSOAPRequest(soapAction, []byte(requestBody))
	if err != nil {
		return "", err
	}

	var envelope models.DeleteKYCResponseEnvelope
	if err := xml.Unmarshal(respBody, &envelope); err != nil {
		log.Printf("Consumer Service: Failed to unmarshal KYC Delete response: %v", err)
		return "", fmt.Errorf("failed to parse SOAP response: %w", err)
	}

	return envelope.Body.DeleteKYCResult.Message, nil
}

// The old CallSOAPService and ParseSOAPResponse are removed as they are replaced by the new CRUD specific functions.
