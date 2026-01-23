package models

import "encoding/xml"

// Incoming Kafka message structure
type KafkaMessage struct {
	Type          string `json:"type"` // "Process" or "KYC" (e.g., "READ", "CREATE", "UPDATE", "DELETE")
	CorrelationID string `json:"correlationId"`
	ClientID      string `json:"clientId,omitempty"` // For KYC type
	// Add other fields as needed for specific request types
	UserData UserData `json:"userData,omitempty"` // For Create/Update operations
}

// UserData struct representing a comprehensive KYC response for Kafka and internal use
type UserData struct {
	XMLName  xml.Name `xml:"UserData"` // Added for proper unmarshalling when embedded
	ClientID string   `xml:"ClientID" json:"clientId"`
	Risk     float64  `xml:"Risk" json:"risk"`
	// Status and Message are part of the overall response, not typically within UserData itself
	// in the KYC service's UserData struct, but included here if needed for client-side processing
	Status  string `json:"status,omitempty"`
	Message string `json:"message,omitempty"`
}

// --- Structures for Standard KYC SOAP Responses (Read, Create, Update) ---

// StandardKYCResponseEnvelope is the top-level SOAP envelope for non-delete KYC responses
type StandardKYCResponseEnvelope struct {
	XMLName      xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Envelope"`
	XmlnsSoapenv string   `xml:"xmlns:soapenv,attr"`
	Body         StandardKYCResponseBody
}

// StandardKYCResponseBody contains the actual KYCResult
type StandardKYCResponseBody struct {
	XMLName   xml.Name          `xml:"http://schemas.xmlsoap.org/soap/envelope/ Body"`
	KYCResult StandardKYCResult `xml:"http://example.com/kyc KYCResponse"`
}

// StandardKYCResult contains the status, message, and optional UserData for standard operations
type StandardKYCResult struct {
	XMLName   xml.Name  `xml:"http://example.com/kyc KYCResponse"`
	XmlnsKyc string    `xml:"xmlns,attr"`
	Status    string    `xml:"Status"`
	Message   string    `xml:"Message"`
	UserData  *UserData `xml:"UserData,omitempty"`
}

// --- Structures for Delete KYC SOAP Responses ---

// DeleteKYCResponseEnvelope is the top-level SOAP envelope for delete KYC responses
type DeleteKYCResponseEnvelope struct {
	XMLName      xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Envelope"`
	XmlnsSoapenv string   `xml:"xmlns:soapenv,attr"`
	Body         DeleteKYCResponseBody
}

// DeleteKYCResponseBody contains the DeleteKYCResult
type DeleteKYCResponseBody struct {
	XMLName         xml.Name        `xml:"http://schemas.xmlsoap.org/soap/envelope/ Body"`
	DeleteKYCResult DeleteKYCResult `xml:"http://example.com/kyc DeleteKYCResponse"`
}

// DeleteKYCResult contains the status and message for delete operations
type DeleteKYCResult struct {
	XMLName   xml.Name `xml:"http://example.com/kyc DeleteKYCResponse"`
	XmlnsKyc string   `xml:"xmlns,attr"`
	Status    string   `xml:"Status"`
	Message   string   `xml:"Message"`
}
