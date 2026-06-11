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
