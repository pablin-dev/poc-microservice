package framework

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const (
	defaultHTTPTimeout = 30 * time.Second
)

// Client is a Mountebank API client.
type MountebankClient struct {
	MountebankURL   string
	AdminPort       int
	ImpostersPort   []map[string]int
	HTTPClient      *http.Client
	StoredImposters map[int]*DetailedImposter // Exported field
}

// NewClient creates and returns a new MountebankClient.
func NewMountebankClient(cfg MountebankConfig) *MountebankClient {
	log.Printf("Initializing MountebankClient with URL: %s, Admin Port: %d, Imposters Ports: %v", cfg.URL, cfg.Ports.Admin, cfg.Ports.Imposters)
	return &MountebankClient{
		MountebankURL:   cfg.URL,
		AdminPort:       cfg.Ports.Admin,
		ImpostersPort:   cfg.Ports.Imposters,
		HTTPClient:      &http.Client{Timeout: defaultHTTPTimeout},
		StoredImposters: make(map[int]*DetailedImposter), // Initialize the exported field
	}
}

// GetAdminBaseURL returns the base URL for the Mountebank admin API.
func (c *MountebankClient) GetAdminBaseURL() string {
	return fmt.Sprintf("%s:%d", c.MountebankURL, c.AdminPort)
}

// Init initializes the Mountebank client by waiting for Mountebank to be ready
// and then storing all currently active impostors.
func (c *MountebankClient) Init(timeout time.Duration) error {
	if err := c.WaitForMountebank(timeout); err != nil {
		return fmt.Errorf("failed to wait for Mountebank: %w", err)
	}

	imposters, err := c.StoreAllImpostors()
	if err != nil {
		return fmt.Errorf("failed to store all impostors: %w", err)
	}
	c.StoredImposters = imposters // Assign to the exported field
	return nil
}

