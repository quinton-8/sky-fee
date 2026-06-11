package db

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/sky-fee/payment-processor/internal/models"
)

// Store defines the interface for database operations
type Store interface {
	GetSchools() ([]models.School, error)
	GetSchool(id int) (*models.School, error)
	GetStudent(schoolID int, admissionNumber string) (*models.Student, error)
	CreatePayment(payment *models.Payment) error
	GetPayment(id uuid.UUID) (*models.Payment, error)
	GetPaymentByHash(paymentHash string) (*models.Payment, error)
	UpdatePaymentStatus(id uuid.UUID, status models.PaymentStatus, mpesaReceipt string) error
}

// PostgresStore implements Store using database/sql and PostgreSQL
type PostgresStore struct {
	db *sql.DB
}

// MemoryStore implements Store in-memory (useful for quick local development/testing)
type MemoryStore struct {
	mu       sync.RWMutex
	schools  map[int]models.School
	students map[string]models.Student // Key format: schoolId_admissionNumber
	payments map[uuid.UUID]models.Payment
}

// NewStore creates a connection to the database or falls back to MemoryStore if connection fails
func NewStore(connStr string) (Store, error) {
	if connStr == "" {
		log.Println("⚡ No DB connection string provided, using in-memory database store")
		return NewMemoryStore(), nil
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Printf("⚠️ Error opening database connection: %v. Falling back to in-memory store.\n", err)
		return NewMemoryStore(), nil
	}

	// Try pinging the database with a short timeout
	db.SetConnMaxLifetime(time.Second * 3)
	err = db.Ping()
	if err != nil {
		log.Printf("⚠️ Database ping failed: %v. Falling back to in-memory store.\n", err)
		return NewMemoryStore(), nil
	}

	log.Println("🎉 Successfully connected to PostgreSQL database")
	return &PostgresStore{db: db}, nil
}

// ==========================================
// PostgresStore Implementation
// ==========================================

func (ps *PostgresStore) GetSchools() ([]models.School, error) {
	query := `SELECT id, name, paybill, account_number, created_at FROM schools ORDER BY name ASC`
	rows, err := ps.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schools []models.School
	for rows.Next() {
		var s models.School
		if err := rows.Scan(&s.ID, &s.Name, &s.Paybill, &s.AccountNumber, &s.CreatedAt); err != nil {
			return nil, err
		}
		schools = append(schools, s)
	}
	return schools, nil
}

