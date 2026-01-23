package main

import (
	"encoding/xml"
	"fmt"

	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"kafka-soap-e2e-test/services/consumer/models"

	"github.com/stretchr/testify/assert"
	kycModels "kafka-soap-e2e-test/services/providers/kyc/models"
)

// Helper function to create a SOAP request XML
func createSOAPRequest(operation string, clientID string, userData *models.UserData) string {
	var bodyContent string
	switch operation {
	case "KYCQuery":
		bodyContent = fmt.Sprintf(`<%s xmlns="%s"><ClientID>%s</ClientID></%s>`, operation, kycNamespaceAttr, clientID, operation)
	case "CreateKYC", "UpdateKYC":
		if userData == nil {
			userData = &models.UserData{ClientID: clientID, Risk: 0.0, Status: "Pending", Message: "Default message"}
		}
		userDataXML, _ := xml.Marshal(userData)
		bodyContent = fmt.Sprintf(`<%s xmlns="%s">%s</%s>`, operation, kycNamespaceAttr, string(userDataXML), operation)
	case "DeleteKYC":
		bodyContent = fmt.Sprintf(`<%s xmlns="%s"><ClientID>%s</ClientID></%s>`, operation, kycNamespaceAttr, clientID, operation)
	default:
		bodyContent = fmt.Sprintf(`<%s xmlns="%s"></%s>`, operation, kycNamespaceAttr, operation) // Unknown operation
	}

	return fmt.Sprintf(
		`%s
  %s
    %s
  %s
%s`,
		soapEnvelopeStart,
		soapBodyStart,
		bodyContent,
		soapBodyEnd,
		soapEnvelopeEnd,
	)
}

// Helper to unmarshal SOAP response
func unmarshalSOAPResponse(responseBody []byte, target interface{}) error {
	var envelope kycModels.SOAPEnvelope
	if err := xml.Unmarshal(responseBody, &envelope); err != nil {
		return fmt.Errorf("failed to unmarshal SOAP envelope: %w", err)
	}
	if len(envelope.Body.Content) == 0 {
		return fmt.Errorf("SOAP body content is empty")
	}
	return xml.Unmarshal(envelope.Body.Content, target)
}

func TestSOAPHandler_KYCQuery(t *testing.T) {
	repo := NewInMemoryRepo()
	assert.NoError(t, repo.Create(models.UserData{ClientID: "client123", Risk: 0.7, Status: "Active", Message: "OK"}))

	tests := []struct {
		name             string
		clientID         string
		expectedCode     int
		expectedStatus   string
		expectedMessage  string
		expectedUserData *models.UserData
	}{
		{
			name:             "Successful KYCQuery",
			clientID:         "client123",
			expectedCode:     http.StatusOK,
			expectedStatus:   "Success",
			expectedMessage:  "User data retrieved",
			expectedUserData: &models.UserData{ClientID: "client123", Risk: 0.7, Status: "Active", Message: "OK"},
		},
		{
			name:             "KYCQuery Client Not Found",
			clientID:         "nonexistent",
			expectedCode:     http.StatusNotFound,
			expectedStatus:   "Error",
			expectedMessage:  "user with ClientID 'nonexistent' not found",
			expectedUserData: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody := createSOAPRequest("KYCQuery", tt.clientID, nil)
			req := httptest.NewRequest("POST", "/soap", strings.NewReader(reqBody))
			rec := httptest.NewRecorder()

			soapHandler(repo, rec, req)

			assert.Equal(t, tt.expectedCode, rec.Code)
			var response kycModels.KYCResult
			err := unmarshalSOAPResponse(rec.Body.Bytes(), &response)
			assert.NoError(t, err)

			assert.Equal(t, tt.expectedStatus, response.Status)
			assert.Contains(t, response.Message, tt.expectedMessage)
			if tt.expectedUserData != nil {
				assert.Equal(t, tt.expectedUserData, response.UserData)
			} else {
				assert.Nil(t, response.UserData)
			}
		})
	}
}

func TestSOAPHandler_CreateKYC(t *testing.T) {
	repo := NewInMemoryRepo()

	tests := []struct {
		name            string
		userData        models.UserData
		expectedCode    int
		expectedStatus  string
		expectedMessage string
	}{
		{
			name:            "Successful CreateKYC",
			userData:        models.UserData{ClientID: "newclient", Risk: 0.1, Status: "Pending", Message: "New user"},
			expectedCode:    http.StatusOK,
			expectedStatus:  "Success",
			expectedMessage: "User created",
		},
		{
			name:            "CreateKYC Already Exists",
			userData:        models.UserData{ClientID: "client123", Risk: 0.2, Status: "Active", Message: "Existing user"},
			expectedCode:    http.StatusConflict,
			expectedStatus:  "Error",
			expectedMessage: "user with ClientID 'client123' already exists",
		},
	}

	// Create a client123 before running tests to test conflict
	assert.NoError(t, repo.Create(models.UserData{ClientID: "client123", Risk: 0.7, Status: "Active", Message: "OK"}))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody := createSOAPRequest("CreateKYC", tt.userData.ClientID, &tt.userData)
			req := httptest.NewRequest("POST", "/soap", strings.NewReader(reqBody))
			rec := httptest.NewRecorder()

			soapHandler(repo, rec, req)

			assert.Equal(t, tt.expectedCode, rec.Code)
			var response kycModels.KYCResult
			err := unmarshalSOAPResponse(rec.Body.Bytes(), &response)
			assert.NoError(t, err)

			assert.Equal(t, tt.expectedStatus, response.Status)
			assert.Contains(t, response.Message, tt.expectedMessage)
			if tt.expectedStatus == "Success" {
				// Risk might be slightly different due to float precision, compare ClientID
				assert.NotNil(t, response.UserData)
				assert.Equal(t, tt.userData.ClientID, response.UserData.ClientID)
				assert.Equal(t, tt.userData.Status, response.UserData.Status)
				assert.Equal(t, tt.userData.Message, response.UserData.Message)
				// Risk is assigned by server, so just check it's a float
				assert.NotZero(t, response.UserData.Risk) // Ensure it's not default 0
			} else {
				assert.Nil(t, response.UserData)
			}
		})
	}
}

