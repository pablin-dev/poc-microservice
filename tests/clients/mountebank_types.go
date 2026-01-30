package clients

// Imposter represents a single Mountebank imposter.
type Imposter struct {
	Protocol string `json:"protocol"`
	Port     int    `json:"port"`
	Name     string `json:"name,omitempty"`
	// Add other fields as needed, based on Mountebank's response
}

// ImpostersResponse is the top-level structure for the GET /impostors response.
type ImpostersResponse struct {
	Imposters []Imposter `json:"imposters"`
}

// DetailedImposter represents a single Mountebank imposter with more details.
type DetailedImposter struct {
	Protocol  string        `json:"protocol"`
	Port      int           `json:"port"`
	Name      string        `json:"name,omitempty"`
	Requests  []interface{} `json:"requests,omitempty"`  // Can be complex, use interface{} for now
	Responses []interface{} `json:"responses,omitempty"` // Can be complex, use interface{} for now
	Stubs     []interface{} `json:"stubs,omitempty"`     // Can be complex, use interface{} for now
	// Add other fields as needed, based on Mountebank's response for a single imposter
}
