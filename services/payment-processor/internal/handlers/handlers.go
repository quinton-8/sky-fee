package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

)

// Server holds the dependencies for API routing
type Server struct {
	Store     db.Store
	Lightning lightning.Client
	MPesa     mpesa.Service
	wsManager *WSManager
}

// WSManager tracks connected WebSocket clients interested in specific payment status updates
type WSManager struct {
	sync.Mutex
	clients map[string][]*websocket.Conn
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow cross-origin requests for Next.js web application
	},
}

// NewServer configures a new Server instance
func NewServer(store db.Store, lightningClient lightning.Client, mpesaService mpesa.Service) *Server {
	return &Server{
		Store:     store,
		Lightning: lightningClient,
		MPesa:     mpesaService,
		wsManager: &WSManager{clients: make(map[string][]*websocket.Conn)},
	}
}

// Helper: respond with error JSON
func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

// Helper: respond with standard JSON
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error encoding JSON response"))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

// ==========================================
// Handlers Implementation
// ==========================================

// GetSchools returns a list of all registered schools
func (s *Server) GetSchools(w http.ResponseWriter, r *http.Request) {
	schools, err := s.Store.GetSchools()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to retrieve schools: "+err.Error())
		return
	}
	respondWithJSON(w, http.StatusOK, schools)
}

// VerifyStudent validates student details against enrolled schools
func (s *Server) VerifyStudent(w http.ResponseWriter, r *http.Request) {
	schoolIDStr := chi.URLParam(r, "schoolID")
	admissionNumber := chi.URLParam(r, "admissionNumber")

	schoolID, err := strconv.Atoi(schoolIDStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid school ID")
		return
	}

	student, err := s.Store.GetStudent(schoolID, admissionNumber)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Student verification failed: "+err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, student)
}

type createPaymentRequest struct {
	SchoolID               int     `json:"school_id"`
	StudentAdmissionNumber string  `json:"student_admission_number"`
	ParentName             string  `json:"parent_name"`
	AmountKES              float64 `json:"amount_kes"`
}

type createPaymentResponse struct {
	PaymentID        uuid.UUID `json:"payment_id"`
	AmountKES        float64   `json:"amount_kes"`
	AmountSats       int64     `json:"amount_sats"`
	LightningInvoice string    `json:"lightning_invoice"`
	PaymentHash      string    `json:"payment_hash"`
	BTCKESRate       float64   `json:"btc_kes_rate"`
}

// CreatePayment generates a Lightning invoice based on requested Kenyan Shilling fees
func (s *Server) CreatePayment(w http.ResponseWriter, r *http.Request) {
	var req createPaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid JSON input")
		return
	}

	if req.SchoolID <= 0 || req.StudentAdmissionNumber == "" || req.ParentName == "" || req.AmountKES <= 0 {
		respondWithError(w, http.StatusBadRequest, "Missing required payment fields")
		return
	}

	// 1. Validate School exists
	school, err := s.Store.GetSchool(req.SchoolID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "School not enrolled on our platform")
		return
	}

	// 2. Validate Student is registered in that school
	student, err := s.Store.GetStudent(req.SchoolID, req.StudentAdmissionNumber)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Student not found in the selected school")
		return
	}

	// 3. Fetch current exchange rate (BTC/KES)
	rate, err := s.Lightning.GetBTCKESRate()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to fetch exchange rates: "+err.Error())
		return
	}

	// 4. Convert KES to Satoshis
	// amountSats = (amountKES / btc_kes_rate) * 100,000,000
	btcAmount := req.AmountKES / rate
	amountSats := int64(btcAmount * 100000000.0)
	if amountSats <= 0 {
		respondWithError(w, http.StatusBadRequest, "Amount requested translates to less than 1 Satoshi")
		return
	}

	// 5. Generate Lightning Invoice
	memo := fmt.Sprintf("SkyFee: School fees for %s (Adm: %s) at %s", student.Name, student.AdmissionNumber, school.Name)
	paymentHash, invoice, err := s.Lightning.CreateInvoice(amountSats, memo)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to generate Lightning invoice: "+err.Error())
		return
	}

	// 6. Save Payment into the Database as PENDING
	paymentID := uuid.New()
	payment := &models.Payment{
		ID:                     paymentID,
		SchoolID:               req.SchoolID,
		StudentAdmissionNumber: req.StudentAdmissionNumber,
		StudentName:            student.Name,
		ParentName:             req.ParentName,
		AmountKES:              req.AmountKES,
		AmountSats:             amountSats,
		LightningInvoice:       invoice,
		PaymentHash:            paymentHash,
		Status:                 models.StatusPending,
		CreatedAt:              time.Now(),
		UpdatedAt:              time.Now(),
	}

	if err := s.Store.CreatePayment(payment); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to log payment transaction: "+err.Error())
		return
	}

	respondWithJSON(w, http.StatusCreated, createPaymentResponse{
		PaymentID:        paymentID,
		AmountKES:        req.AmountKES,
		AmountSats:       amountSats,
		LightningInvoice: invoice,
		PaymentHash:      paymentHash,
		BTCKESRate:       rate,
	})
}

