package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	
	"github.com/google/uuid"
	"github.com/tpdoyle87/simple-email-server/internal/config"
	"github.com/tpdoyle87/simple-email-server/internal/queue"
	"github.com/tpdoyle87/simple-email-server/pkg/email"
)

var (
	ErrQueueFull      = errors.New("queue is full")
	ErrUnauthorized   = errors.New("unauthorized")
	ErrInvalidRequest = errors.New("invalid request")
	ErrNotFound       = errors.New("not found")
)

type API struct {
	config         *config.APIConfig
	queue          queue.Queue
	maxMessageSize int64
	
	// Stats
	totalSent      atomic.Int64
	totalFailed    atomic.Int64
	totalDelivered atomic.Int64
	
	// Email status tracking
	emailStatus sync.Map // map[string]*email.Email
	
	mux *http.ServeMux
}

type SendEmailRequest struct {
	From        string            `json:"from"`
	To          []string          `json:"to"`
	CC          []string          `json:"cc,omitempty"`
	BCC         []string          `json:"bcc,omitempty"`
	Subject     string            `json:"subject"`
	Body        string            `json:"body"`
	HTML        string            `json:"html,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	ScheduledAt *time.Time        `json:"scheduled_at,omitempty"`
}

type SendEmailResponse struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type StatusResponse struct {
	ID          string     `json:"id"`
	Status      string     `json:"status"`
	RetryCount  int        `json:"retry_count"`
	LastError   string     `json:"last_error,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeliveredAt *time.Time `json:"delivered_at,omitempty"`
}

type StatsResponse struct {
	QueueSize      int   `json:"queue_size"`
	TotalSent      int64 `json:"total_sent"`
	TotalDelivered int64 `json:"total_delivered"`
	TotalFailed    int64 `json:"total_failed"`
}

type HealthResponse struct {
	Status    string `json:"status"`
	QueueSize int    `json:"queue_size"`
	Uptime    string `json:"uptime"`
}

func New(cfg *config.APIConfig, q queue.Queue, maxMessageSize int64) *API {
	api := &API{
		config:         cfg,
		queue:          q,
		maxMessageSize: maxMessageSize,
		mux:            http.NewServeMux(),
	}
	
	// Register routes
	api.mux.HandleFunc("/send", api.authenticate(api.handleSendEmail))
	api.mux.HandleFunc("/send/batch", api.authenticate(api.handleSendBatch))
	api.mux.HandleFunc("/status/", api.authenticate(api.handleGetStatus))
	api.mux.HandleFunc("/stats", api.authenticate(api.handleGetStats))
	api.mux.HandleFunc("/health", api.handleHealthCheck)
	
	return api
}

func (a *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.mux.ServeHTTP(w, r)
}

func (a *API) authenticate(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			a.errorResponse(w, http.StatusUnauthorized, "missing authorization header")
			return
		}
		
		parts := strings.Split(auth, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			a.errorResponse(w, http.StatusUnauthorized, "invalid authorization format")
			return
		}
		
		if parts[1] != a.config.AuthToken {
			a.errorResponse(w, http.StatusUnauthorized, "invalid token")
			return
		}
		
		handler(w, r)
	}
}

func (a *API) handleSendEmail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		a.errorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	
	var req SendEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		a.errorResponse(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	
	// Create email
	e := &email.Email{
		ID:          uuid.New().String(),
		From:        req.From,
		To:          req.To,
		CC:          req.CC,
		BCC:         req.BCC,
		Subject:     req.Subject,
		Body:        req.Body,
		HTML:        req.HTML,
		Headers:     req.Headers,
		Status:      email.StatusQueued,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		ScheduledAt: req.ScheduledAt,
	}
	
	// Validate
	if err := e.Validate(a.maxMessageSize); err != nil {
		a.errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	
	// Enqueue
	if err := a.queue.Enqueue(e); err != nil {
		if err == queue.ErrQueueFull {
			a.errorResponse(w, http.StatusServiceUnavailable, "queue is full")
			return
		}
		a.errorResponse(w, http.StatusInternalServerError, "failed to queue email")
		return
	}
	
	// Track email
	a.emailStatus.Store(e.ID, e)
	a.totalSent.Add(1)
	
	// Response
	resp := SendEmailResponse{
		ID:      e.ID,
		Status:  string(e.Status),
		Message: "Email queued for delivery",
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(resp)
}

func (a *API) handleSendBatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		a.errorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	
	var requests []SendEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&requests); err != nil {
		a.errorResponse(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	
	if len(requests) > 100 {
		a.errorResponse(w, http.StatusBadRequest, "batch size exceeds limit (100)")
		return
	}
	
	responses := make([]SendEmailResponse, 0, len(requests))
	
	for _, req := range requests {
		e := &email.Email{
			ID:          uuid.New().String(),
			From:        req.From,
			To:          req.To,
			CC:          req.CC,
			BCC:         req.BCC,
			Subject:     req.Subject,
			Body:        req.Body,
			HTML:        req.HTML,
			Headers:     req.Headers,
			Status:      email.StatusQueued,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			ScheduledAt: req.ScheduledAt,
		}
		
		// Validate
		if err := e.Validate(a.maxMessageSize); err != nil {
			responses = append(responses, SendEmailResponse{
				ID:      "",
				Status:  "error",
				Message: err.Error(),
			})
			continue
		}
		
		// Enqueue
		if err := a.queue.Enqueue(e); err != nil {
			responses = append(responses, SendEmailResponse{
				ID:      "",
				Status:  "error",
				Message: "failed to queue",
			})
			continue
		}
		
		// Track email
		a.emailStatus.Store(e.ID, e)
		a.totalSent.Add(1)
		
		responses = append(responses, SendEmailResponse{
			ID:      e.ID,
			Status:  string(e.Status),
			Message: "Email queued for delivery",
		})
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(responses)
}

func (a *API) handleGetStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		a.errorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	
	// Extract email ID from path
	path := strings.TrimPrefix(r.URL.Path, "/status/")
	if path == "" {
		a.errorResponse(w, http.StatusBadRequest, "missing email ID")
		return
	}
	
	// Look up email
	value, ok := a.emailStatus.Load(path)
	if !ok {
		a.errorResponse(w, http.StatusNotFound, "email not found")
		return
	}
	
	e := value.(*email.Email)
	
	resp := StatusResponse{
		ID:          e.ID,
		Status:      string(e.Status),
		RetryCount:  e.RetryCount,
		LastError:   e.LastError,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
		DeliveredAt: e.DeliveredAt,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (a *API) handleGetStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		a.errorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	
	resp := StatsResponse{
		QueueSize:      a.queue.Size(),
		TotalSent:      a.totalSent.Load(),
		TotalDelivered: a.totalDelivered.Load(),
		TotalFailed:    a.totalFailed.Load(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (a *API) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		a.errorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	
	resp := HealthResponse{
		Status:    "healthy",
		QueueSize: a.queue.Size(),
		Uptime:    "0s", // TODO: Track actual uptime
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (a *API) errorResponse(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}

func (a *API) Start() error {
	log.Printf("Starting API server on %s", a.config.ListenAddress)
	return http.ListenAndServe(a.config.ListenAddress, a)
}