func (ps *PostgresStore) GetSchool(id int) (*models.School, error) {
	query := `SELECT id, name, paybill, account_number, created_at FROM schools WHERE id = $1`
	var s models.School
	err := ps.db.QueryRow(query, id).Scan(&s.ID, &s.Name, &s.Paybill, &s.AccountNumber, &s.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, errors.New("school not found")
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (ps *PostgresStore) GetStudent(schoolID int, admissionNumber string) (*models.Student, error) {
	query := `SELECT id, school_id, admission_number, name, grade, created_at FROM students WHERE school_id = $1 AND admission_number = $2`
	var s models.Student
	err := ps.db.QueryRow(query, schoolID, admissionNumber).Scan(&s.ID, &s.SchoolID, &s.AdmissionNumber, &s.Name, &s.Grade, &s.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, errors.New("student not found")
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (ps *PostgresStore) CreatePayment(payment *models.Payment) error {
	query := `INSERT INTO payments (id, school_id, student_admission_number, student_name, parent_name, amount_kes, amount_sats, lightning_invoice, payment_hash, status, created_at, updated_at) 
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`
	_, err := ps.db.Exec(query, payment.ID, payment.SchoolID, payment.StudentAdmissionNumber, payment.StudentName, payment.ParentName, payment.AmountKES, payment.AmountSats, payment.LightningInvoice, payment.PaymentHash, payment.Status, payment.CreatedAt, payment.UpdatedAt)
	return err
}

func (ps *PostgresStore) GetPayment(id uuid.UUID) (*models.Payment, error) {
	query := `SELECT id, school_id, student_admission_number, student_name, parent_name, amount_kes, amount_sats, lightning_invoice, payment_hash, status, COALESCE(mpesa_receipt, ''), created_at, updated_at FROM payments WHERE id = $1`
	var p models.Payment
	err := ps.db.QueryRow(query, id).Scan(&p.ID, &p.SchoolID, &p.StudentAdmissionNumber, &p.StudentName, &p.ParentName, &p.AmountKES, &p.AmountSats, &p.LightningInvoice, &p.PaymentHash, &p.Status, &p.MPesaReceipt, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, errors.New("payment not found")
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (ps *PostgresStore) GetPaymentByHash(paymentHash string) (*models.Payment, error) {
	query := `SELECT id, school_id, student_admission_number, student_name, parent_name, amount_kes, amount_sats, lightning_invoice, payment_hash, status, COALESCE(mpesa_receipt, ''), created_at, updated_at FROM payments WHERE payment_hash = $1`
	var p models.Payment
	err := ps.db.QueryRow(query, paymentHash).Scan(&p.ID, &p.SchoolID, &p.StudentAdmissionNumber, &p.StudentName, &p.ParentName, &p.AmountKES, &p.AmountSats, &p.LightningInvoice, &p.PaymentHash, &p.Status, &p.MPesaReceipt, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, errors.New("payment not found")
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (ps *PostgresStore) UpdatePaymentStatus(id uuid.UUID, status models.PaymentStatus, mpesaReceipt string) error {
	query := `UPDATE payments SET status = $1, mpesa_receipt = $2, updated_at = $3 WHERE id = $4`
	_, err := ps.db.Exec(query, status, mpesaReceipt, time.Now(), id)
	return err
}

// ==========================================
// MemoryStore Implementation (Mock Fallback)
// ==========================================

func NewMemoryStore() *MemoryStore {
	ms := &MemoryStore{
		schools:  make(map[int]models.School),
		students: make(map[string]models.Student),
		payments: make(map[uuid.UUID]models.Payment),
	}

	// Seed Memory Store with same sample data
	ms.schools[1] = models.School{ID: 1, Name: "Alliance High School", Paybill: "222111", AccountNumber: "SchoolFees", CreatedAt: time.Now()}
	ms.schools[2] = models.School{ID: 2, Name: "Kenya High School", Paybill: "333222", AccountNumber: "SchoolFees", CreatedAt: time.Now()}
	ms.schools[3] = models.School{ID: 3, Name: "Lenana School", Paybill: "444333", AccountNumber: "SchoolFees", CreatedAt: time.Now()}

	ms.students["1_AHS-8899"] = models.Student{ID: 1, SchoolID: 1, AdmissionNumber: "AHS-8899", Name: "John Kiprop", Grade: "Form 3 Green", CreatedAt: time.Now()}
	ms.students["1_AHS-9012"] = models.Student{ID: 2, SchoolID: 1, AdmissionNumber: "AHS-9012", Name: "David Mwangi", Grade: "Form 1 Blue", CreatedAt: time.Now()}
	ms.students["2_KHS-4455"] = models.Student{ID: 3, SchoolID: 2, AdmissionNumber: "KHS-4455", Name: "Sarah Cherono", Grade: "Form 4 East", CreatedAt: time.Now()}
	ms.students["3_LEN-1234"] = models.Student{ID: 4, SchoolID: 3, AdmissionNumber: "LEN-1234", Name: "Joseph Kamau", Grade: "Form 2 West", CreatedAt: time.Now()}

	return ms
}

func (ms *MemoryStore) GetSchools() ([]models.School, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	var list []models.School
	for _, s := range ms.schools {
		list = append(list, s)
	}
	return list, nil
}

func (ms *MemoryStore) GetSchool(id int) (*models.School, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	s, ok := ms.schools[id]
	if !ok {
		return nil, errors.New("school not found")
	}
	return &s, nil
}

func (ms *MemoryStore) GetStudent(schoolID int, admissionNumber string) (*models.Student, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	key := fmt.Sprintf("%d_%s", schoolID, admissionNumber)
	s, ok := ms.students[key]
	if !ok {
		return nil, errors.New("student not found")
	}
	return &s, nil
}

func (ms *MemoryStore) CreatePayment(payment *models.Payment) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.payments[payment.ID] = *payment
	return nil
}

func (ms *MemoryStore) GetPayment(id uuid.UUID) (*models.Payment, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	p, ok := ms.payments[id]
	if !ok {
		return nil, errors.New("payment not found")
	}
	return &p, nil
}

func (ms *MemoryStore) GetPaymentByHash(paymentHash string) (*models.Payment, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	for _, p := range ms.payments {
		if p.PaymentHash == paymentHash {
			return &p, nil
		}
	}
	return nil, errors.New("payment not found")
}

func (ms *MemoryStore) UpdatePaymentStatus(id uuid.UUID, status models.PaymentStatus, mpesaReceipt string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	p, ok := ms.payments[id]
	if !ok {
		return errors.New("payment not found")
	}
	p.Status = status
	p.MPesaReceipt = mpesaReceipt
	p.UpdatedAt = time.Now()
	ms.payments[id] = p
	return nil
}
