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
