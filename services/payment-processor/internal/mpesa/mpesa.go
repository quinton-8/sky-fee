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

// EncryptSecurityCredential encrypts the initiator password using the Safaricom public key certificate.
func EncryptSecurityCredential(initiatorPassword string, certPEM string) (string, error) {
	block, _ := pem.Decode([]byte(certPEM))
	if block == nil {
		return "", errors.New("failed to parse certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("failed to parse x509 certificate: %w", err)
	}

	pubKey, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return "", errors.New("certificate does not contain an RSA public key")
	}

	// Safaricom public key certificate usually requires PKCS1v15 padding
	encrypted, err := rsa.EncryptPKCS1v15(cryptorand.Reader, pubKey, []byte(initiatorPassword))
	if err != nil {
		return "", fmt.Errorf("failed to encrypt password: %w", err)
	}

	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// getAccessToken fetches the OAuth token from Safaricom API
func (s *RealMpesaService) getAccessToken() (string, error) {
	url := "https://sandbox.safaricom.co.ke/oauth/v1/generate?grant_type=client_credentials"
	if s.env == "production" {
		url = "https://api.safaricom.co.ke/oauth/v1/generate?grant_type=client_credentials"
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	req.SetBasicAuth(s.consumerKey, s.consumerSecret)
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("oauth token generation failed with status %d: %s", resp.StatusCode, string(respBytes))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   string `json:"expires_in"`
	}
	if err := json.Unmarshal(respBytes, &tokenResp); err != nil {
		return "", err
	}

	return tokenResp.AccessToken, nil
}

// ExecuteOffRamp converts satoshis to KES currency using fiat off-ramp service APIs
func (s *RealMpesaService) ExecuteOffRamp(amountSats int64, targetRate float64) (float64, error) {
	// KES amount is derived from satoshis: amountSats / 100,000,000 * rate
	btcAmount := float64(amountSats) / 100000000.0
	amountKES := btcAmount * targetRate

	log.Printf(" [Off-Ramp] Initiating real off-ramp conversion of %d Sats (~%.8f BTC) to KES at rate %.2f -> Expected KES %.2f\n", 
		amountSats, btcAmount, targetRate, amountKES)

	// If no provider API key/URL is configured, we calculate the KES value and return it (simulating off-ramp)
	if s.offrampAPIKey == "" {
		log.Println(" Off-ramp API key not configured. Returning calculated KES value without calling external API.")
		return amountKES, nil
	}

	url := s.offrampAPIURL
	if url == "" {
		switch s.offrampProvider {
		case "bitnob":
			url = "https://api.bitnob.co/v1/wallets/sell"
		case "kotanipay":
			url = "https://api.kotanipay.com/api/v1/payments/withdraw"
		default:
			url = "https://api.kotanipay.com/api/v1/payments/withdraw"
		}
	}

	var reqBody []byte
	var err error

	if s.offrampProvider == "bitnob" {
		payload := map[string]interface{}{
			"amount":        amountSats,
			"currency":      "SATS",
			"fiat_currency": "KES",
		}
		reqBody, err = json.Marshal(payload)
	} else {
		payload := map[string]interface{}{
			"amount":         amountKES,
			"currency":       "KES",
			"payment_method": "mpesa",
		}
		reqBody, err = json.Marshal(payload)
	}
	if err != nil {
		return 0, fmt.Errorf("failed to marshal off-ramp payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return 0, fmt.Errorf("failed to create off-ramp request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.offrampAPIKey)
	req.Header.Set("X-Api-Key", s.offrampAPIKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("off-ramp API call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusAccepted {
		respBytes, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("off-ramp API returned error status %d: %s", resp.StatusCode, string(respBytes))
	}

	log.Printf("🎉 [Off-Ramp] Conversion request successfully sent. Provider responded with status %d.\n", resp.StatusCode)
	return amountKES, nil
}

// PayoutSchoolFees sends M-Pesa B2C/B2B school fee payments using Safaricom's Daraja APIs
func (s *RealMpesaService) PayoutSchoolFees(paybill string, accountNumber string, amountKES float64) (string, error) {
	log.Printf("📲 [M-Pesa] Disbursing KES %.2f to Paybill/Recipient: %s, Account: %s\n", amountKES, paybill, accountNumber)

	token, err := s.getAccessToken()
	if err != nil {
		return "", fmt.Errorf("failed to get M-Pesa access token: %w", err)
	}

	// 1. Prepare SecurityCredential
	securityCredential := s.securityCred
	if securityCredential == "" {
		if s.certPEM != "" && s.initiatorPwd != "" {
			var encErr error
			securityCredential, encErr = EncryptSecurityCredential(s.initiatorPwd, s.certPEM)
			if encErr != nil {
				log.Printf("⚠️ Failed to encrypt initiator password: %v. Falling back to raw password.\n", encErr)
				securityCredential = s.initiatorPwd
			}
		} else {
			securityCredential = s.initiatorPwd
		}
	}

	// 2. Determine if recipient is a phone number (B2C) or a shortcode/paybill (B2B)
	isPhone := false
	cleanedPaybill := strings.TrimSpace(paybill)
	if strings.HasPrefix(cleanedPaybill, "+") || strings.HasPrefix(cleanedPaybill, "254") || strings.HasPrefix(cleanedPaybill, "07") || strings.HasPrefix(cleanedPaybill, "01") {
		isPhone = true
	}

	var url string
	var reqBody []byte

	baseDomain := "sandbox.safaricom.co.ke"
	if s.env == "production" {
		baseDomain = "api.safaricom.co.ke"
	}

	amountInt := int64(amountKES)
	if amountInt <= 0 {
		amountInt = 1
	}

	if isPhone {
		// B2C Payout
		url = fmt.Sprintf("https://%s/mpesa/b2c/v1/paymentrequest", baseDomain)
		
		commandID := os.Getenv("MPESA_B2C_COMMAND_ID")
		if commandID == "" {
			commandID = "BusinessPayment"
		}

		b2cReq := map[string]interface{}{
			"InitiatorName":      s.initiatorName,
			"SecurityCredential": securityCredential,
			"CommandID":          commandID,
			"Amount":             amountInt,
			"PartyA":             s.shortcode,
			"PartyB":             cleanedPaybill,
			"Remarks":            "School fees disbursment",
			"QueueTimeOutURL":    s.callbackURL + "/timeout",
			"ResultURL":          s.callbackURL + "/result",
			"Occasion":           "SchoolFees",
		}
		reqBody, err = json.Marshal(b2cReq)
	} else {
		// B2B Payout
		url = fmt.Sprintf("https://%s/mpesa/b2b/v1/paymentrequest", baseDomain)
		
		commandID := os.Getenv("MPESA_B2B_COMMAND_ID")
		if commandID == "" {
			commandID = "BusinessPayBill"
		}

		b2bReq := map[string]interface{}{
			"Initiator":              s.initiatorName,
			"SecurityCredential":      securityCredential,
			"CommandID":              commandID,
			"SenderIdentifierType":   "4",
			"RecieverIdentifierType": "4",
			"Amount":                 amountInt,
			"PartyA":                 s.shortcode,
			"PartyB":                 cleanedPaybill,
			"AccountReference":       accountNumber,
			"Remarks":                "School fees disbursment to paybill",
			"QueueTimeOutURL":        s.callbackURL + "/timeout",
			"ResultURL":              s.callbackURL + "/result",
		}
		reqBody, err = json.Marshal(b2bReq)
	}

	if err != nil {
		return "", fmt.Errorf("failed to marshal payout payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to create payout request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("payout API call failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return "", fmt.Errorf("M-Pesa payout failed with status %d: %s", resp.StatusCode, string(respBytes))
	}

	var mpesaResp struct {
		ConversationID           string `json:"ConversationID"`
		OriginatorConversationID string `json:"OriginatorConversationID"`
		ResponseCode             string `json:"ResponseCode"`
		ResponseDescription      string `json:"ResponseDescription"`
	}

	if err := json.Unmarshal(respBytes, &mpesaResp); err != nil {
		log.Printf("⚠️ Failed to parse M-Pesa response JSON: %v. Raw response: %s\n", err, string(respBytes))
		return "MP_TX_" + time.Now().Format("20060102150405"), nil
	}

	if mpesaResp.ResponseCode != "0" {
		return "", fmt.Errorf("M-Pesa payout rejected: Code %s - %s", mpesaResp.ResponseCode, mpesaResp.ResponseDescription)
	}

	log.Printf("🎉 [M-Pesa] Payout request accepted. ConversationID: %s\n", mpesaResp.ConversationID)

	if s.env == "sandbox" || s.env == "" {
		if len(mpesaResp.ConversationID) > 6 {
			return fmt.Sprintf("SNDB_%s", mpesaResp.ConversationID[len(mpesaResp.ConversationID)-6:]), nil
		}
		return "SNDB_" + mpesaResp.ConversationID, nil
	}

	return mpesaResp.ConversationID, nil
}
