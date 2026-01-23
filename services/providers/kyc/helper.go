package main

import (
	"encoding/json"

	"log"
	"os"
	"path/filepath"
	"strings"

	"kafka-soap-e2e-test/services/consumer/models" // Import UserData from consumer's models
)

// loadUserDataFromFiles reads JSON files from a directory and populates the repository
func loadUserDataFromFiles(repo *InMemoryRepo, folderPath string) {
	log.Printf("KYC SOAP Server: Attempting to load UserData from %s", folderPath)
	files, err := os.ReadDir(folderPath)
	if err != nil {
		log.Printf("KYC SOAP Server: Could not read directory %s: %v. Starting with empty repository.", folderPath, err)
		return
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue // Skip directories and non-json files
		}

		filePath := filepath.Join(folderPath, file.Name())
		data, err := os.ReadFile(filePath) // Use os.ReadFile for Go 1.16+
		if err != nil {
			log.Printf("KYC SOAP Server: Failed to read file %s: %v", filePath, err)
			continue
		}

		var userData models.UserData
		if err := json.Unmarshal(data, &userData); err != nil {
			log.Printf("KYC SOAP Server: Failed to unmarshal JSON from file %s: %v", filePath, err)
			continue
		}

		if err := repo.Create(userData); err != nil {
			log.Printf("KYC SOAP Server: Failed to add UserData for ClientID '%s' from file %s to repository: %v", userData.ClientID, filePath, err)
		} else {
			log.Printf("KYC SOAP Server: Loaded UserData for ClientID '%s' from %s", userData.ClientID, filePath)
		}
	}
	log.Printf("KYC SOAP Server: Finished loading UserData from files.")
}
