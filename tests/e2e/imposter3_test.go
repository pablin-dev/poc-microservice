package e2e

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// NOTE: This file is part of the Mountebank E2E test suite.
// The BeforeAll setup and helper functions are defined in mountebank_suite_test.go.

const apiNamespace = "http://api.com"

// Define SOAP structures for the API namespace
type APICreateCustomer struct {
	XMLName          xml.Name         `xml:"api:CreateCustomer"`
	ID               string           `xml:"api:id,omitempty"`
	FirstName        string           `xml:"api:firstName,omitempty"`
	LastName         string           `xml:"api:lastName,omitempty"`
	Email            string           `xml:"api:email,omitempty"`
	Phone            string           `xml:"api:phone,omitempty"`
	Address          string           `xml:"api:address,omitempty"`
	City             string           `xml:"api:city,omitempty"`
	State            string           `xml:"api:state,omitempty"`
	ZipCode          string           `xml:"api:zipCode,omitempty"`
	Country          string           `xml:"api:country,omitempty"`
	DateOfBirth      string           `xml:"api:dateOfBirth,omitempty"`
	AccountStatus    string           `xml:"api:accountStatus,omitempty"`
	MembershipLevel  string           `xml:"api:membershipLevel,omitempty"`
	LastActivity     string           `xml:"api:lastActivity,omitempty"`
	RegistrationDate string           `xml:"api:registrationDate,omitempty"`
	PreferredContact string           `xml:"api:preferredContact,omitempty"`
	CustomFields     []APICustomField `xml:"api:CustomField,omitempty"`
}

type APICustomField struct {
	XMLName xml.Name `xml:"api:CustomField"`
	Key     string   `xml:"api:key"`
	Value   string   `xml:"api:value"`
}

type APICreateCustomerResponse struct {
	ReturnCode    string `xml:"http://api.com returnCode"`
	ReturnMessage string `xml:"http://api.com returnMessage"`
	CustID        string `xml:"http://api.com custId"`
}

type APIGetCustomerInformation struct {
	XMLName xml.Name `xml:"api:GetCustomerInformation"`
	CustID  string   `xml:"api:custId"`
}

type APIGetCustomerInformationResponse struct {
	ReturnCode    string           `xml:"http://api.com returnCode"`
	ReturnMessage string           `xml:"http://api.com returnMessage"`
	CustomerData  *APICustomerData `xml:"http://api.com CustomerData,omitempty"`
}

type APICustomerData struct {
	ID               string           `xml:"http://api.com id,omitempty"`
	FirstName        string           `xml:"http://api.com firstName,omitempty"`
	LastName         string           `xml:"http://api.com lastName,omitempty"`
	Email            string           `xml:"http://api.com email,omitempty"`
	Phone            string           `xml:"http://api.com phone,omitempty"`
	Address          string           `xml:"http://api.com address,omitempty"`
	City             string           `xml:"http://api.com city,omitempty"`
	State            string           `xml:"http://api.com state,omitempty"`
	ZipCode          string           `xml:"http://api.com zipCode,omitempty"`
	Country          string           `xml:"http://api.com country,omitempty"`
	DateOfBirth      string           `xml:"http://api.com dateOfBirth,omitempty"`
	AccountStatus    string           `xml:"http://api.com accountStatus,omitempty"`
	MembershipLevel  string           `xml:"http://api.com membershipLevel,omitempty"`
	LastActivity     string           `xml:"http://api.com lastActivity,omitempty"`
	RegistrationDate string           `xml:"http://api.com registrationDate,omitempty"`
	PreferredContact string           `xml:"http://api.com preferredContact,omitempty"`
	CustomFields     []APICustomField `xml:"http://api.com CustomField,omitempty"`
}

// Generic SOAP Envelope and Body for unmarshaling responses
type SoapEnvelopeResponse struct {
	XMLName xml.Name         `xml:"http://schemas.xmlsoap.org/soap/envelope/ Envelope"`
	Body    SoapBodyResponse `xml:"Body"`
}

type SoapBodyResponse struct {
	XMLName                        xml.Name                           `xml:"Body"`
	CreateCustomerResponse         *APICreateCustomerResponse         `xml:"http://api.com CreateCustomerResponse,omitempty"`
	GetCustomerInformationResponse *APIGetCustomerInformationResponse `xml:"http://api.com GetCustomerInformationResponse,omitempty"`
	// Add other response types here as needed
}

// Helper to generate SOAP request for CreateCustomer
func generateCreateCustomerSOAPRequest(customer APICreateCustomer) string {
	bodyContent, err := xml.Marshal(customer)
	Expect(err).NotTo(HaveOccurred(), "Failed to marshal CreateCustomer body content")

	return fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/" xmlns:api="%s">
  <soapenv:Header/>
  <soapenv:Body>
    %s
  </soapenv:Body>
</soapenv:Envelope>`, apiNamespace, string(bodyContent))
}

// Helper to generate SOAP request for GetCustomerInformation
func generateGetCustomerInformationSOAPRequest(customer APIGetCustomerInformation) string {
	bodyContent, err := xml.Marshal(customer)
	Expect(err).NotTo(HaveOccurred(), "Failed to marshal GetCustomerInformation body content")

	return fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/" xmlns:api="%s">
  <soapenv:Header/>
  <soapenv:Body>
    %s
  </soapenv:Body>
</soapenv:Envelope>`, apiNamespace, string(bodyContent))
}

