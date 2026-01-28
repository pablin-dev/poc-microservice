package soapclient

import (
	"io"
	"net/http"
	"net/http/httptest"

	"testing"
	"time"

	"kafka-soap-e2e-test/services/consumer/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock SOAP responses for testing
const (
	mockReadKYCResponseSuccess = `<?xml version="1.0" encoding="UTF-8"?>
<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/">
  <soapenv:Body>
    <KYCResponse xmlns="http://example.com/kyc">
      <Status>Success</Status>
      <Message>User data retrieved</Message>
      <UserData>
        <ClientID>client123</ClientID>
        <Risk>0.5</Risk>
      </UserData>
    </KYCResponse>
  </soapenv:Body>
</soapenv:Envelope>`

	mockReadKYCResponseNoUserData = `<?xml version="1.0" encoding="UTF-8"?>
<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/">
  <soapenv:Body>
    <KYCResponse xmlns="http://example.com/kyc">
      <Status>Success</Status>
      <Message>No User data found</Message>
    </KYCResponse>
  </soapenv:Body>
</soapenv:Envelope>`

	mockReadKYCResponseError = `<?xml version="1.0" encoding="UTF-8"?>
<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/">
  <soapenv:Body>
    <KYCResponse xmlns="http://example.com/kyc">
      <Status>Error</Status>
      <Message>User not found</Message>
    </KYCResponse>
  </soapenv:Body>
</soapenv:Envelope>`

	mockCreateKYCResponseSuccess = `<?xml version="1.0" encoding="UTF-8"?>
<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/">
  <soapenv:Body>
    <KYCResponse xmlns="http://example.com/kyc">
      <Status>Success</Status>
      <Message>User created</Message>
      <UserData>
        <ClientID>newclient</ClientID>
        <Risk>0.1</Risk>
      </UserData>
    </KYCResponse>
  </soapenv:Body>
</soapenv:Envelope>`

	mockUpdateKYCResponseSuccess = `<?xml version="1.0" encoding="UTF-8"?>
<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/">
  <soapenv:Body>
    <KYCResponse xmlns="http://example.com/kyc">
      <Status>Success</Status>
      <Message>User updated</Message>
      <UserData>
        <ClientID>client123</ClientID>
        <Risk>0.8</Risk>
      </UserData>
    </KYCResponse>
  </soapenv:Body>
</soapenv:Envelope>`

	mockDeleteKYCResponseSuccess = `<?xml version="1.0" encoding="UTF-8"?>
<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/">
  <soapenv:Body>
    <DeleteKYCResponse xmlns="http://example.com/kyc">
      <Status>Success</Status>
      <Message>User deleted successfully</Message>
    </DeleteKYCResponse>
  </soapenv:Body>
</soapenv:Envelope>`
)

func TestNewSOAPClient(t *testing.T) {
	client := NewSOAPClient("http://localhost:8081/soap")
	assert.NotNil(t, client)
	assert.Equal(t, "http://localhost:8081/soap", client.url)
	assert.NotNil(t, client.client)
	assert.Equal(t, 10*time.Second, client.client.Timeout)
}

func TestDoSOAPRequest(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "text/xml; charset=utf-8", r.Header.Get("Content-Type"))
			assert.Equal(t, "http://example.com/kyc/KYCQuery", r.Header.Get("SOAPAction"))
			_, _ = io.WriteString(w, mockReadKYCResponseSuccess)
		}))
		defer ts.Close()

		sc := NewSOAPClient(ts.URL)
		respBody, err := sc.doSOAPRequest("http://example.com/kyc/KYCQuery", []byte("<request/>"))

		require.NoError(t, err)
		assert.Equal(t, mockReadKYCResponseSuccess, string(respBody))
	})

	t.Run("HTTP Error", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = io.WriteString(w, "Internal Server Error")
		}))
		defer ts.Close()

		sc := NewSOAPClient(ts.URL)
		_, err := sc.doSOAPRequest("http://example.com/kyc/KYCQuery", []byte("<request/>"))

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "SOAP service returned non-OK status: 500")
	})

	t.Run("Request Creation Error", func(t *testing.T) {
		sc := NewSOAPClient("://invalid-url") // Invalid URL to trigger http.NewRequest error
		_, err := sc.doSOAPRequest("http://example.com/kyc/KYCQuery", []byte("<request/>"))

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create SOAP request")
	})
}

