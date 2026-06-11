package mpesa

import (
	"log"
	"math/rand"
	"time"
)

// Service defines the interface for converting BTC and executing payouts via M-Pesa
type Service interface {
	// ExecuteOffRamp converts satoshis to KES currency
	ExecuteOffRamp(amountSats int64, targetRate float64) (float64, error)
	
	// PayoutSchoolFees sends M-Pesa B2C school fee payments
	PayoutSchoolFees(paybill string, accountNumber string, amountKES float64) (receiptNumber string, err error)
}

// MockMpesaService is a stub implementation that simulates successful payouts
type MockMpesaService struct{}

// NewService returns a MockMpesaService by default. The Job 3 Engineer will wire their real API implementations here.
func NewService() Service {
	log.Println("⚡ Initializing M-Pesa & Off-Ramp Mock Service (Job 3 stub)")
	return &MockMpesaService{}
}
