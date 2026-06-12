package db

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sky-fee/payment-processor/internal/models"
)

func TestNewMemoryStore(t *testing.T) {
	ms := NewMemoryStore()
	if ms == nil {
		t.Fatal("NewMemoryStore() returned nil")
	}

	schools, err := ms.GetSchools()
	if err != nil {
		t.Errorf("GetSchools() error: %v", err)
	}
	if len(schools) == 0 {
		t.Error("GetSchools() returned empty list, expected seeded schools")
	}
}

func TestMemoryStore_GetSchool(t *testing.T) {
	ms := NewMemoryStore()

	tests := []struct {
		name    string
		id      int
		wantErr bool
	}{
		{"Valid school 1", 1, false},
		{"Valid school 2", 2, false},
		{"Valid school 3", 3, false},
		{"Invalid school 4", 4, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ms.GetSchool(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetSchool() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got.ID != tt.id {
				t.Errorf("GetSchool() got ID = %v, want %v", got.ID, tt.id)
			}
		})
	}
}

func TestMemoryStore_GetStudent(t *testing.T) {
	ms := NewMemoryStore()

	tests := []struct {
		name      string
		schoolID  int
		admNumber string
		wantErr   bool
	}{
		{"Valid student 1", 1, "AHS-8899", false},
		{"Valid student 2", 1, "AHS-9012", false},
		{"Valid student 3", 2, "KHS-4455", false},
		{"Valid student 4", 3, "LEN-1234", false},
		{"Invalid school ID", 4, "AHS-8899", true},
		{"Invalid admission number", 1, "INVALID", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ms.GetStudent(tt.schoolID, tt.admNumber)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetStudent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.SchoolID != tt.schoolID {
					t.Errorf("GetStudent() got schoolID = %v, want %v", got.SchoolID, tt.schoolID)
				}
				if got.AdmissionNumber != tt.admNumber {
					t.Errorf("GetStudent() got admissionNumber = %v, want %v", got.AdmissionNumber, tt.admNumber)
				}
			}
		})
	}
}

func TestMemoryStore_PaymentOperations(t *testing.T) {
	ms := NewMemoryStore()
	paymentID := uuid.New()
	paymentHash := "test-hash-123"

	payment := &models.Payment{
		ID:                     paymentID,
		SchoolID:               1,
		StudentAdmissionNumber: "AHS-8899",
		StudentName:            "John Kiprop",
		ParentName:             "Jane Doe",
		AmountKES:              1000.0,
		AmountSats:             5000,
		LightningInvoice:       "lnbc1...",
		PaymentHash:            paymentHash,
		Status:                 models.StatusPending,
		CreatedAt:              time.Now(),
		UpdatedAt:              time.Now(),
	}

	// Test CreatePayment
	t.Run("CreatePayment", func(t *testing.T) {
		err := ms.CreatePayment(payment)
		if err != nil {
			t.Errorf("CreatePayment() error: %v", err)
		}
	})

	// Test GetPayment
	t.Run("GetPayment", func(t *testing.T) {
		got, err := ms.GetPayment(paymentID)
		if err != nil {
			t.Errorf("GetPayment() error: %v", err)
			return
		}
		if got.ID != paymentID {
			t.Errorf("GetPayment() got ID = %v, want %v", got.ID, paymentID)
		}
	})

	// Test GetPaymentByHash
	t.Run("GetPaymentByHash", func(t *testing.T) {
		got, err := ms.GetPaymentByHash(paymentHash)
		if err != nil {
			t.Errorf("GetPaymentByHash() error: %v", err)
			return
		}
		if got.PaymentHash != paymentHash {
			t.Errorf("GetPaymentByHash() got Hash = %v, want %v", got.PaymentHash, paymentHash)
		}
		if got.ID != paymentID {
			t.Errorf("GetPaymentByHash() got ID = %v, want %v", got.ID, paymentID)
		}
	})

	// Test UpdatePaymentStatus
	t.Run("UpdatePaymentStatus", func(t *testing.T) {
		newStatus := models.StatusPaid
		receipt := "MPESA12345"
		
		// Capture current UpdatedAt to compare later
		pBefore, _ := ms.GetPayment(paymentID)
		oldUpdatedAt := pBefore.UpdatedAt

		// Add a small sleep to ensure time change
		time.Sleep(10 * time.Millisecond)

		err := ms.UpdatePaymentStatus(paymentID, newStatus, receipt)
		if err != nil {
			t.Errorf("UpdatePaymentStatus() error: %v", err)
		}

		got, _ := ms.GetPayment(paymentID)
		if got.Status != newStatus {
			t.Errorf("UpdatePaymentStatus() status = %v, want %v", got.Status, newStatus)
		}
		if got.MPesaReceipt != receipt {
			t.Errorf("UpdatePaymentStatus() receipt = %v, want %v", got.MPesaReceipt, receipt)
		}
		if !got.UpdatedAt.After(oldUpdatedAt) {
			t.Errorf("UpdatePaymentStatus() UpdatedAt was not updated: got %v, want after %v", got.UpdatedAt, oldUpdatedAt)
		}
	})

	// Test UpdatePaymentStatus for unknown payment
	t.Run("UpdatePaymentStatus_Unknown", func(t *testing.T) {
		err := ms.UpdatePaymentStatus(uuid.New(), models.StatusPaid, "N/A")
		if err == nil {
			t.Error("UpdatePaymentStatus() expected error for unknown ID, got nil")
		}
	})
	
	// Test GetPayment for unknown payment
	t.Run("GetPayment_Unknown", func(t *testing.T) {
		_, err := ms.GetPayment(uuid.New())
		if err == nil {
			t.Error("GetPayment() expected error for unknown ID, got nil")
		}
	})

	// Test GetPaymentByHash for unknown hash
	t.Run("GetPaymentByHash_Unknown", func(t *testing.T) {
		_, err := ms.GetPaymentByHash("unknown-hash")
		if err == nil {
			t.Error("GetPaymentByHash() expected error for unknown hash, got nil")
		}
	})
}
