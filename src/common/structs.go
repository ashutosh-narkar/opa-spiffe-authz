package common

// Patient holds patient info
type Patient struct {
	ID           string `json:"id,omitempty"`
	Firstname    string `json:"firstname,omitempty"`
	Lastname     string `json:"lastname,omitempty"`
	SSN          string `json:"ssn,omitempty"`
	EnrolleeType string `json:"enrollee_type,omitempty"`
}

// Result holds the final response to return to the client
type Result struct {
	Client           string    `json:"client,omitempty"`
	ConnectionStatus string    `json:"connection_status,omitempty"`
	Reason           string    `json:"reason,omitempty"`
	Patients         []Patient `json:"patients,omitempty"`
}