func TestSOAPHandler_UpdateKYC(t *testing.T) {
	repo := NewInMemoryRepo()
	assert.NoError(t, repo.Create(models.UserData{ClientID: "client123", Risk: 0.7, Status: "Active", Message: "OK"}))

	tests := []struct {
		name            string
		userData        models.UserData
		expectedCode    int
		expectedStatus  string
		expectedMessage string
	}{
		{
			name:            "Successful UpdateKYC",
			userData:        models.UserData{ClientID: "client123", Risk: 0.9, Status: "Suspended", Message: "Updated message"},
			expectedCode:    http.StatusOK,
			expectedStatus:  "Success",
			expectedMessage: "User updated",
		},
		{
			name:            "UpdateKYC Client Not Found",
			userData:        models.UserData{ClientID: "nonexistent", Risk: 0.1},
			expectedCode:    http.StatusNotFound,
			expectedStatus:  "Error",
			expectedMessage: "user with ClientID 'nonexistent' not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody := createSOAPRequest("UpdateKYC", tt.userData.ClientID, &tt.userData)
			req := httptest.NewRequest("POST", "/soap", strings.NewReader(reqBody))
			rec := httptest.NewRecorder()

			soapHandler(repo, rec, req)

			assert.Equal(t, tt.expectedCode, rec.Code)
			var response kycModels.KYCResult
			err := unmarshalSOAPResponse(rec.Body.Bytes(), &response)
			assert.NoError(t, err)

			assert.Equal(t, tt.expectedStatus, response.Status)
			assert.Contains(t, response.Message, tt.expectedMessage)
			if tt.expectedStatus == "Success" {
				assert.NotNil(t, response.UserData)
				assert.Equal(t, tt.userData.ClientID, response.UserData.ClientID)
				assert.Equal(t, tt.userData.Risk, response.UserData.Risk)
				assert.Equal(t, tt.userData.Status, response.UserData.Status)
				assert.Equal(t, tt.userData.Message, response.UserData.Message)
			} else {
				assert.Nil(t, response.UserData)
			}
		})
	}
}

func TestSOAPHandler_DeleteKYC(t *testing.T) {
	repo := NewInMemoryRepo()
	assert.NoError(t, repo.Create(models.UserData{ClientID: "client123", Risk: 0.7, Status: "Active", Message: "OK"}))

	tests := []struct {
		name            string
		clientID        string
		expectedCode    int
		expectedStatus  string
		expectedMessage string
	}{
		{
			name:            "Successful DeleteKYC",
			clientID:        "client123",
			expectedCode:    http.StatusOK,
			expectedStatus:  "Success",
			expectedMessage: "User deleted",
		},
		{
			name:            "DeleteKYC Client Not Found",
			clientID:        "nonexistent",
			expectedCode:    http.StatusNotFound,
			expectedStatus:  "Error",
			expectedMessage: "user with ClientID 'nonexistent' not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody := createSOAPRequest("DeleteKYC", tt.clientID, nil)
			req := httptest.NewRequest("POST", "/soap", strings.NewReader(reqBody))
			rec := httptest.NewRecorder()

			soapHandler(repo, rec, req)

			assert.Equal(t, tt.expectedCode, rec.Code)
			var response kycModels.DeleteKYCResult // Note: DeleteKYCResult for delete operation
			err := unmarshalSOAPResponse(rec.Body.Bytes(), &response)
			assert.NoError(t, err)

			assert.Equal(t, tt.expectedStatus, response.Status)
			assert.Contains(t, response.Message, tt.expectedMessage)
		})
	}
}

func TestSOAPHandler_InvalidMethod(t *testing.T) {
	repo := NewInMemoryRepo()
	req := httptest.NewRequest("GET", "/soap", nil) // Use GET method
	rec := httptest.NewRecorder()

	soapHandler(repo, rec, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
	assert.Contains(t, rec.Body.String(), "SOAP requests must use POST method")
}

func TestSOAPHandler_InvalidXML(t *testing.T) {
	repo := NewInMemoryRepo()
	reqBody := `this is not xml`
	req := httptest.NewRequest("POST", "/soap", strings.NewReader(reqBody))
	rec := httptest.NewRecorder()

	soapHandler(repo, rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Failed to parse SOAP request")
}

func TestSOAPHandler_UnknownOperation(t *testing.T) {
	repo := NewInMemoryRepo()
	reqBody := createSOAPRequest("UnknownOperation", "someclient", nil)
	req := httptest.NewRequest("POST", "/soap", strings.NewReader(reqBody))
	rec := httptest.NewRecorder()

	soapHandler(repo, rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var response kycModels.KYCResult
	err := unmarshalSOAPResponse(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Error", response.Status)
	assert.Contains(t, response.Message, "Unknown SOAP operation: UnknownOperation")
}
