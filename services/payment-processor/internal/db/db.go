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
