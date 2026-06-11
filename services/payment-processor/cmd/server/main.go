package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/sky-fee/payment-processor/internal/db"
	"github.com/sky-fee/payment-processor/internal/handlers"
	"github.com/sky-fee/payment-processor/internal/lightning"
	"github.com/sky-fee/payment-processor/internal/mpesa"
)

func main() {
	log.Println("🚀 Starting SkyFee Payment Processor Server...")

	// 1. Load Configurations from Environment Variables
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbURL := os.Getenv("DATABASE_URL")
	lnbitsURL := os.Getenv("LNBITS_URL")
	lnbitsKey := os.Getenv("LNBITS_INVOICE_KEY")

	// 2. Initialize Core Modules
	// Database (with PostgreSQL/Memory automatic fallback)
	store, err := db.NewStore(dbURL)
	if err != nil {
		log.Fatalf("❌ Database connection error: %v", err)
	}

	// Lightning client (with LNbits/Mock fallback)
	lnClient := lightning.NewClient(lnbitsURL, lnbitsKey)

	// M-Pesa & Offramp Payout Service (Mocks for Job 3)
	mpesaService := mpesa.NewService()

	// 3. Initialize Server & Routers
	server := handlers.NewServer(store, lnClient, mpesaService)
	
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Register API endpoints
	server.RegisterRoutes(r)

	// 4. Start HTTP Listener
	log.Printf("📡 SkyFee API Backend listening on port %s\n", port)
	err = http.ListenAndServe(":"+port, r)
	if err != nil {
		log.Fatalf("❌ Server failed to start: %v", err)
	}
}
