package models

import (
	"encoding/xml"
	consumerModels "kafka-soap-e2e-test/services/consumer/models" // Alias to avoid conflict with this package's name
)

// --- SOAP Request/Response Models for this Service ---

// Generic SOAP Envelope for parsing incoming requests
type SOAPEnvelope struct {
	XMLName xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Envelope"`
	Body    SOAPBody
}

type SOAPBody struct {
	XMLName xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Body"`
	Content []byte   `xml:",innerxml"` // To unmarshal the actual payload of the operation
}

// KYCQuery - For Read operation
type KYCQuery struct {
	XMLName  xml.Name `xml:"http://example.com/kyc KYCQuery"`
	ClientID string   `xml:"ClientID"`
}

// CreateKYCRequest - For Create operation, expects full UserData
type CreateKYCRequest struct {
	XMLName  xml.Name                `xml:"http://example.com/kyc CreateKYC"`
	UserData consumerModels.UserData `xml:"UserData"`
}

// UpdateKYCRequest - For Update operation, expects full UserData
type UpdateKYCRequest struct {
	XMLName  xml.Name                `xml:"http://example.com/kyc UpdateKYC"`
	UserData consumerModels.UserData `xml:"UserData"`
}

// DeleteKYCRequest - For Delete operation, expects ClientID
type DeleteKYCRequest struct {
	XMLName  xml.Name `xml:"http://example.com/kyc DeleteKYC"`
	ClientID string   `xml:"ClientID"`
}

// KYCResponseEnvelope for Read, Update, Create operations
type KYCResponseEnvelope struct {
	XMLName      xml.Name `xml:"soapenv:Envelope"`
	XmlnsSoapenv string   `xml:"xmlns:soapenv,attr"`
	Body         KYCResponseBody
}

type KYCResponseBody struct {
	XMLName   xml.Name  `xml:"soapenv:Body"`
	KYCResult KYCResult `xml:"http://example.com/kyc KYCResponse"` // Use full namespace
}

type KYCResult struct {
	XMLName  xml.Name                 `xml:"http://example.com/kyc KYCResponse"` // Use full namespace
	XmlnsKyc string                   `xml:"xmlns"`
	Status   string                   `xml:"Status"`
	Message  string                   `xml:"Message"`
	UserData *consumerModels.UserData `xml:"UserData,omitempty"` // Pointer to optionally include UserData
}

// DeleteKYCResponse - specific for delete
type DeleteKYCResponseEnvelope struct {
	XMLName      xml.Name `xml:"soapenv:Envelope"`
	XmlnsSoapenv string   `xml:"xmlns:soapenv,attr"`
	Body         DeleteKYCResponseBody
}

type DeleteKYCResponseBody struct {
	XMLName         xml.Name        `xml:"soapenv:Body"`
	DeleteKYCResult DeleteKYCResult `xml:"http://example.com/kyc DeleteKYCResponse"` // Use full namespace
}

type DeleteKYCResult struct {
	XMLName  xml.Name `xml:"http://example.com/kyc DeleteKYCResponse"` // Use full namespace
	XmlnsKyc string   `xml:"xmlns"`
	Status   string   `xml:"Status"`
	Message  string   `xml:"Message"`
}
