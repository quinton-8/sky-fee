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

func (m *MockMpesaService) ExecuteOffRamp(amountSats int64, targetRate float64) (float64, error) {
	// KES amount is derived from satoshis: amountSats / 100,000,000 * rate
	btcAmount := float64(amountSats) / 100000000.0
	amountKES := btcAmount * targetRate
	
	log.Printf("💸 [Off-Ramp Mock] Converting %d Sats (~%.8f BTC) to KES at rate %.2f -> Received KES %.2f\n", 
		amountSats, btcAmount, targetRate, amountKES)
	
	return amountKES, nil
}

func (m *MockMpesaService) PayoutSchoolFees(paybill string, accountNumber string, amountKES float64) (string, error) {
	// Simulate Safaricom M-Pesa API response time
	time.Sleep(500 * time.Millisecond)
	
	// Generate mock M-Pesa transaction ID (e.g. SGR5A9X7F2)
	rand.Seed(time.Now().UnixNano())
	const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 10)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	receiptNumber := string(b)
	
	log.Printf("📲 [M-Pesa Mock] Disbursed KES %.2f to Paybill: %s, Account: %s. Receipt: %s\n", 
		amountKES, paybill, accountNumber, receiptNumber)
	
	return receiptNumber, nil
}
