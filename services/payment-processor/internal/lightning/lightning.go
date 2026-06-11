package lightning

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"
)

// Client defines the interface for Lightning and Exchange Rate operations
type Client interface {
	CreateInvoice(amountSats int64, memo string) (paymentHash string, invoice string, err error)
	CheckInvoice(paymentHash string) (settled bool, err error)
	GetBTCKESRate() (float64, error)
}

// LNbitsClient implements Client using the LNbits REST API
type LNbitsClient struct {
	URL        string
	InvoiceKey string
	HTTPClient *http.Client
}

// MockClient implements Client locally without making external network calls (fallback)
type MockClient struct {
	paidInvoices map[string]bool
}

// NewClient returns an active LNbitsClient if configurations exist, otherwise returns MockClient
func NewClient(url, invoiceKey string) Client {
	if url == "" || invoiceKey == "" {
		log.Println("⚡ No LNbits configurations found. Initializing Lightning Mock Client")
		return &MockClient{paidInvoices: make(map[string]bool)}
	}

	log.Printf("🎉 Initializing LNbits Lightning Client. Endpoint: %s\n", url)
	return &LNbitsClient{
		URL:        url,
		InvoiceKey: invoiceKey,
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// ==========================================
// LNbitsClient Implementation
// ==========================================

type lnbitsCreateInvoiceReq struct {
	Out     bool   `json:"out"`
	Amount  int64  `json:"amount"` // in Sats
	Memo    string `json:"memo"`
	Webhook string `json:"webhook,omitempty"`
}

type lnbitsCreateInvoiceResp struct {
	PaymentHash string `json:"payment_hash"`
	PaymentRequest string `json:"payment_request"` // bolt11
}

type lnbitsCheckInvoiceResp struct {
	Paid bool `json:"paid"`
}

func (c *LNbitsClient) CreateInvoice(amountSats int64, memo string) (string, string, error) {
	reqBody := lnbitsCreateInvoiceReq{
		Out:    false,
		Amount: amountSats,
		Memo:   memo,
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", "", err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/api/v1/payments", c.URL), bytes.NewBuffer(jsonBytes))
	if err != nil {
		return "", "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Key", c.InvoiceKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("lnbits returned status code %d", resp.StatusCode)
	}

	var result lnbitsCreateInvoiceResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", err
	}

	return result.PaymentHash, result.PaymentRequest, nil
}

func (c *LNbitsClient) CheckInvoice(paymentHash string) (bool, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/payments/%s", c.URL, paymentHash), nil)
	if err != nil {
		return false, err
	}

	req.Header.Set("X-Api-Key", c.InvoiceKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("lnbits payment check returned status %d", resp.StatusCode)
	}

	var result lnbitsCheckInvoiceResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}

	return result.Paid, nil
}

func (c *LNbitsClient) GetBTCKESRate() (float64, error) {
	// Call public Coinbase API for exchange rate
	resp, err := c.HTTPClient.Get("https://api.coinbase.com/v2/prices/BTC-KES/spot")
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("coinbase API status: %d", resp.StatusCode)
	}

	var payload struct {
		Data struct {
			Amount string `json:"amount"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return 0, err
	}

	var rate float64
	_, err = fmt.Sscanf(payload.Data.Amount, "%f", &rate)
	if err != nil {
		return 0, err
	}

	return rate, nil
}

// ==========================================
// MockClient Implementation (Mock Fallback)
// ==========================================

func (m *MockClient) CreateInvoice(amountSats int64, memo string) (string, string, error) {
	// Generate random payment hash
	randBytes := make([]byte, 16)
	rand.Read(randBytes)
	h := sha256.New()
	h.Write(randBytes)
	paymentHash := hex.EncodeToString(h.Sum(nil))

	// Generate fake bolt11 payment request
	invoice := fmt.Sprintf("lnbc%ds1p%smockpaymentrequest...", amountSats, paymentHash[:10])

	return paymentHash, invoice, nil
}

func (m *MockClient) CheckInvoice(paymentHash string) (bool, error) {
	// We simulate a mock invoice settlement check.
	// In mock mode, the developer can trigger a payment webhook directly,
	// or we can simulate payment after a short duration.
	return false, nil
}

func (m *MockClient) GetBTCKESRate() (float64, error) {
	// Coinbase public API works on local host too, let's try calling it, and fallback if internet is out.
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("https://api.coinbase.com/v2/prices/BTC-KES/spot")
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			var payload struct {
				Data struct {
					Amount string `json:"amount"`
				} `json:"data"`
			}
			if json.NewDecoder(resp.Body).Decode(&payload) == nil {
				var rate float64
				if _, scanErr := fmt.Sscanf(payload.Data.Amount, "%f", &rate); scanErr == nil {
					return rate, nil
				}
			}
		}
	}

	// Default fallback exchange rate (e.g. 1 BTC = 9,000,000 KES)
	log.Println("⚠️ Coinbase exchange rate query failed or offline. Using fallback rate: 9,000,000 KES/BTC")
	return 9000000.0, nil
}
