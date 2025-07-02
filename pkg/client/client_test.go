package client

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_Send(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check auth
		if r.Header.Get("Authorization") != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		
		// Check method and path
		if r.Method != "POST" || r.URL.Path != "/send" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		
		// Return success response
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"id":"test-123","status":"queued","message":"Email queued for delivery"}`))
	}))
	defer server.Close()
	
	// Create client
	client := New(server.URL, "test-token")
	
	// Send email
	email := &Email{
		From:    "sender@example.com",
		To:      []string{"recipient@example.com"},
		Subject: "Test",
		Body:    "Test body",
	}
	
	resp, err := client.Send(email)
	if err != nil {
		t.Fatalf("Failed to send email: %v", err)
	}
	
	if resp.ID != "test-123" {
		t.Errorf("Expected ID test-123, got %s", resp.ID)
	}
	
	if resp.Status != "queued" {
		t.Errorf("Expected status queued, got %s", resp.Status)
	}
}

func TestClient_GetStatus(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check auth
		if r.Header.Get("Authorization") != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		
		// Check method and path
		if r.Method != "GET" || r.URL.Path != "/status/test-123" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		
		// Return status response
		w.Write([]byte(`{
			"id":"test-123",
			"status":"delivered",
			"retry_count":0,
			"created_at":"2023-01-01T00:00:00Z",
			"updated_at":"2023-01-01T00:01:00Z",
			"delivered_at":"2023-01-01T00:01:00Z"
		}`))
	}))
	defer server.Close()
	
	// Create client
	client := New(server.URL, "test-token")
	
	// Get status
	status, err := client.GetStatus("test-123")
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}
	
	if status.ID != "test-123" {
		t.Errorf("Expected ID test-123, got %s", status.ID)
	}
	
	if status.Status != "delivered" {
		t.Errorf("Expected status delivered, got %s", status.Status)
	}
}

func TestClient_SendBatch(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check auth
		if r.Header.Get("Authorization") != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		
		// Check method and path
		if r.Method != "POST" || r.URL.Path != "/send/batch" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		
		// Return batch response
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`[
			{"id":"test-1","status":"queued","message":"Email queued for delivery"},
			{"id":"test-2","status":"queued","message":"Email queued for delivery"}
		]`))
	}))
	defer server.Close()
	
	// Create client
	client := New(server.URL, "test-token")
	
	// Send batch
	emails := []*Email{
		{
			From:    "sender@example.com",
			To:      []string{"recipient1@example.com"},
			Subject: "Test 1",
			Body:    "Test body 1",
		},
		{
			From:    "sender@example.com",
			To:      []string{"recipient2@example.com"},
			Subject: "Test 2",
			Body:    "Test body 2",
		},
	}
	
	responses, err := client.SendBatch(emails)
	if err != nil {
		t.Fatalf("Failed to send batch: %v", err)
	}
	
	if len(responses) != 2 {
		t.Fatalf("Expected 2 responses, got %d", len(responses))
	}
	
	if responses[0].ID != "test-1" {
		t.Errorf("Expected first ID test-1, got %s", responses[0].ID)
	}
	
	if responses[1].ID != "test-2" {
		t.Errorf("Expected second ID test-2, got %s", responses[1].ID)
	}
}