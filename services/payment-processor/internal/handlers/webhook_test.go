package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/sky-fee/payment-processor/internal/db"
	"github.com/sky-fee/payment-processor/internal/models"
)

func TestLightningWebhook(t *testing.T) {
	ms := db.NewMemoryStore()
	store := &mockStore{MemoryStore: ms}
	ln := &mockLightning{}
	mp := &mockMpesa{}
	s := NewServer(store, ln, mp)
	r := chi.NewRouter()
	s.RegisterRoutes(r)

	t.Run("Valid payment_hash settles payment", func(t *testing.T) {
		hash := "webhook-success-hash"
		id := uuid.New()
		p := &models.Payment{
			ID:          id,
			SchoolID:    1,
			PaymentHash: hash,
			Status:      models.StatusPending,
			AmountSats:  1000,
		}
		store.CreatePayment(p)

		body := map[string]string{"payment_hash": hash}
		jsonBody, _ := json.Marshal(body)
		req, _ := http.NewRequest("POST", "/api/webhooks/lightning", bytes.NewBuffer(jsonBody))
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}

		var resp map[string]string
		json.NewDecoder(rr.Body).Decode(&resp)
		if resp["status"] != "ok" {
			t.Errorf("expected status ok, got %s", resp["status"])
		}

		updated, _ := store.GetPayment(id)
		if updated.Status != models.StatusDisbursed {
			t.Errorf("expected status DISBURSED, got %s", updated.Status)
		}
	})

	t.Run("Invalid JSON returns 400", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/webhooks/lightning", bytes.NewBufferString("invalid-json"))
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rr.Code)
		}
	})

	t.Run("Missing payment_hash returns 400", func(t *testing.T) {
		body := map[string]string{}
		jsonBody, _ := json.Marshal(body)
		req, _ := http.NewRequest("POST", "/api/webhooks/lightning", bytes.NewBuffer(jsonBody))
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rr.Code)
		}
	})

	t.Run("Unknown payment_hash returns 500", func(t *testing.T) {
		body := map[string]string{"payment_hash": "unknown-hash"}
		jsonBody, _ := json.Marshal(body)
		req, _ := http.NewRequest("POST", "/api/webhooks/lightning", bytes.NewBuffer(jsonBody))
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rr.Code)
		}
	})

	t.Run("Settlement failure returns 500 and marks FAILED", func(t *testing.T) {
		hash := "webhook-fail-hash"
		id := uuid.New()
		p := &models.Payment{
			ID:          id,
			SchoolID:    1,
			PaymentHash: hash,
			Status:      models.StatusPending,
			AmountSats:  1000,
		}
		store.CreatePayment(p)

		mp.offRampErr = errors.New("offramp error")
		defer func() { mp.offRampErr = nil }()

		body := map[string]string{"payment_hash": hash}
		jsonBody, _ := json.Marshal(body)
		req, _ := http.NewRequest("POST", "/api/webhooks/lightning", bytes.NewBuffer(jsonBody))
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rr.Code)
		}

		updated, _ := store.GetPayment(id)
		if updated.Status != models.StatusFailed {
			t.Errorf("expected status FAILED, got %s", updated.Status)
		}
	})
}
