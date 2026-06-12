package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sky-fee/payment-processor/internal/db"
	"github.com/sky-fee/payment-processor/internal/models"
)

func TestHandleWebSocket(t *testing.T) {
	ms := db.NewMemoryStore()
	store := &mockStore{MemoryStore: ms}
	ln := &mockLightning{}
	mp := &mockMpesa{}
	s := NewServer(store, ln, mp)
	r := chi.NewRouter()
	s.RegisterRoutes(r)

	ts := httptest.NewServer(r)
	defer ts.Close()

	wsURL := strings.Replace(ts.URL, "http", "ws", 1)

	t.Run("Reject malformed UUID", func(t *testing.T) {
		_, resp, err := websocket.DefaultDialer.Dial(wsURL+"/api/payments/invalid-uuid/ws", nil)
		if err == nil {
			t.Fatal("expected error dialing malformed UUID, got nil")
		}
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", resp.StatusCode)
		}
	})

	t.Run("Successful upgrade and notification", func(t *testing.T) {
		paymentID := uuid.New()
		paymentHash := "ws-success-hash"
		p := &models.Payment{
			ID:          paymentID,
			SchoolID:    1,
			PaymentHash: paymentHash,
			Status:      models.StatusPending,
			AmountSats:  1000,
		}
		store.CreatePayment(p)

		// Dial the WebSocket
		conn, resp, err := websocket.DefaultDialer.Dial(wsURL+"/api/payments/"+paymentID.String()+"/ws", nil)
		if err != nil {
			t.Fatalf("failed to dial: %v", err)
		}
		defer conn.Close()

		if resp.StatusCode != http.StatusSwitchingProtocols {
			t.Errorf("expected 101, got %d", resp.StatusCode)
		}

		// Trigger settlement in a goroutine
		errChan := make(chan error, 1)
		go func() {
			// Small delay to ensure WS registration is processed
			time.Sleep(50 * time.Millisecond)
			errChan <- s.ProcessInvoiceSettlement(paymentHash)
		}()

		// Read from WebSocket
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		var updatedPayment models.Payment
		err = conn.ReadJSON(&updatedPayment)
		if err != nil {
			// It might be the first notification (PAID) or the second (DISBURSED)
			// ProcessInvoiceSettlement notifies twice: once for PAID, once for DISBURSED
			t.Fatalf("failed to read JSON from WS: %v", err)
		}

		// If first message is PAID, read again for DISBURSED
		if updatedPayment.Status == models.StatusPaid {
			err = conn.ReadJSON(&updatedPayment)
			if err != nil {
				t.Fatalf("failed to read second JSON from WS: %v", err)
			}
		}

		if updatedPayment.Status != models.StatusDisbursed {
			t.Errorf("expected status DISBURSED, got %s", updatedPayment.Status)
		}

		if err := <-errChan; err != nil {
			t.Fatalf("settlement failed: %v", err)
		}
	})
}
