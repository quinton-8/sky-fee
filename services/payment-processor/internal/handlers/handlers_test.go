package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/sky-fee/payment-processor/internal/db"
	"github.com/sky-fee/payment-processor/internal/models"
)

// Fake implementations for testing
type fakeLightningClient struct{}

func (f *fakeLightningClient) CreateInvoice(amountSats int64, memo string) (string, string, error) {
	return "fake-hash-123", "lnbc1-fake-invoice", nil
}
func (f *fakeLightningClient) CheckInvoice(paymentHash string) (bool, error) {
	return false, nil
}
func (f *fakeLightningClient) GetBTCKESRate() (float64, error) {
	return 10000000.0, nil
}

type fakeMpesaService struct{}

func (f *fakeMpesaService) ExecuteOffRamp(amountSats int64, targetRate float64) (float64, error) {
	return 100.0, nil
}
func (f *fakeMpesaService) PayoutSchoolFees(paybill string, accountNumber string, amountKES float64) (string, error) {
	return "MOCKRECEIPT", nil
}

func setupTestServer() (*Server, *chi.Mux) {
	store := db.NewMemoryStore()
	ln := &fakeLightningClient{}
	mp := &fakeMpesaService{}
	s := NewServer(store, ln, mp)
	r := chi.NewRouter()
	s.RegisterRoutes(r)
	return s, r
}

func TestGetSchools(t *testing.T) {
	_, r := setupTestServer()
	req, _ := http.NewRequest("GET", "/api/schools", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var schools []models.School
	if err := json.NewDecoder(rr.Body).Decode(&schools); err != nil {
		t.Fatalf("failed to decode schools: %v", err)
	}

	if len(schools) == 0 {
		t.Error("expected seeded schools, got none")
	}
}

func TestVerifyStudent(t *testing.T) {
	_, r := setupTestServer()

	t.Run("Valid student", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/schools/1/students/AHS-8899", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}

		var student models.Student
		if err := json.NewDecoder(rr.Body).Decode(&student); err != nil {
			t.Fatalf("failed to decode student: %v", err)
		}
		if student.AdmissionNumber != "AHS-8899" {
			t.Errorf("expected AHS-8899, got %s", student.AdmissionNumber)
		}
	})

	t.Run("Invalid admission number", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/schools/1/students/INVALID", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", rr.Code)
		}
	})

	t.Run("Invalid school ID", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/schools/abc/students/AHS-8899", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rr.Code)
		}
	})
}

func TestCreatePayment(t *testing.T) {
	_, r := setupTestServer()

	t.Run("Valid request", func(t *testing.T) {
		body := map[string]interface{}{
			"school_id":                1,
			"student_admission_number": "AHS-8899",
			"parent_name":             "John Doe",
			"amount_kes":              500.0,
		}
		jsonBody, _ := json.Marshal(body)
		req, _ := http.NewRequest("POST", "/api/payments", bytes.NewBuffer(jsonBody))
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusCreated {
			t.Errorf("expected 201, got %d: %s", rr.Code, rr.Body.String())
		}

		var resp createPaymentResponse
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp.PaymentID == uuid.Nil {
			t.Error("expected non-nil payment ID")
		}
		if resp.LightningInvoice == "" {
			t.Error("expected non-empty lightning invoice")
		}
	})

	t.Run("Missing fields", func(t *testing.T) {
		body := map[string]interface{}{
			"school_id": 1,
		}
		jsonBody, _ := json.Marshal(body)
		req, _ := http.NewRequest("POST", "/api/payments", bytes.NewBuffer(jsonBody))
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rr.Code)
		}
	})

	t.Run("Unknown school", func(t *testing.T) {
		body := map[string]interface{}{
			"school_id":                999,
			"student_admission_number": "AHS-8899",
			"parent_name":             "John Doe",
			"amount_kes":              500.0,
		}
		jsonBody, _ := json.Marshal(body)
		req, _ := http.NewRequest("POST", "/api/payments", bytes.NewBuffer(jsonBody))
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", rr.Code)
		}
	})
}

func TestGetPayment(t *testing.T) {
	s, r := setupTestServer()

	// Seed a payment
	paymentID := uuid.New()
	p := &models.Payment{
		ID:                     paymentID,
		SchoolID:               1,
		StudentAdmissionNumber: "AHS-8899",
		Status:                 models.StatusPending,
	}
	s.Store.CreatePayment(p)

	t.Run("Retrieve existing payment", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/payments/"+paymentID.String(), nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}

		var got models.Payment
		if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
			t.Fatalf("failed to decode payment: %v", err)
		}
		if got.ID != paymentID {
			t.Errorf("expected ID %v, got %v", paymentID, got.ID)
		}
	})

	t.Run("Malformed UUID", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/payments/invalid-uuid", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rr.Code)
		}
	})

	t.Run("Unknown UUID", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/payments/"+uuid.New().String(), nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", rr.Code)
		}
	})
}
