package models

import (
	"time"

	"github.com/google/uuid"
)

// PaymentStatus represents the current state of a school fees transaction
type PaymentStatus string



// School represents an enrolled school on the SkyFee platform
type School struct {
	ID         int       `json:"id" db:"id"`
	Name       string    `json:"name" db:"name"`
	Paybill    string    `json:"paybill" db:"paybill"`         // M-Pesa Paybill number
	AccountNumber string `json:"account_number" db:"account_number"` // School M-Pesa account indicator (e.g. Student Adm No)
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// Student represents a registered student in an enrolled school
type Student struct {
	ID              int       `json:"id" db:"id"`
	SchoolID        int       `json:"school_id" db:"school_id"`
	AdmissionNumber string    `json:"admission_number" db:"admission_number"`
	Name            string    `json:"name" db:"name"`
	Grade           string    `json:"grade" db:"grade"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

// Payment represents a single transaction where a parent pays school fees via Lightning
type Payment struct {
	ID                     uuid.UUID     `json:"id" db:"id"`
	SchoolID               int           `json:"school_id" db:"school_id"`
	StudentAdmissionNumber string        `json:"student_admission_number" db:"student_admission_number"`
	StudentName            string        `json:"student_name" db:"student_name"`
	ParentName             string        `json:"parent_name" db:"parent_name"`
	AmountKES              float64       `json:"amount_kes" db:"amount_kes"`
	AmountSats             int64         `json:"amount_sats" db:"amount_sats"`
	LightningInvoice       string        `json:"lightning_invoice" db:"lightning_invoice"`
	PaymentHash            string        `json:"payment_hash" db:"payment_hash"`
	Status                 PaymentStatus `json:"status" db:"status"`
	MPesaReceipt           string        `json:"mpesa_receipt" db:"mpesa_receipt"`
	CreatedAt              time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time     `json:"updated_at" db:"updated_at"`
}

// ExchangeRate holds the cached conversion rate details
type ExchangeRate struct {
	BTCKES    float64   `json:"btc_kes"`
	UpdatedAt time.Time `json:"updated_at"`
}
