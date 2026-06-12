package lightning

import (
	"strings"
	"testing"
)

func TestNewClient(t *testing.T) {
	t.Run("Empty configurations return MockClient", func(t *testing.T) {
		client := NewClient("", "")
		_, ok := client.(*MockClient)
		if !ok {
			t.Errorf("Expected MockClient, got %T", client)
		}
	})

	t.Run("Valid configurations return LNbitsClient", func(t *testing.T) {
		client := NewClient("http://localhost:5000", "invoice-key")
		_, ok := client.(*LNbitsClient)
		if !ok {
			t.Errorf("Expected LNbitsClient, got %T", client)
		}
	})
}

func TestMockClient_CreateInvoice(t *testing.T) {
	client := &MockClient{paidInvoices: make(map[string]bool)}
	amount := int64(1000)
	memo := "Test Invoice"

	hash, invoice, err := client.CreateInvoice(amount, memo)
	if err != nil {
		t.Fatalf("CreateInvoice failed: %v", err)
	}

	if hash == "" {
		t.Error("Expected non-empty payment hash")
	}

	if invoice == "" {
		t.Error("Expected non-empty invoice")
	}

	if !strings.HasPrefix(invoice, "lnbc") {
		t.Errorf("Expected invoice to start with lnbc, got %s", invoice)
	}
}

func TestMockClient_CheckInvoice(t *testing.T) {
	client := &MockClient{paidInvoices: make(map[string]bool)}
	
	settled, err := client.CheckInvoice("any-hash")
	if err != nil {
		t.Fatalf("CheckInvoice failed: %v", err)
	}

	if settled {
		t.Error("Expected MockClient.CheckInvoice to return false")
	}
}

func TestMockClient_GetBTCKESRate(t *testing.T) {
	client := &MockClient{paidInvoices: make(map[string]bool)}
	
	rate, err := client.GetBTCKESRate()
	// GetBTCKESRate might attempt network call but has fallback.
	// It should not fail if internet is down.
	if err != nil {
		t.Fatalf("GetBTCKESRate failed: %v", err)
	}

	if rate <= 0 {
		t.Errorf("Expected positive rate, got %f", rate)
	}
}
