package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"kafka-soap-e2e-test/services/consumer/models" // For models.UserData
)

// writeJSONResponse is a helper function to send JSON responses
func writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			log.Printf("Failed to write JSON response: %v", err)
			http.Error(w, "Failed to write JSON response", http.StatusInternalServerError)
		}
	}
}

// adminListUsers handles GET /admin/v1/users
func adminListUsers(repo *InMemoryRepo, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	repo.mu.RLock()
	defer repo.mu.RUnlock()
	users := make([]models.UserData, 0, len(repo.users))
	for _, user := range repo.users {
		users = append(users, user)
	}
	writeJSONResponse(w, http.StatusOK, users)
}

// adminGetUser handles GET /admin/v1/users/{clientID}
func adminGetUser(repo *InMemoryRepo, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	clientID := strings.TrimPrefix(r.URL.Path, "/admin/v1/users/")
	if clientID == "" {
		http.Error(w, "Client ID is required", http.StatusBadRequest)
		return
	}

	user, err := repo.Read(clientID)
	if err != nil {
		writeJSONResponse(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	writeJSONResponse(w, http.StatusOK, user)
}

// adminCreateUser handles POST /admin/v1/users
func adminCreateUser(repo *InMemoryRepo, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var userData models.UserData
	if err := json.NewDecoder(r.Body).Decode(&userData); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	if err := repo.Create(userData); err != nil {
		writeJSONResponse(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	writeJSONResponse(w, http.StatusCreated, userData)
}

// adminUpdateUser handles PUT /admin/v1/users/{clientID}
func adminUpdateUser(repo *InMemoryRepo, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	clientID := strings.TrimPrefix(r.URL.Path, "/admin/v1/users/")
	if clientID == "" {
		http.Error(w, "Client ID is required", http.StatusBadRequest)
		return
	}

	var userData models.UserData
	if err := json.NewDecoder(r.Body).Decode(&userData); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}
	if userData.ClientID != "" && userData.ClientID != clientID {
		writeJSONResponse(w, http.StatusBadRequest, map[string]string{"error": "Client ID in path and body do not match"})
		return
	}
	userData.ClientID = clientID // Ensure clientID from path is used

	if err := repo.Update(userData); err != nil {
		writeJSONResponse(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	writeJSONResponse(w, http.StatusOK, userData)
}

// adminDeleteUser handles DELETE /admin/v1/users/{clientID}
func adminDeleteUser(repo *InMemoryRepo, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	clientID := strings.TrimPrefix(r.URL.Path, "/admin/v1/users/")
	if clientID == "" {
		http.Error(w, "Client ID is required", http.StatusBadRequest)
		return
	}

	if err := repo.Delete(clientID); err != nil {
		writeJSONResponse(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	writeJSONResponse(w, http.StatusOK, map[string]string{"message": fmt.Sprintf("User '%s' deleted", clientID)})
}

// adminSetAction handles POST /admin/v1/actions/{clientID}
func adminSetAction(repo *InMemoryRepo, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	clientID := strings.TrimPrefix(r.URL.Path, "/admin/v1/actions/")
	if clientID == "" {
		http.Error(w, "Client ID is required", http.StatusBadRequest)
		return
	}

	var requestBody struct {
		Action string `json:"action"`
	}
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	action := Action(requestBody.Action)
	switch action {
	case ActionTimeout, ActionInternalError, ActionNotFound:
		repo.SetAction(clientID, action)
		writeJSONResponse(w, http.StatusOK, map[string]string{"message": fmt.Sprintf("Action '%s' set for ClientID '%s'", action, clientID)})
	default:
		writeJSONResponse(w, http.StatusBadRequest, map[string]string{"error": "Invalid action type"})
	}
}

// adminClearAction handles DELETE /admin/v1/actions/{clientID}
func adminClearAction(repo *InMemoryRepo, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	clientID := strings.TrimPrefix(r.URL.Path, "/admin/v1/actions/")
	if clientID == "" {
		http.Error(w, "Client ID is required", http.StatusBadRequest)
		return
	}

	repo.ClearAction(clientID)
	writeJSONResponse(w, http.StatusOK, map[string]string{"message": fmt.Sprintf("Action cleared for ClientID '%s'", clientID)})
}

// initAdminRoutes registers all admin API routes to the given ServeMux.
func initAdminRoutes(mux *http.ServeMux, repo *InMemoryRepo) {
	// Register Admin API handlers
	mux.HandleFunc("/admin/v1/users/", func(w http.ResponseWriter, r *http.Request) { // Trailing slash to match {clientID}
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/admin/v1/users/"), "/")
		if len(parts) == 1 && parts[0] != "" { // Matches /admin/v1/users/{clientID}
			switch r.Method {
			case http.MethodGet:
				adminGetUser(repo, w, r)
			case http.MethodPut:
				adminUpdateUser(repo, w, r)
			case http.MethodDelete:
				adminDeleteUser(repo, w, r)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		} else { // This else branch is now specifically for the case where it's /admin/v1/users/ with no clientID (which is valid for POST to list users)
			http.NotFound(w, r) // If there's nothing after the trailing slash, it's not a valid /users/{id} request
		}
	})
	mux.HandleFunc("/admin/v1/actions/", func(w http.ResponseWriter, r *http.Request) { // Trailing slash to match {clientID}
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/admin/v1/actions/"), "/")
		if len(parts) == 1 && parts[0] != "" { // Matches /admin/v1/actions/{clientID}
			switch r.Method {
			case http.MethodPost:
				adminSetAction(repo, w, r)
			case http.MethodDelete:
				adminClearAction(repo, w, r)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		} else {
			http.NotFound(w, r)
		}
	})

	// Handle /admin/v1/users (without trailing slash) for LIST (GET) and CREATE (POST)
	mux.HandleFunc("/admin/v1/users", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			adminListUsers(repo, w, r)
		case http.MethodPost:
			adminCreateUser(repo, w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
}