// GetPayment fetches status details of a specific payment
func (s *Server) GetPayment(w http.ResponseWriter, r *http.Request) {
	paymentIDStr := chi.URLParam(r, "paymentID")
	paymentID, err := uuid.Parse(paymentIDStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid payment ID format")
		return
	}

	payment, err := s.Store.GetPayment(paymentID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Payment not found: "+err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, payment)
}

// LightningWebhook receives payment confirmation hooks from lightning nodes (LNbits webhook format)
func (s *Server) LightningWebhook(w http.ResponseWriter, r *http.Request) {
	var body struct {
		PaymentHash string `json:"payment_hash"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if body.PaymentHash == "" {
		respondWithError(w, http.StatusBadRequest, "payment_hash is required")
		return
	}

	err := s.ProcessInvoiceSettlement(body.PaymentHash)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Settlement processing failed: "+err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// TriggerMockPayment allows manual checkout trigger without running real lightning nodes (for development convenience)
func (s *Server) TriggerMockPayment(w http.ResponseWriter, r *http.Request) {
	paymentIDStr := chi.URLParam(r, "paymentID")
	paymentID, err := uuid.Parse(paymentIDStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid payment ID")
		return
	}

	payment, err := s.Store.GetPayment(paymentID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Payment transaction not found")
		return
	}

	if payment.Status != models.StatusPending {
		respondWithError(w, http.StatusBadRequest, "Payment is already processed")
		return
	}

	err = s.ProcessInvoiceSettlement(payment.PaymentHash)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to settle payment: "+err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"message": "Mock payment successfully processed"})
}

// ProcessInvoiceSettlement transitions invoice state, executes off-ramp, and triggers M-Pesa payouts
func (s *Server) ProcessInvoiceSettlement(paymentHash string) error {
	// 1. Fetch Payment details
	payment, err := s.Store.GetPaymentByHash(paymentHash)
	if err != nil {
		return err
	}

	// Double check to ensure we aren't double-processing payments
	if payment.Status != models.StatusPending {
		log.Printf("⚠️ Payment %s already processed. Status: %s\n", payment.ID, payment.Status)
		return nil
	}

	log.Printf("⚡ [Settlement] Invoice paid! Hash: %s. Initiating payout sequence...\n", paymentHash)

	// 2. Mark payment as PAID in database
	payment.Status = models.StatusPaid
	s.Store.UpdatePaymentStatus(payment.ID, models.StatusPaid, "")
	s.wsManager.Notify(payment.ID.String(), payment)

	// 3. Resolve School details to fetch their Paybill/Till configurations
	school, err := s.Store.GetSchool(payment.SchoolID)
	if err != nil {
		payment.Status = models.StatusFailed
		s.Store.UpdatePaymentStatus(payment.ID, models.StatusFailed, "")
		s.wsManager.Notify(payment.ID.String(), payment)
		return fmt.Errorf("failed to fetch school details during payout: %v", err)
	}

	// 4. Execute Off-Ramp (BTC -> KES)
	// Fetch live rate or fallback to cache rate
	rate, rateErr := s.Lightning.GetBTCKESRate()
	if rateErr != nil {
		// fallback to a reasonable static rate if service rate call goes down at payment instant
		rate = 9000000.0 
	}
	
	amountKES, err := s.MPesa.ExecuteOffRamp(payment.AmountSats, rate)
	if err != nil {
		log.Printf("❌ Off-ramp conversion failed for payment %s: %v\n", payment.ID, err)
		payment.Status = models.StatusFailed
		s.Store.UpdatePaymentStatus(payment.ID, models.StatusFailed, "")
		s.wsManager.Notify(payment.ID.String(), payment)
		return err
	}

	// 5. Execute M-Pesa paybill transaction to the school
	// Account name contains student registration info
	accountIndicator := fmt.Sprintf("Adm:%s", payment.StudentAdmissionNumber)
	receiptNumber, err := s.MPesa.PayoutSchoolFees(school.Paybill, accountIndicator, amountKES)
	if err != nil {
		log.Printf("❌ M-Pesa B2C payout failed for payment %s: %v\n", payment.ID, err)
		payment.Status = models.StatusFailed
		s.Store.UpdatePaymentStatus(payment.ID, models.StatusFailed, "")
		s.wsManager.Notify(payment.ID.String(), payment)
		return err
	}

	// 6. Complete payout updates
	payment.Status = models.StatusDisbursed
	payment.MPesaReceipt = receiptNumber
	err = s.Store.UpdatePaymentStatus(payment.ID, models.StatusDisbursed, receiptNumber)
	if err != nil {
		log.Printf("⚠️ Payment completed but failed updating database status: %v\n", err)
	}

	log.Printf("🎉 [Success] Payment %s fully disbursed! M-Pesa Receipt: %s\n", payment.ID, receiptNumber)
	s.wsManager.Notify(payment.ID.String(), payment)
	return nil
}
