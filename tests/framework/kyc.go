package framework

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"kafka-soap-e2e-test/services/consumer/models"
)

// AdminAPIClient struct for interacting with the kyc-service admin API
type AdminAPIClient struct {
	client  *http.Client
	baseURL string
}

// NewAdminAPIClient initializes a new AdminAPIClient
func NewAdminAPIClient(kycServiceURL string) *AdminAPIClient {
	adminURL := strings.TrimSuffix(kycServiceURL, "/soap") + "/admin/v1"
	log.Printf("Initializing Admin API client with base URL: %s", adminURL)
	return &AdminAPIClient{
		client:  &http.Client{Timeout: 5 * time.Second}, // Admin operations should be quick
		baseURL: adminURL,
	}
}

// CreateUser via Admin API
func (a *AdminAPIClient) CreateUser(userData models.UserData) error {
	url := fmt.Sprintf("%s/users", a.baseURL)
	body, err := json.Marshal(userData)
	if err != nil {
		return fmt.Errorf("failed to marshal user data: %w", err)
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create admin create user request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call admin create user API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusConflict {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("admin create user API returned non-201/409 status: %d, body: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// ReadUser via Admin API
func (a *AdminAPIClient) ReadUser(clientID string) (models.UserData, error) {
	url := fmt.Sprintf("%s/users/%s", a.baseURL, clientID)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return models.UserData{}, fmt.Errorf("failed to create admin read user request: %w", err)
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return models.UserData{}, fmt.Errorf("failed to call admin read user API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return models.UserData{}, fmt.Errorf("admin read user API returned non-200 status: %d, body: %s", resp.StatusCode, string(respBody))
	}

	var userData models.UserData
	if err := json.NewDecoder(resp.Body).Decode(&userData); err != nil {
		return models.UserData{}, fmt.Errorf("failed to decode user data from admin read response: %w", err)
	}
	return userData, nil
}

// UpdateUser via Admin API
func (a *AdminAPIClient) UpdateUser(userData models.UserData) error {
	url := fmt.Sprintf("%s/users/%s", a.baseURL, userData.ClientID)
	body, err := json.Marshal(userData)
	if err != nil {
		return fmt.Errorf("failed to marshal user data for update: %w", err)
	}
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create admin update user request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call admin update user API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("admin update user API returned non-200 status: %d, body: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// DeleteUser via Admin API
func (a *AdminAPIClient) DeleteUser(clientID string) error {
	url := fmt.Sprintf("%s/users/%s", a.baseURL, clientID)
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create admin delete user request: %w", err)
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call admin delete user API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("admin delete user API returned non-200/404 status: %d, body: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// SetAction via Admin API
func (a *AdminAPIClient) SetAction(clientID string, actionType string) error {
	url := fmt.Sprintf("%s/actions/%s", a.baseURL, clientID)
	body, err := json.Marshal(map[string]string{"action": actionType})
	if err != nil {
		return fmt.Errorf("failed to marshal action data: %w", err)
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create admin set action request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call admin set action API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("admin set action API returned non-200 status: %d, body: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// ClearAction via Admin API
func (a *AdminAPIClient) ClearAction(clientID string) error {
	url := fmt.Sprintf("%s/actions/%s", a.baseURL, clientID)
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create admin clear action request: %w", err)
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call admin clear action API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("admin clear action API returned non-200 status: %d, body: %s", resp.StatusCode, string(respBody))
	}
	return nil
}
