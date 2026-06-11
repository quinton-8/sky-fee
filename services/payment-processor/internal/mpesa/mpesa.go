package mpesa

import (
	"bytes"
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log"
	mathrand "math/rand"
	"net/http"
	"os"
	"strings"
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

// ExecuteOffRamp converts satoshis to KES currency (Mock)
func (m *MockMpesaService) ExecuteOffRamp(amountSats int64, targetRate float64) (float64, error) {
	// KES amount is derived from satoshis: amountSats / 100,000,000 * rate
	btcAmount := float64(amountSats) / 100000000.0
	amountKES := btcAmount * targetRate
	
	log.Printf("💸 [Off-Ramp Mock] Converting %d Sats (~%.8f BTC) to KES at rate %.2f -> Received KES %.2f\n", 
		amountSats, btcAmount, targetRate, amountKES)
	
	return amountKES, nil
}

// PayoutSchoolFees sends M-Pesa B2C school fee payments (Mock)
func (m *MockMpesaService) PayoutSchoolFees(paybill string, accountNumber string, amountKES float64) (string, error) {
	// Simulate Safaricom M-Pesa API response time
	time.Sleep(500 * time.Millisecond)
	
	// Generate mock M-Pesa transaction ID (e.g. SGR5A9X7F2)
	mathrand.Seed(time.Now().UnixNano())
	const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 10)
	for i := range b {
		b[i] = letters[mathrand.Intn(len(letters))]
	}
	receiptNumber := string(b)
	
	log.Printf("📲 [M-Pesa Mock] Disbursed KES %.2f to Paybill: %s, Account: %s. Receipt: %s\n", 
		amountKES, paybill, accountNumber, receiptNumber)
	
	return receiptNumber, nil
}

// RealMpesaService handles real integration with Safaricom Daraja API and fiat off-ramp service APIs
type RealMpesaService struct {
	client          *http.Client
	env             string // "sandbox" or "production"
	consumerKey     string
	consumerSecret  string
	shortcode       string // PartyA (initiator shortcode)
	initiatorName   string
	initiatorPwd    string
	securityCred    string // pre-encrypted security credential (optional)
	certPEM         string // certificate PEM for RSA encryption (optional)
	callbackURL     string // QueueTimeOutURL and ResultURL callback base
	offrampProvider string // "kotanipay", "bitnob", or "generic"
	offrampAPIKey   string
	offrampAPIURL   string
}

// NewService returns a RealMpesaService if the required credentials are set,
// otherwise it falls back to a MockMpesaService for local development.
func NewService() Service {
	consumerKey := os.Getenv("MPESA_CONSUMER_KEY")
	consumerSecret := os.Getenv("MPESA_CONSUMER_SECRET")

	if consumerKey == "" || consumerSecret == "" {
		log.Println("⚡ Initializing M-Pesa & Off-Ramp Mock Service (Credentials missing: MPESA_CONSUMER_KEY/MPESA_CONSUMER_SECRET)")
		return &MockMpesaService{}
	}

	log.Println("🎉 Initializing Real M-Pesa & Off-Ramp Service with Safaricom Daraja API")

	env := os.Getenv("MPESA_ENV")
	if env == "" {
		env = "sandbox"
	}

	shortcode := os.Getenv("MPESA_SHORTCODE")
	if shortcode == "" {
		shortcode = "600192" // Default Safaricom sandbox shortcode
	}

	initiatorName := os.Getenv("MPESA_INITIATOR_NAME")
	if initiatorName == "" {
		initiatorName = "testapi" // Default Safaricom sandbox initiator name
	}

	initiatorPwd := os.Getenv("MPESA_INITIATOR_PASSWORD")
	securityCred := os.Getenv("MPESA_SECURITY_CREDENTIAL")

	certPEM := os.Getenv("MPESA_CERT_PEM")
	if certPEM == "" {
		certPath := os.Getenv("MPESA_CERT_PATH")
		if certPath != "" {
			certBytes, err := os.ReadFile(certPath)
			if err == nil {
				certPEM = string(certBytes)
			}
		}
	}

	callbackURL := os.Getenv("MPESA_CALLBACK_URL")
	if callbackURL == "" {
		callbackURL = "https://example.com/mpesa"
	}

	offrampProvider := os.Getenv("OFFRAMP_PROVIDER")
	if offrampProvider == "" {
		offrampProvider = "kotanipay"
	}

	offrampAPIKey := os.Getenv("OFFRAMP_API_KEY")
	offrampAPIURL := os.Getenv("OFFRAMP_API_URL")

	return &RealMpesaService{
		client:          &http.Client{Timeout: 15 * time.Second},
		env:             env,
		consumerKey:     consumerKey,
		consumerSecret:  consumerSecret,
		shortcode:       shortcode,
		initiatorName:   initiatorName,
		initiatorPwd:    initiatorPwd,
		securityCred:    securityCred,
		certPEM:         certPEM,
		callbackURL:     callbackURL,
		offrampProvider: offrampProvider,
		offrampAPIKey:   offrampAPIKey,
		offrampAPIURL:   offrampAPIURL,
	}
}