// WaitForMountebank polls Mountebank until it's ready to serve requests or a timeout occurs.
func (c *MountebankClient) WaitForMountebank(timeout time.Duration) error {
	adminBaseURL := c.GetAdminBaseURL()
	log.Printf("Waiting for Mountebank to be ready at %s for %s", adminBaseURL, timeout)
	endTime := time.Now().Add(timeout)
	for time.Now().Before(endTime) {
		// Check if /imposters endpoint is responsive
		_, errImposters := c.GetAllImpostors()
		if errImposters == nil {
			log.Println("Mountebank is ready (/imposters endpoint is responsive).")
			return nil
		}
		log.Printf("Mountebank not yet ready. /imposters endpoint not responsive: %v. Retrying in 1 second...", errImposters)
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("timed out waiting for Mountebank to be ready")
}

// GetAllImpostors retrieves all active impostors from Mountebank.
func (c *MountebankClient) GetAllImpostors() (*ImpostersResponse, error) {
	adminBaseURL := c.GetAdminBaseURL()
	url := fmt.Sprintf("%s/imposters", adminBaseURL)
	log.Printf("DEBUG: Attempting GET request to URL: %s", url)
	resp, err := c.HTTPClient.Get(url)
	if err != nil {
		log.Printf("ERROR: Failed to make GET request to %s: %v", url, err)
		return nil, fmt.Errorf("failed to make GET request to %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			log.Printf("ERROR: Failed GET %s: unexpected status code %d, failed to read response body: %v", url, resp.StatusCode, readErr)
			return nil, fmt.Errorf("unexpected status code for GET %s: %d, failed to read response body: %w", url, resp.StatusCode, readErr)
		}
		log.Printf("ERROR: Failed GET %s: unexpected status code %d, body: %s", url, resp.StatusCode, string(body))
		return nil, fmt.Errorf("unexpected status code for GET %s: %d, body: %s", url, resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var impostersResponse ImpostersResponse
	if err := json.Unmarshal(body, &impostersResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal imposters response: %w", err)
	}

	return &impostersResponse, nil
}

// DeleteAllImpostors deletes all active imposters from Mountebank.
// func (c *MountebankClient) DeleteAllImpostors() error {
// 	url := fmt.Sprintf("%s/imposters", c.BaseURL)
// 	log.Printf("DEBUG: Attempting DELETE request to URL: %s", url)
// 	req, err := http.NewRequest(http.MethodDelete, url, nil)
// 	if err != nil {
// 		log.Printf("ERROR: Failed to create DELETE request to %s: %v", url, err)
// 		return fmt.Errorf("failed to create DELETE request to %s: %w", url, err)
// 	}
// 	req.Header.Add("Accept", "*/*")
// 	req.Header.Add("User-Agent", "Go-http-client/1.1")

// 	resp, err := c.HTTPClient.Do(req)
// 	if err != nil {
// 		log.Printf("ERROR: Failed to make DELETE request to %s: %v", url, err)
// 		return fmt.Errorf("failed to make DELETE request to %s: %w", url, err)
// 	}
// 	defer resp.Body.Close()

// 	if resp.StatusCode != http.StatusOK {
// 		body, readErr := io.ReadAll(resp.Body)
// 		if readErr != nil {
// 			log.Printf("ERROR: Failed DELETE %s: unexpected status code %d, failed to read response body: %v", url, resp.StatusCode, readErr)
// 			return fmt.Errorf("unexpected status code for DELETE %s: %d, failed to read response body: %w", url, resp.StatusCode, readErr)
// 		}
// 		log.Printf("ERROR: Failed DELETE %s: unexpected status code %d, body: %s", url, resp.StatusCode, string(body))
// 		return fmt.Errorf("unexpected status code for DELETE %s: %d, body: %s", url, resp.StatusCode, string(body))
// 	}

// 	return nil
// }

// GetImposter retrieves a single imposter by its port from Mountebank.
func (c *MountebankClient) GetImposter(port int) (*DetailedImposter, error) {
	adminBaseURL := c.GetAdminBaseURL()
	url := fmt.Sprintf("%s/imposters/%d", adminBaseURL, port)
	log.Printf("DEBUG: Attempting GET request to URL: %s", url)
	resp, err := c.HTTPClient.Get(url)
	if err != nil {
		log.Printf("ERROR: Failed to make GET request to %s: %v", url, err)
		return nil, fmt.Errorf("failed to make GET request to %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			log.Printf("ERROR: Failed GET %s: unexpected status code %d, failed to read response body: %v", url, resp.StatusCode, readErr)
			return nil, fmt.Errorf("unexpected status code for GET %s: %d, failed to read response body: %w", url, resp.StatusCode, readErr)
		}
		log.Printf("ERROR: Failed GET %s: unexpected status code %d, body: %s", url, resp.StatusCode, string(body))
		return nil, fmt.Errorf("unexpected status code for GET %s: %d, body: %s", url, resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var imposter DetailedImposter
	if err := json.Unmarshal(body, &imposter); err != nil {
		return nil, fmt.Errorf("failed to unmarshal imposter response for port %d: %w", port, err)
	}

	return &imposter, nil
}

// DeleteImpostor deletes a specific imposter from Mountebank by its port.
func (c *MountebankClient) DeleteImpostor(port int) error {
	adminBaseURL := c.GetAdminBaseURL()
	url := fmt.Sprintf("%s/imposters/%d", adminBaseURL, port)
	log.Printf("DEBUG: Attempting DELETE request to URL: %s", url)
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		log.Printf("ERROR: Failed to create DELETE request to %s: %v", url, err)
		return fmt.Errorf("failed to create DELETE request to %s: %w", url, err)
	}
	req.Header.Add("Accept", "*/*")
	req.Header.Add("User-Agent", "Go-http-client/1.1")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		log.Printf("ERROR: Failed to make DELETE request to %s: %v", url, err)
		return fmt.Errorf("failed to make DELETE request to %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			log.Printf("ERROR: Failed DELETE %s: unexpected status code %d, failed to read response body: %v", url, resp.StatusCode, readErr)
			return fmt.Errorf("unexpected status code for DELETE %s: %d, failed to read response body: %w", url, resp.StatusCode, readErr)
		}
		log.Printf("ERROR: Failed DELETE %s: unexpected status code %d, body: %s", url, resp.StatusCode, string(body))
		return fmt.Errorf("unexpected status code for DELETE %s: %d, body: %s", url, resp.StatusCode, string(body))
	}

	return nil
}

// StoreAllImpostors retrieves all active impostors and stores their detailed configurations in a map.
func (c *MountebankClient) StoreAllImpostors() (map[int]*DetailedImposter, error) {
	storedImposters := make(map[int]*DetailedImposter)

	impostersResponse, err := c.GetAllImpostors()
	if err != nil {
		return nil, fmt.Errorf("failed to get all impostors: %w", err)
	}

	for _, imposter := range impostersResponse.Imposters {
		detailedImposter, err := c.GetImposter(imposter.Port)
		if err != nil {
			return nil, fmt.Errorf("failed to get detailed imposter for port %d: %w", imposter.Port, err)
		}
		storedImposters[imposter.Port] = detailedImposter
	}

	return storedImposters, nil
}

// CreateImposter creates a new imposter in Mountebank.
func (c *MountebankClient) CreateImposter(imposter *DetailedImposter) error {
	adminBaseURL := c.GetAdminBaseURL()
	url := fmt.Sprintf("%s/imposters", adminBaseURL)
	log.Printf("DEBUG: Attempting POST request to URL: %s for imposter on port %d", url, imposter.Port)

	imposterJSON, err := json.Marshal(imposter)
	if err != nil {
		return fmt.Errorf("failed to marshal imposter to JSON: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(imposterJSON))
	if err != nil {
		return fmt.Errorf("failed to create POST request to %s: %w", url, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("Accept", "*/*")
	req.Header.Add("User-Agent", "Go-http-client/1.1")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make POST request to %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated { // Mountebank returns 201 Created for successful impostor creation
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return fmt.Errorf("unexpected status code for POST %s: %d, failed to read response body: %w", url, resp.StatusCode, readErr)
		}
		return fmt.Errorf("unexpected status code for POST %s: %d, body: %s", url, resp.StatusCode, string(body))
	}

	return nil
}

// RestoreImposter restores a previously stored imposter to Mountebank.
func (c *MountebankClient) RestoreImposter(port int) error {
	imposter, found := c.StoredImposters[port]
	if !found {
		return fmt.Errorf("imposter with port %d not found in stored imposters map", port)
	}
	log.Printf("DEBUG: Restoring imposter %s on port %d", imposter.Name, imposter.Port)
	return c.CreateImposter(imposter)
}

// DeleteRequests deletes all recorded requests for a specific imposter.
func (c *MountebankClient) DeleteRequests(port int) error {
	adminBaseURL := c.GetAdminBaseURL()
	url := fmt.Sprintf("%s/imposters/%d/requests", adminBaseURL, port)
	log.Printf("DEBUG: Attempting DELETE request to URL: %s", url)
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		log.Printf("ERROR: Failed to create DELETE request to %s: %v", url, err)
		return fmt.Errorf("failed to create DELETE request to %s: %w", url, err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		log.Printf("ERROR: Failed to make DELETE request to %s: %v", url, err)
		return fmt.Errorf("failed to make DELETE request to %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			log.Printf("ERROR: Failed DELETE %s: unexpected status code %d, failed to read response body: %v", url, resp.StatusCode, readErr)
			return fmt.Errorf("unexpected status code for DELETE %s: %d, failed to read response body: %w", url, resp.StatusCode, readErr)
		}
		log.Printf("ERROR: Failed DELETE %s: unexpected status code %d, body: %s", url, resp.StatusCode, string(body))
		return fmt.Errorf("unexpected status code for DELETE %s: %d, body: %s", url, resp.StatusCode, string(body))
	}

	return nil
}

// DeleteResponses deletes all recorded proxy responses for a specific imposter.
func (c *MountebankClient) DeleteResponses(port int) error {
	adminBaseURL := c.GetAdminBaseURL()
	url := fmt.Sprintf("%s/imposters/%d/proxy/responses", adminBaseURL, port)
	log.Printf("DEBUG: Attempting DELETE request to URL: %s", url)
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		log.Printf("ERROR: Failed to create DELETE request to %s: %v", url, err)
		return fmt.Errorf("failed to create DELETE request to %s: %w", url, err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		log.Printf("ERROR: Failed to make DELETE request to %s: %v", url, err)
		return fmt.Errorf("failed to make DELETE request to %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			log.Printf("ERROR: Failed DELETE %s: unexpected status code %d, failed to read response body: %v", url, resp.StatusCode, readErr)
			return fmt.Errorf("unexpected status code for DELETE %s: %d, failed to read response body: %w", url, resp.StatusCode, readErr)
		}
		log.Printf("ERROR: Failed DELETE %s: unexpected status code %d, body: %s", url, resp.StatusCode, string(body))
		return fmt.Errorf("unexpected status code for DELETE %s: %d, body: %s", url, resp.StatusCode, string(body))
	}

	return nil
}
