package handlers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/sky-fee/payment-processor/internal/db"
	"github.com/sky-fee/payment-processor/internal/models"
)

// Configurable fakes for error injection
type mockStore struct {
	*db.MemoryStore
	getSchoolErr error
}

func (m *mockStore) GetSchool(id int) (*models.School, error) {
	if m.getSchoolErr != nil {
		return nil, m.getSchoolErr
	}
	return m.MemoryStore.GetSchool(id)
}

type mockLightning struct {
	fakeLightningClient
	rateErr error
}

func (m *mockLightning) GetBTCKESRate() (float64, error) {
	if m.rateErr != nil {
		return 0, m.rateErr
	}
	return m.fakeLightningClient.GetBTCKESRate()
}

type mockMpesa struct {
	fakeMpesaService
	offRampErr error
	payoutErr  error
}

func (m *mockMpesa) ExecuteOffRamp(amountSats int64, targetRate float64) (float64, error) {
	if m.offRampErr != nil {
		return 0, m.offRampErr
	}
	return m.fakeMpesaService.ExecuteOffRamp(amountSats, targetRate)
}

func (m *mockMpesa) PayoutSchoolFees(paybill string, accountNumber string, amountKES float64) (string, error) {
	if m.payoutErr != nil {
		return "", m.payoutErr
	}
	return m.fakeMpesaService.PayoutSchoolFees(paybill, accountNumber, amountKES)
}

func TestProcessInvoiceSettlement(t *testing.T) {
	ms := db.NewMemoryStore()
	store := &mockStore{MemoryStore: ms}
	ln := &mockLightning{}
	mp := &mockMpesa{}
	s := NewServer(store, ln, mp)

	setupPayment := func(hash string) uuid.UUID {
		id := uuid.New()
		p := &models.Payment{
			ID:          id,
			SchoolID:    1,
			PaymentHash: hash,
			Status:      models.StatusPending,
			AmountSats:  1000,
		}
		store.CreatePayment(p)
		return id
	}

	t.Run("Successful settlement transitions to DISBURSED", func(t *testing.T) {
		hash := "success-hash"
		id := setupPayment(hash)

		err := s.ProcessInvoiceSettlement(hash)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}

		p, _ := store.GetPayment(id)
		if p.Status != models.StatusDisbursed {
			t.Errorf("expected status DISBURSED, got %s", p.Status)
		}
		if p.MPesaReceipt != "MOCKRECEIPT" {
			t.Errorf("expected receipt MOCKRECEIPT, got %s", p.MPesaReceipt)
		}
	})

	t.Run("Duplicate settlement is idempotent", func(t *testing.T) {
		hash := "idempotent-hash"
		setupPayment(hash)

		// First call
		s.ProcessInvoiceSettlement(hash)
		
		// Second call
		err := s.ProcessInvoiceSettlement(hash)
		if err != nil {
			t.Fatalf("expected nil error on duplicate, got %v", err)
		}
	})

	t.Run("Unknown payment hash returns error", func(t *testing.T) {
		err := s.ProcessInvoiceSettlement("unknown-hash")
		if err == nil {
			t.Error("expected error for unknown hash, got nil")
		}
	})

	t.Run("Settlement FAILED if school lookup fails", func(t *testing.T) {
		hash := "school-fail-hash"
		id := setupPayment(hash)
		store.getSchoolErr = errors.New("db error")
		defer func() { store.getSchoolErr = nil }()

		err := s.ProcessInvoiceSettlement(hash)
		if err == nil {
			t.Error("expected error, got nil")
		}

		p, _ := store.GetPayment(id)
		if p.Status != models.StatusFailed {
			t.Errorf("expected status FAILED, got %s", p.Status)
		}
	})

	t.Run("Settlement FAILED if off-ramp fails", func(t *testing.T) {
		hash := "offramp-fail-hash"
		id := setupPayment(hash)
		mp.offRampErr = errors.New("offramp error")
		defer func() { mp.offRampErr = nil }()

		err := s.ProcessInvoiceSettlement(hash)
		if err == nil {
			t.Error("expected error, got nil")
		}

		p, _ := store.GetPayment(id)
		if p.Status != models.StatusFailed {
			t.Errorf("expected status FAILED, got %s", p.Status)
		}
	})

	t.Run("Settlement FAILED if M-Pesa payout fails", func(t *testing.T) {
		hash := "payout-fail-hash"
		id := setupPayment(hash)
		mp.payoutErr = errors.New("payout error")
		defer func() { mp.payoutErr = nil }()

		err := s.ProcessInvoiceSettlement(hash)
		if err == nil {
			t.Error("expected error, got nil")
		}

		p, _ := store.GetPayment(id)
		if p.Status != models.StatusFailed {
			t.Errorf("expected status FAILED, got %s", p.Status)
		}
	})
}

func TestTriggerMockPaymentEndpoint(t *testing.T) {
	ms := db.NewMemoryStore()
	store := &mockStore{MemoryStore: ms}
	ln := &mockLightning{}
	mp := &mockMpesa{}
	s := NewServer(store, ln, mp)
	r := chi.NewRouter()
	s.RegisterRoutes(r)

	t.Run("Settle pending payment returns 200", func(t *testing.T) {
		id := uuid.New()
		p := &models.Payment{
			ID:          id,
			SchoolID:    1,
			PaymentHash: "settle-endpoint-hash",
			Status:      models.StatusPending,
		}
		store.CreatePayment(p)

		req, _ := http.NewRequest("POST", "/api/payments/"+id.String()+"/settle", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("Settle malformed UUID returns 400", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/payments/invalid-uuid/settle", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rr.Code)
		}
	})

	t.Run("Settle unknown UUID returns 404", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/payments/"+uuid.New().String()+"/settle", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", rr.Code)
		}
	})

	t.Run("Settle already processed payment returns 400", func(t *testing.T) {
		id := uuid.New()
		p := &models.Payment{
			ID:          id,
			Status:      models.StatusPaid,
			UpdatedAt:   time.Now(),
		}
		store.CreatePayment(p)

		req, _ := http.NewRequest("POST", "/api/payments/"+id.String()+"/settle", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rr.Code)
		}
	})
}
