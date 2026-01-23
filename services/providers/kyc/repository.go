package main

import (
	"fmt"
	"sync"

	"kafka-soap-e2e-test/services/consumer/models" // Import UserData from consumer's models
)

// Action type for simulating different responses
type Action string

const (
	ActionTimeout       Action = "Timeout"
	ActionInternalError Action = "InternalError"
	ActionNotFound      Action = "NotFound"
)

// InMemoryRepo stores UserData in memory and simulated actions
type InMemoryRepo struct {
	mu      sync.RWMutex
	users   map[string]models.UserData
	actions map[string]Action // New field to store simulated actions per ClientID
}

// NewInMemoryRepo initializes a new InMemoryRepo
func NewInMemoryRepo() *InMemoryRepo {
	repo := &InMemoryRepo{
		users:   make(map[string]models.UserData),
		actions: make(map[string]Action), // Initialize the actions map
	}
	return repo
}

// Create adds new UserData to the repository
func (r *InMemoryRepo) Create(userData models.UserData) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.users[userData.ClientID]; exists {
		return fmt.Errorf("user with ClientID '%s' already exists", userData.ClientID)
	}
	r.users[userData.ClientID] = userData
	return nil
}

// Read retrieves UserData by ClientID
func (r *InMemoryRepo) Read(clientID string) (models.UserData, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if user, exists := r.users[clientID]; exists {
		return user, nil
	}
	return models.UserData{}, fmt.Errorf("user with ClientID '%s' not found", clientID)
}

// Update modifies existing UserData in the repository
func (r *InMemoryRepo) Update(userData models.UserData) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.users[userData.ClientID]; !exists {
		return fmt.Errorf("user with ClientID '%s' not found", userData.ClientID)
	}
	r.users[userData.ClientID] = userData
	return nil
}

// Delete removes UserData by ClientID
func (r *InMemoryRepo) Delete(clientID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.users[clientID]; !exists {
		return fmt.Errorf("user with ClientID '%s' not found", clientID)
	}
	delete(r.users, clientID)
	return nil
}

// SetAction sets a simulated action for a given ClientID
func (r *InMemoryRepo) SetAction(clientID string, action Action) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.actions[clientID] = action
}

// GetAction retrieves the simulated action for a given ClientID
func (r *InMemoryRepo) GetAction(clientID string) (Action, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	action, exists := r.actions[clientID]
	return action, exists
}

// ClearAction removes the simulated action for a given ClientID
func (r *InMemoryRepo) ClearAction(clientID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.actions, clientID)
}
