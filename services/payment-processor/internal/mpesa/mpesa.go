package mpesa

import (
	"bytes"
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log"
	mathrand "math/rand"
	"net/http"
	"os"
	"strings"
	"time"
)

// Service defines the interface for converting BTC and executing payouts via M-Pesa
type Service interface {
	// ExecuteOffRamp converts satoshis to KES currency
	ExecuteOffRamp(amountSats int64, targetRate float64) (float64, error)
	
	// PayoutSchoolFees sends M-Pesa B2C school fee payments
	PayoutSchoolFees(paybill string, accountNumber string, amountKES float64) (receiptNumber string, err error)
}

// MockMpesaService is a stub implementation that simulates successful payouts
type MockMpesaService struct{}
