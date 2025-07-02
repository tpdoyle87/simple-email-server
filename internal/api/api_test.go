package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	
	"github.com/tpdoyle87/simple-email-server/internal/config"
	"github.com/tpdoyle87/simple-email-server/pkg/email"
)

type mockQueue struct {
	emails   []*email.Email
	failNext bool
}

func (m *mockQueue) Enqueue(e *email.Email) error {
	if m.failNext {
		return ErrQueueFull
	}
	m.emails = append(m.emails, e)
	return nil
}

func (m *mockQueue) Dequeue(count int) ([]*email.Email, error) {
	return nil, nil
}

func (m *mockQueue) MarkDelivered(id string) error {
	return nil
}

func (m *mockQueue) MarkFailed(id string, reason string, retry bool) error {
	return nil
}

func (m *mockQueue) Size() int {
	return len(m.emails)
}

func TestAPI_SendEmail(t *testing.T) {
	cfg := &config.APIConfig{
		AuthToken: "test-token",
	}
	
	queue := &mockQueue{}
	api := New(cfg, queue, 25*1024*1024)
	
	tests := []struct {
		name       string
		token      string
		payload    interface{}
		wantStatus int
	}{
		{
			name:  "valid email",
			token: "test-token",
			payload: SendEmailRequest{
				From:    "sender@example.com",
				To:      []string{"recipient@example.com"},
				Subject: "Test",
				Body:    "Test body",
			},
			wantStatus: http.StatusAccepted,
		},
		{
			name:       "missing auth token",
			token:      "",
			payload:    SendEmailRequest{},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:  "invalid auth token",
			token: "wrong-token",
			payload: SendEmailRequest{
				From: "sender@example.com",
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:  "invalid email",
			token: "test-token",
			payload: SendEmailRequest{
				From: "invalid-email",
				To:   []string{"recipient@example.com"},
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:  "missing recipients",
			token: "test-token",
			payload: SendEmailRequest{
				From:    "sender@example.com",
				Subject: "Test",
				Body:    "Test body",
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid JSON",
			token:      "test-token",
			payload:    "invalid json",
			wantStatus: http.StatusBadRequest,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest("POST", "/send", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}
			
			w := httptest.NewRecorder()
			api.ServeHTTP(w, req)
			
			if w.Code != tt.wantStatus {
				t.Errorf("Expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

func TestAPI_GetStatus(t *testing.T) {
	cfg := &config.APIConfig{
		AuthToken: "test-token",
	}
	
	queue := &mockQueue{}
	api := New(cfg, queue, 25*1024*1024)
	
	// Add test email to tracking
	testEmail := &email.Email{
		ID:     "test-123",
		Status: email.StatusDelivered,
	}
	api.emailStatus.Store("test-123", testEmail)
	
	tests := []struct {
		name       string
		emailID    string
		token      string
		wantStatus int
	}{
		{
			name:       "valid status request",
			emailID:    "test-123",
			token:      "test-token",
			wantStatus: http.StatusOK,
		},
		{
			name:       "missing auth",
			emailID:    "test-123",
			token:      "",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "email not found",
			emailID:    "unknown-id",
			token:      "test-token",
			wantStatus: http.StatusNotFound,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/status/"+tt.emailID, nil)
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}
			
			w := httptest.NewRecorder()
			api.ServeHTTP(w, req)
			
			if w.Code != tt.wantStatus {
				t.Errorf("Expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

func TestAPI_GetStats(t *testing.T) {
	cfg := &config.APIConfig{
		AuthToken: "test-token",
	}
	
	queue := &mockQueue{}
	api := New(cfg, queue, 25*1024*1024)
	
	req := httptest.NewRequest("GET", "/stats", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	
	var stats StatsResponse
	if err := json.NewDecoder(w.Body).Decode(&stats); err != nil {
		t.Fatalf("Failed to decode stats response: %v", err)
	}
	
	if stats.QueueSize != 0 {
		t.Errorf("Expected queue size 0, got %d", stats.QueueSize)
	}
}

func TestAPI_HealthCheck(t *testing.T) {
	cfg := &config.APIConfig{
		AuthToken: "test-token",
	}
	
	queue := &mockQueue{}
	api := New(cfg, queue, 25*1024*1024)
	
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	
	var health HealthResponse
	if err := json.NewDecoder(w.Body).Decode(&health); err != nil {
		t.Fatalf("Failed to decode health response: %v", err)
	}
	
	if health.Status != "healthy" {
		t.Errorf("Expected status 'healthy', got '%s'", health.Status)
	}
}