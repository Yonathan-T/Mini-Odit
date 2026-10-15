package model

import "time"

type Participant struct {
	Name  string `json:"name,omitempty"`
	Phone string `json:"phone,omitempty"`
	Role  string `json:"role,omitempty"`
	Type  string `json:"type,omitempty"`
}

type Transaction struct {
	Provider        string            `json:"provider"`
	PatternName     string            `json:"pattern_name"`
	Category        string            `json:"category"`
	Direction       Direction         `json:"direction"`
	Principal       float64           `json:"principal"`
	Fees            float64           `json:"fees"`
	TotalAmount     float64           `json:"total_amount"`
	Balance         *float64          `json:"balance,omitempty"`
	Participants    []Participant     `json:"participants,omitempty"`
	TransactionIDs  map[string]string `json:"transaction_ids,omitempty"`
	ReceiptURL      string            `json:"receipt_url,omitempty"`
	RawTimestamp    string            `json:"raw_timestamp,omitempty"`
	ParsedTime      *time.Time        `json:"parsed_time,omitempty"`
	ExtractedFields map[string]string `json:"extracted_fields,omitempty"`
	RawMessage      string            `json:"raw_message"`
	ExtractedAt     time.Time         `json:"extracted_at"`
}