func TestReadKYC(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.WriteString(w, mockReadKYCResponseSuccess)
		}))
		defer ts.Close()

		sc := NewSOAPClient(ts.URL)
		userData, err := sc.ReadKYC("client123")

		require.NoError(t, err)
		assert.Equal(t, "client123", userData.ClientID)
		assert.InDelta(t, 0.5, userData.Risk, 0.001)
		assert.Equal(t, "Success", userData.Status)
		assert.Equal(t, "User data retrieved", userData.Message)
	})

	t.Run("No UserData in Response", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.WriteString(w, mockReadKYCResponseNoUserData)
		}))
		defer ts.Close()

		sc := NewSOAPClient(ts.URL)
		userData, err := sc.ReadKYC("client123")
		require.NoError(t, err)
		assert.Equal(t, "", userData.ClientID)
		assert.Equal(t, 0.0, userData.Risk)
		assert.Equal(t, "Success", userData.Status)
		assert.Equal(t, "No User data found", userData.Message)
	})

	t.Run("Service Error Response", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.WriteString(w, mockReadKYCResponseError)
		}))
		defer ts.Close()

		sc := NewSOAPClient(ts.URL)
		userData, err := sc.ReadKYC("nonexistent") // Call should succeed, error is in response body

		require.NoError(t, err)                           // No HTTP error, but SOAP response indicates error
		assert.Equal(t, "nonexistent", userData.ClientID) // ClientID should be populated from the request in error response
		assert.Equal(t, 0.0, userData.Risk)
		assert.Equal(t, "Error", userData.Status)
		assert.Equal(t, "User not found", userData.Message)
	})

	t.Run("HTTP Error During Read", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
		}))
		defer ts.Close()

		sc := NewSOAPClient(ts.URL)
		_, err := sc.ReadKYC("client123")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "SOAP service returned non-OK status: 400")
	})

	t.Run("Unmarshal Error", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.WriteString(w, "<invalid-xml>")
		}))
		defer ts.Close()

		sc := NewSOAPClient(ts.URL)
		_, err := sc.ReadKYC("client123")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse SOAP response")
	})
}

func TestCreateKYC(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.WriteString(w, mockCreateKYCResponseSuccess)
		}))
		defer ts.Close()

		sc := NewSOAPClient(ts.URL)
		newUserData := models.UserData{ClientID: "newclient", Risk: 0.1}
		userData, err := sc.CreateKYC(newUserData)

		require.NoError(t, err)
		assert.Equal(t, "newclient", userData.ClientID)
		assert.InDelta(t, 0.1, userData.Risk, 0.001)
		assert.Equal(t, "Success", userData.Status)
		assert.Equal(t, "User created", userData.Message)
	})

	t.Run("HTTP Error During Create", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer ts.Close()

		sc := NewSOAPClient(ts.URL)
		newUserData := models.UserData{ClientID: "newclient", Risk: 0.1}
		_, err := sc.CreateKYC(newUserData)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "SOAP service returned non-OK status: 500")
	})

	t.Run("Unmarshal Error", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.WriteString(w, "<invalid-xml>")
		}))
		defer ts.Close()

		sc := NewSOAPClient(ts.URL)
		newUserData := models.UserData{ClientID: "newclient", Risk: 0.1}
		_, err := sc.CreateKYC(newUserData)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse SOAP response")
	})
}

func TestUpdateKYC(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.WriteString(w, mockUpdateKYCResponseSuccess)
		}))
		defer ts.Close()

		sc := NewSOAPClient(ts.URL)
		updateUserData := models.UserData{ClientID: "client123", Risk: 0.8}
		userData, err := sc.UpdateKYC(updateUserData)

		require.NoError(t, err)
		assert.Equal(t, "client123", userData.ClientID)
		assert.InDelta(t, 0.8, userData.Risk, 0.001)
		assert.Equal(t, "Success", userData.Status)
		assert.Equal(t, "User updated", userData.Message)
	})

	t.Run("HTTP Error During Update", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer ts.Close()

		sc := NewSOAPClient(ts.URL)
		updateUserData := models.UserData{ClientID: "client123", Risk: 0.8}
		_, err := sc.UpdateKYC(updateUserData)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "SOAP service returned non-OK status: 404")
	})

	t.Run("Unmarshal Error", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.WriteString(w, "<invalid-xml>")
		}))
		defer ts.Close()

		sc := NewSOAPClient(ts.URL)
		updateUserData := models.UserData{ClientID: "client123", Risk: 0.8}
		_, err := sc.UpdateKYC(updateUserData)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse SOAP response")
	})
}

func TestDeleteKYC(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.WriteString(w, mockDeleteKYCResponseSuccess)
		}))
		defer ts.Close()

		sc := NewSOAPClient(ts.URL)
		message, err := sc.DeleteKYC("client123")

		require.NoError(t, err)
		assert.Equal(t, "User deleted successfully", message)
	})

	t.Run("HTTP Error During Delete", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusConflict) // Example: trying to delete non-existent user
		}))
		defer ts.Close()

		sc := NewSOAPClient(ts.URL)
		_, err := sc.DeleteKYC("client123")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "SOAP service returned non-OK status: 409")
	})

	t.Run("Unmarshal Error", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.WriteString(w, "<invalid-xml>")
		}))
		defer ts.Close()

		sc := NewSOAPClient(ts.URL)
		_, err := sc.DeleteKYC("client123")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse SOAP response")
	})
}
