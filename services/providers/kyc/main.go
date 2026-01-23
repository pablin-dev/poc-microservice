package main

import (

	"encoding/xml"
	"fmt"
	"io"

	kycModels "kafka-soap-e2e-test/services/providers/kyc/models" // Import the new models package
	"log"
	"net/http"
	"os"

	"time"    // Added for time.Sleep for ActionTimeout
)

const (
	soapEnvelopeStart = `<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/">`
	soapEnvelopeEnd   = `</soapenv:Envelope>`
	soapBodyStart     = `<soapenv:Body>`
	soapBodyEnd       = `</soapenv:Body>`
	kycNamespace      = `xmlns="http://example.com/kyc"`
	kycNamespaceAttr  = `http://example.com/kyc`
)

// --- SOAP Request/Response Models for this Service ---

// --- SOAP Server Handler ---

// soapHandler handles all incoming SOAP requests for CRUD operations
func soapHandler(repo *InMemoryRepo, w http.ResponseWriter, r *http.Request) {
	log.Printf("KYC SOAP Server: Received request for %s %s", r.Method, r.URL.Path)

	// Assume POST for all SOAP operations, as is standard for SOAP
	if r.Method != "POST" {
		http.Error(w, "SOAP requests must use POST method", http.StatusMethodNotAllowed)
		log.Printf("KYC SOAP Server: Invalid method %s. Must be POST.", r.Method)
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read request body: %v", err), http.StatusInternalServerError)
		log.Printf("KYC SOAP Server: Failed to read request body: %v", err)
		return
	}
	log.Printf("KYC SOAP Server: Raw request body: %s", string(bodyBytes))

	var envelope kycModels.SOAPEnvelope
	if err := xml.Unmarshal(bodyBytes, &envelope); err != nil {
		log.Printf("KYC SOAP Server: Failed to unmarshal SOAP envelope: %v. Raw body: %s", err, string(bodyBytes)) // Add raw body to log
		http.Error(w, fmt.Sprintf("Failed to parse SOAP request: %v", err), http.StatusBadRequest)
		return
	}

	var responseEnvelope interface{}
	var statusCode int = http.StatusOK

	// Determine operation based on the root element in the SOAP body
	root := ""
	if len(envelope.Body.Content) > 0 {
		// Use a temporary struct to extract the root element name
		var temp struct {
			XMLName xml.Name
		}
		if err := xml.Unmarshal(envelope.Body.Content, &temp); err == nil {
			root = temp.XMLName.Local
		} else {
			log.Printf("KYC SOAP Server: Could not determine root element from SOAP body content: %v. Content: %s", err, string(envelope.Body.Content)) // Add content to log
		}
	}

	log.Printf("KYC SOAP Server: Detected SOAP operation: %s", root)

	// Handle different CRUD operations
	switch root {
	case "KYCQuery": // Read operation
		var req kycModels.KYCQuery
		if err := xml.Unmarshal(envelope.Body.Content, &req); err != nil {
			log.Printf("KYC SOAP Server: Failed to unmarshal KYCQuery request: %v", err)
			statusCode = http.StatusBadRequest
			responseEnvelope = kycModels.KYCResponseEnvelope{
				XmlnsSoapenv: "http://schemas.xmlsoap.org/soap/envelope/",
				Body: kycModels.KYCResponseBody{
					KYCResult: kycModels.KYCResult{XmlnsKyc: kycNamespaceAttr, Status: "Error", Message: fmt.Sprintf("Invalid KYCQuery request: %v", err)},
				},
			}
		} else {
			// Check for action override
			if action, exists := repo.GetAction(req.ClientID); exists {
				log.Printf("KYC SOAP Server: Applying action override '%s' for ClientID '%s'", action, req.ClientID)
				switch action {
				case ActionTimeout:
					time.Sleep(5 * time.Second) // Simulate a timeout
					http.Error(w, "Timeout", http.StatusRequestTimeout)
					return
				case ActionInternalError:
					http.Error(w, "Internal Server Error (simulated)", http.StatusInternalServerError)
					return
				case ActionNotFound:
					statusCode = http.StatusNotFound
					responseEnvelope = kycModels.KYCResponseEnvelope{
						XmlnsSoapenv: "http://schemas.xmlsoap.org/soap/envelope/",
						Body: kycModels.KYCResponseBody{
							KYCResult: kycModels.KYCResult{XmlnsKyc: kycNamespaceAttr, Status: "Error", Message: fmt.Sprintf("User with ClientID '%s' not found (simulated)", req.ClientID)},
						},
					}
				}
			}

			if responseEnvelope == nil { // Only proceed if no action override has generated a response yet
				userData, err := repo.Read(req.ClientID)
				if err != nil {
					log.Printf("KYC SOAP Server: Error reading user %s: %v", req.ClientID, err)
					statusCode = http.StatusNotFound
					responseEnvelope = kycModels.KYCResponseEnvelope{
						XmlnsSoapenv: "http://schemas.xmlsoap.org/soap/envelope/",
						Body: kycModels.KYCResponseBody{
							KYCResult: kycModels.KYCResult{XmlnsKyc: kycNamespaceAttr, Status: "Error", Message: err.Error()},
						},
					}
				} else {
					responseEnvelope = kycModels.KYCResponseEnvelope{
						XmlnsSoapenv: "http://schemas.xmlsoap.org/soap/envelope/",
						Body: kycModels.KYCResponseBody{
							KYCResult: kycModels.KYCResult{XmlnsKyc: kycNamespaceAttr, Status: "Success", Message: "User data retrieved", UserData: &userData},
						},
					}
				}
			}
		}

	case "CreateKYC":
		var req kycModels.CreateKYCRequest
		if err := xml.Unmarshal(envelope.Body.Content, &req); err != nil {
			log.Printf("KYC SOAP Server: Failed to unmarshal CreateKYC request: %v", err)
			statusCode = http.StatusBadRequest
			responseEnvelope = kycModels.KYCResponseEnvelope{
				XmlnsSoapenv: "http://schemas.xmlsoap.org/soap/envelope/",
				Body: kycModels.KYCResponseBody{
					KYCResult: kycModels.KYCResult{XmlnsKyc: kycNamespaceAttr, Status: "Error", Message: fmt.Sprintf("Invalid CreateKYC request: %v", err)},
				},
			}
		} else {
			if err := repo.Create(req.UserData); err != nil {
				log.Printf("KYC SOAP Server: Error creating user %s: %v", req.UserData.ClientID, err)
				statusCode = http.StatusConflict // 409 for resource conflict
				responseEnvelope = kycModels.KYCResponseEnvelope{
					XmlnsSoapenv: "http://schemas.xmlsoap.org/soap/envelope/",
					Body: kycModels.KYCResponseBody{
						KYCResult: kycModels.KYCResult{XmlnsKyc: kycNamespaceAttr, Status: "Error", Message: err.Error()},
					},
				}
			} else {
				responseEnvelope = kycModels.KYCResponseEnvelope{
					XmlnsSoapenv: "http://schemas.xmlsoap.org/soap/envelope/",
					Body: kycModels.KYCResponseBody{
						KYCResult: kycModels.KYCResult{XmlnsKyc: kycNamespaceAttr, Status: "Success", Message: "User created", UserData: &req.UserData},
					},
				}
			}
		}

	case "UpdateKYC":
		var req kycModels.UpdateKYCRequest
		if err := xml.Unmarshal(envelope.Body.Content, &req); err != nil {
			log.Printf("KYC SOAP Server: Failed to unmarshal UpdateKYC request: %v", err)
			statusCode = http.StatusBadRequest
			responseEnvelope = kycModels.KYCResponseEnvelope{
				XmlnsSoapenv: "http://schemas.xmlsoap.org/soap/envelope/",
				Body: kycModels.KYCResponseBody{
					KYCResult: kycModels.KYCResult{XmlnsKyc: kycNamespaceAttr, Status: "Error", Message: fmt.Sprintf("Invalid UpdateKYC request: %v", err)},
				},
			}
		} else {
			if err := repo.Update(req.UserData); err != nil {
				log.Printf("KYC SOAP Server: Error updating user %s: %v", req.UserData.ClientID, err)
				statusCode = http.StatusNotFound
				responseEnvelope = kycModels.KYCResponseEnvelope{
					XmlnsSoapenv: "http://schemas.xmlsoap.org/soap/envelope/",
					Body: kycModels.KYCResponseBody{
						KYCResult: kycModels.KYCResult{XmlnsKyc: kycNamespaceAttr, Status: "Error", Message: err.Error()},
					},
				}
			} else {
				responseEnvelope = kycModels.KYCResponseEnvelope{
					XmlnsSoapenv: "http://schemas.xmlsoap.org/soap/envelope/",
					Body: kycModels.KYCResponseBody{
						KYCResult: kycModels.KYCResult{XmlnsKyc: kycNamespaceAttr, Status: "Success", Message: "User updated", UserData: &req.UserData},
					},
				}
			}
		}

	case "DeleteKYC":
		var req kycModels.DeleteKYCRequest
		if err := xml.Unmarshal(envelope.Body.Content, &req); err != nil {
			log.Printf("KYC SOAP Server: Failed to unmarshal DeleteKYC request: %v", err)
			statusCode = http.StatusBadRequest
			responseEnvelope = kycModels.DeleteKYCResponseEnvelope{
				XmlnsSoapenv: "http://schemas.xmlsoap.org/soap/envelope/",
				Body: kycModels.DeleteKYCResponseBody{
					DeleteKYCResult: kycModels.DeleteKYCResult{XmlnsKyc: kycNamespaceAttr, Status: "Error", Message: fmt.Sprintf("Invalid DeleteKYC request: %v", err)},
				},
			}
		} else {
			if err := repo.Delete(req.ClientID); err != nil {
				log.Printf("KYC SOAP Server: Error deleting user %s: %v", req.ClientID, err)
				statusCode = http.StatusNotFound
				responseEnvelope = kycModels.DeleteKYCResponseEnvelope{
					XmlnsSoapenv: "http://schemas.xmlsoap.org/soap/envelope/",
					Body: kycModels.DeleteKYCResponseBody{
						DeleteKYCResult: kycModels.DeleteKYCResult{XmlnsKyc: kycNamespaceAttr, Status: "Error", Message: err.Error()},
					},
				}
			} else {
				responseEnvelope = kycModels.DeleteKYCResponseEnvelope{
					XmlnsSoapenv: "http://schemas.xmlsoap.org/soap/envelope/",
					Body: kycModels.DeleteKYCResponseBody{
						DeleteKYCResult: kycModels.DeleteKYCResult{XmlnsKyc: kycNamespaceAttr, Status: "Success", Message: "User deleted"},
					},
				}
			}
		}

	default:
		statusCode = http.StatusBadRequest
		responseEnvelope = kycModels.KYCResponseEnvelope{
			XmlnsSoapenv: "http://schemas.xmlsoap.org/soap/envelope/",
			Body: kycModels.KYCResponseBody{
				KYCResult: kycModels.KYCResult{XmlnsKyc: kycNamespaceAttr, Status: "Error", Message: fmt.Sprintf("Unknown SOAP operation: %s", root)}},
		}
	}

	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	w.WriteHeader(statusCode)

	// Marshal the specific response envelope into XML
	responseBytes, err := xml.MarshalIndent(responseEnvelope, "", "  ")
	if err != nil {
		log.Printf("KYC SOAP Server: Failed to marshal response envelope: %v", err)
		http.Error(w, fmt.Sprintf("Failed to marshal response: %v", err), http.StatusInternalServerError)
		return
	}

	_, err = w.Write(responseBytes)
	if err != nil {
		log.Printf("KYC SOAP Server: Failed to write response: %v", err)
	}
	log.Printf("KYC SOAP Server: Sent SOAP response (Status: %d):\n%s", statusCode, string(responseBytes))
}

// main function to start the SOAP server
func main() {
	repo := NewInMemoryRepo() // Use NewInMemoryRepo directly
	log.Println("KYC SOAP Server: Initializing with in-memory repository.")

	dataFolder := "tests/usecases/kyc"      // Define the path
	loadUserDataFromFiles(repo, dataFolder) // Call the new function

	// Create a new ServeMux for routing
	mux := http.NewServeMux()

	// Register SOAP handler
	mux.HandleFunc("/soap", func(w http.ResponseWriter, r *http.Request) {
		soapHandler(repo, w, r)
	})

	initAdminRoutes(mux, repo)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081" // Default port
	}
	listenAddr := fmt.Sprintf(":%s", port)

	log.Printf("KYC SOAP Server: Listening on %s", listenAddr)
	log.Fatal(http.ListenAndServe(listenAddr, mux))
}
