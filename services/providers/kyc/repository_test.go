package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"kafka-soap-e2e-test/services/consumer/models"
)

func TestNewInMemoryRepo(t *testing.T) {
	repo := NewInMemoryRepo()
	assert.NotNil(t, repo)
	assert.Empty(t, repo.users)
}

func TestRepoCreate(t *testing.T) {
	repo := NewInMemoryRepo()
	user := models.UserData{ClientID: "client1", Risk: 0.5, Status: "Approved", Message: "Test message"}

	err := repo.Create(user)
	assert.NoError(t, err)
	assert.Contains(t, repo.users, "client1")
	assert.Equal(t, user, repo.users["client1"])

	err = repo.Create(user) // Attempt to create again
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestRepoRead(t *testing.T) {
	repo := NewInMemoryRepo()
	user := models.UserData{ClientID: "client1", Risk: 0.5, Status: "Approved", Message: "Test message"}
	assert.NoError(t, repo.Create(user))

	// Read existing user
	foundUser, err := repo.Read("client1")
	assert.NoError(t, err)
	assert.Equal(t, user, foundUser)

	// Read non-existing user
	_, err = repo.Read("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRepoUpdate(t *testing.T) {
	repo := NewInMemoryRepo()
	user := models.UserData{ClientID: "client1", Risk: 0.5, Status: "Approved", Message: "Test message"}
	assert.NoError(t, repo.Create(user))

	updatedUser := models.UserData{ClientID: "client1", Risk: 0.8, Status: "Pending", Message: "Updated message"}
	err := repo.Update(updatedUser)
	assert.NoError(t, err)
	assert.Equal(t, updatedUser, repo.users["client1"])

	// Update non-existing user
	nonExistentUser := models.UserData{ClientID: "nonexistent", Risk: 0.1}
	err = repo.Update(nonExistentUser)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRepoDelete(t *testing.T) {
	repo := NewInMemoryRepo()
	user := models.UserData{ClientID: "client1", Risk: 0.5, Status: "Approved", Message: "Test message"}
	assert.NoError(t, repo.Create(user))

	// Delete existing user
	err := repo.Delete("client1")
	assert.NoError(t, err)
	assert.NotContains(t, repo.users, "client1")

	// Delete non-existing user
	err = repo.Delete("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
