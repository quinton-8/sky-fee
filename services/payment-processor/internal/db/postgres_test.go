package db

import (
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/sky-fee/payment-processor/internal/models"
)

func TestPostgresStore(t *testing.T) {
	connStr := os.Getenv("SKYFEE_TEST_DATABASE_URL")
	if connStr == "" {
		t.Skip("Skipping Postgres integration tests: SKYFEE_TEST_DATABASE_URL not set")
	}

	dbConn, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer dbConn.Close()

	if err := dbConn.Ping(); err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}

	// Initialize schema for testing
	schema, err := os.ReadFile("../../../database/schema.sql")
	if err != nil {
		t.Fatalf("Failed to read schema.sql: %v", err)
	}
	if _, err := dbConn.Exec(string(schema)); err != nil {
		t.Fatalf("Failed to execute schema: %v", err)
	}

	store := &PostgresStore{db: dbConn}

	// Test GetSchools
	t.Run("GetSchools", func(t *testing.T) {
		schools, err := store.GetSchools()
		if err != nil {
			t.Errorf("GetSchools() error: %v", err)
		}
		if len(schools) < 3 {
			t.Errorf("expected at least 3 seeded schools, got %d", len(schools))
		}
	})

	// Test GetSchool
	t.Run("GetSchool", func(t *testing.T) {
		s, err := store.GetSchool(1)
		if err != nil {
			t.Fatalf("GetSchool(1) error: %v", err)
		}
		if s.Name != "Alliance High School" {
			t.Errorf("expected Alliance High School, got %s", s.Name)
		}

		_, err = store.GetSchool(999)
		if err == nil {
			t.Error("expected error for unknown school ID, got nil")
		}
	})

	// Test GetStudent
	t.Run("GetStudent", func(t *testing.T) {
		s, err := store.GetStudent(1, "AHS-8899")
		if err != nil {
			t.Fatalf("GetStudent error: %v", err)
		}
		if s.Name != "John Kiprop" {
			t.Errorf("expected John Kiprop, got %s", s.Name)
		}

		_, err = store.GetStudent(1, "INVALID")
		if err == nil {
			t.Error("expected error for unknown student, got nil")
		}
	})

	// Test Payment Lifecycle
	t.Run("Payment Lifecycle", func(t *testing.T) {
		paymentID := uuid.New()
		paymentHash := "pg-test-hash-" + paymentID.String()
		p := &models.Payment{
			ID:                     paymentID,
			SchoolID:               1,
			StudentAdmissionNumber: "AHS-8899",
			StudentName:            "John Kiprop",
			ParentName:             "Jane Doe",
			AmountKES:              1200.50,
			AmountSats:             6000,
			LightningInvoice:       "lnbc...",
			PaymentHash:            paymentHash,
			Status:                 models.StatusPending,
			CreatedAt:              time.Now().UTC(),
			UpdatedAt:              time.Now().UTC(),
		}

		// Create
		err := store.CreatePayment(p)
		if err != nil {
			t.Fatalf("CreatePayment error: %v", err)
		}

		// Get by ID
		got, err := store.GetPayment(paymentID)
		if err != nil {
			t.Fatalf("GetPayment error: %v", err)
		}
		if got.PaymentHash != paymentHash {
			t.Errorf("expected hash %s, got %s", paymentHash, got.PaymentHash)
		}

		// Get by Hash
		gotByHash, err := store.GetPaymentByHash(paymentHash)
		if err != nil {
			t.Fatalf("GetPaymentByHash error: %v", err)
		}
		if gotByHash.ID != paymentID {
			t.Errorf("expected ID %v, got %v", paymentID, gotByHash.ID)
		}

		// Update Status
		receipt := "PG_RECEIPT_123"
		err = store.UpdatePaymentStatus(paymentID, models.StatusDisbursed, receipt)
		if err != nil {
			t.Fatalf("UpdatePaymentStatus error: %v", err)
		}

		updated, _ := store.GetPayment(paymentID)
		if updated.Status != models.StatusDisbursed {
			t.Errorf("expected status DISBURSED, got %s", updated.Status)
		}
		if updated.MPesaReceipt != receipt {
			t.Errorf("expected receipt %s, got %s", receipt, updated.MPesaReceipt)
		}
	})

	// Test Errors
	t.Run("Payment Errors", func(t *testing.T) {
		_, err := store.GetPayment(uuid.New())
		if err == nil {
			t.Error("expected error for unknown payment ID, got nil")
		}

		_, err = store.GetPaymentByHash("non-existent")
		if err == nil {
			t.Error("expected error for unknown payment hash, got nil")
		}
	})
}
