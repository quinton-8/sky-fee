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