var _ = Context("Customer Information SOAP Imposter functionality", func() {
	const imposterPort = 4548
	targetURL := fmt.Sprintf("http://127.0.0.1:%d/ws", imposterPort)

	It("should successfully create a customer", func() {
		By("Sending SOAP request to create a customer")
		customer := APICreateCustomer{
			ID:        "test-cust-123",
			FirstName: "John",
			LastName:  "Doe",
			Email:     "john.doe@example.com",
			Phone:     "123-456-7890",
			CustomFields: []APICustomField{
				{Key: "source", Value: "e2e-test"},
			},
		}
		soapRequest := generateCreateCustomerSOAPRequest(customer)
		soapAction := fmt.Sprintf("%s/CreateCustomer", apiNamespace)

		resp, err := sendSOAPRequest(testFramework.MountebankClient.HTTPClient, targetURL, soapAction, soapRequest)
		Expect(err).NotTo(HaveOccurred(), "Failed to send CreateCustomer SOAP request")
		Expect(resp.StatusCode).To(Equal(http.StatusOK))

		bodyBytes, err := io.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred(), "Failed to read response body")
		var soapResp SoapEnvelopeResponse
		err = xml.Unmarshal(bodyBytes, &soapResp)
		Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal SOAP envelope for CreateCustomerResponse: %s\nResponse Body: %s", err, string(bodyBytes))
		Expect(soapResp.Body.CreateCustomerResponse).NotTo(BeNil(), "CreateCustomerResponse should not be nil")

		Expect(soapResp.Body.CreateCustomerResponse.ReturnCode).To(Equal("0"))
		Expect(soapResp.Body.CreateCustomerResponse.ReturnMessage).To(Equal("Customer created successfully"))
		Expect(soapResp.Body.CreateCustomerResponse.CustID).To(Equal("test-cust-123"))
	})

	It("should successfully retrieve customer information for an existing customer", func() {
		By("Ensuring a customer is created first for retrieval test")
		customerToCreate := APICreateCustomer{
			ID:        "retrieve-cust-456",
			FirstName: "Jane",
			LastName:  "Smith",
			Email:     "jane.smith@example.com",
		}
		createRequest := generateCreateCustomerSOAPRequest(customerToCreate)
		createAction := fmt.Sprintf("%s/CreateCustomer", apiNamespace)
		_, err := sendSOAPRequest(testFramework.MountebankClient.HTTPClient, targetURL, createAction, createRequest)
		Expect(err).NotTo(HaveOccurred(), "Failed to send initial CreateCustomer SOAP request for retrieval")

		By("Sending SOAP request to retrieve customer information")
		customerToRetrieve := APIGetCustomerInformation{
			CustID: "retrieve-cust-456",
		}
		retrieveRequest := generateGetCustomerInformationSOAPRequest(customerToRetrieve)
		retrieveAction := fmt.Sprintf("%s/GetCustomerInformation", apiNamespace)

		resp, err := sendSOAPRequest(testFramework.MountebankClient.HTTPClient, targetURL, retrieveAction, retrieveRequest)
		Expect(err).NotTo(HaveOccurred(), "Failed to send GetCustomerInformation SOAP request")
		Expect(resp.StatusCode).To(Equal(http.StatusOK))

		bodyBytes, err := io.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred(), "Failed to read response body")

		var soapResp SoapEnvelopeResponse
		err = xml.Unmarshal(bodyBytes, &soapResp)
		Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal SOAP envelope for GetCustomerInformationResponse: %s\nResponse Body: %s", err, string(bodyBytes))
		Expect(soapResp.Body.GetCustomerInformationResponse).NotTo(BeNil(), "GetCustomerInformationResponse should not be nil")
		Expect(soapResp.Body.GetCustomerInformationResponse.ReturnCode).To(Equal("0"))
		Expect(soapResp.Body.GetCustomerInformationResponse.ReturnMessage).To(Equal("Customer information retrieved successfully"))
		Expect(soapResp.Body.GetCustomerInformationResponse.CustomerData).NotTo(BeNil())
		Expect(soapResp.Body.GetCustomerInformationResponse.CustomerData.ID).To(Equal("retrieve-cust-456"))
		Expect(soapResp.Body.GetCustomerInformationResponse.CustomerData.FirstName).To(Equal("Jane"))
		Expect(soapResp.Body.GetCustomerInformationResponse.CustomerData.Email).To(Equal("jane.smith@example.com"))
	})

	It("should return an error for retrieving information of a non-existent customer", func() {
		By("Sending SOAP request for a non-existent customer ID")
		customerToRetrieve := APIGetCustomerInformation{
			CustID: "non-existent-cust-789",
		}
		retrieveRequest := generateGetCustomerInformationSOAPRequest(customerToRetrieve)
		retrieveAction := fmt.Sprintf("%s/GetCustomerInformation", apiNamespace)

		resp, err := sendSOAPRequest(testFramework.MountebankClient.HTTPClient, targetURL, retrieveAction, retrieveRequest)
		Expect(err).NotTo(HaveOccurred(), "Failed to send GetCustomerInformation SOAP request for non-existent customer")
		Expect(resp.StatusCode).To(Equal(http.StatusOK)) // Mountebank returns 200 with an error in the SOAP body

		bodyBytes, err := io.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred(), "Failed to read response body")

		var soapResp SoapEnvelopeResponse
		err = xml.Unmarshal(bodyBytes, &soapResp)
		Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal SOAP envelope for GetCustomerInformationResponse: %s\nResponse Body: %s", err, string(bodyBytes))
		Expect(soapResp.Body.GetCustomerInformationResponse).NotTo(BeNil(), "GetCustomerInformationResponse should not be nil")

		Expect(soapResp.Body.GetCustomerInformationResponse.ReturnCode).To(Equal("1"))
		Expect(soapResp.Body.GetCustomerInformationResponse.ReturnMessage).To(ContainSubstring("Customer with ID non-existent-cust-789 not found."))
		Expect(soapResp.Body.GetCustomerInformationResponse.CustomerData).To(BeNil())
	})
})
