package mpesa

import (
	"testing"
)

func TestMockMpesaService_ExecuteOffRamp(t *testing.T) {
	svc := &MockMpesaService{}
	
	tests := []struct {
		name       string
		amountSats int64
		rate       float64
		wantKES    float64
	}{
		{"1 BTC at 1,000,000 KES", 100000000, 1000000.0, 1000000.0},
		{"0.5 BTC at 2,000,000 KES", 50000000, 2000000.0, 1000000.0},
		{"10,000 Sats at 9,000,000 KES", 10000, 9000000.0, 900.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := svc.ExecuteOffRamp(tt.amountSats, tt.rate)
			if err != nil {
				t.Fatalf("ExecuteOffRamp failed: %v", err)
			}
			if got != tt.wantKES {
				t.Errorf("ExecuteOffRamp() = %v, want %v", got, tt.wantKES)
			}
		})
	}
}

func TestMockMpesaService_PayoutSchoolFees(t *testing.T) {
	svc := &MockMpesaService{}
	
	receipt, err := svc.PayoutSchoolFees("222111", "ACC123", 1500.50)
	if err != nil {
		t.Fatalf("PayoutSchoolFees failed: %v", err)
	}

	if len(receipt) != 10 {
		t.Errorf("Expected receipt length 10, got %d", len(receipt))
	}

	if receipt == "" {
		t.Error("Expected non-empty receipt")
	}
}
