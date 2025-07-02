package delivery

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"
	
	"github.com/tpdoyle87/simple-email-server/internal/config"
	"github.com/tpdoyle87/simple-email-server/pkg/email"
)

type mockQueue struct {
	mu           sync.Mutex
	emails       []*email.Email
	dequeueCount int
	delivered    map[string]bool
	failed       map[string]string
}

func newMockQueue() *mockQueue {
	return &mockQueue{
		emails:    make([]*email.Email, 0),
		delivered: make(map[string]bool),
		failed:    make(map[string]string),
	}
}

func (m *mockQueue) Enqueue(e *email.Email) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.emails = append(m.emails, e)
	return nil
}

func (m *mockQueue) Dequeue(count int) ([]*email.Email, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.dequeueCount++
	
	result := make([]*email.Email, 0)
	for i, e := range m.emails {
		if e.Status == email.StatusQueued && len(result) < count {
			e.Status = email.StatusSending
			result = append(result, m.emails[i])
		}
	}
	
	return result, nil
}

func (m *mockQueue) MarkDelivered(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.delivered[id] = true
	return nil
}

func (m *mockQueue) MarkFailed(id string, reason string, retry bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failed[id] = reason
	return nil
}

func (m *mockQueue) Size() int {
	return len(m.emails)
}

type mockDNSResolver struct {
	mx map[string][]*net.MX
}

func (m *mockDNSResolver) LookupMX(domain string) ([]*net.MX, error) {
	if mx, ok := m.mx[domain]; ok {
		return mx, nil
	}
	return nil, &net.DNSError{Err: "no such host", Name: domain}
}

type mockSMTPClient struct {
	mu        sync.Mutex
	sent      []*email.Email
	shouldErr bool
}

func (m *mockSMTPClient) Send(ctx context.Context, host string, e *email.Email) error {
	if m.shouldErr {
		return &net.OpError{Op: "dial", Err: &net.DNSError{Err: "connection refused"}}
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent = append(m.sent, e)
	return nil
}

func TestDeliveryService_Start(t *testing.T) {
	cfg := &config.DeliveryConfig{
		Workers:           2,
		DNSCacheTTL:       5 * time.Minute,
		ConnectionTimeout: 30 * time.Second,
	}
	
	queue := newMockQueue()
	resolver := &mockDNSResolver{
		mx: map[string][]*net.MX{
			"example.com": {{Host: "mail.example.com", Pref: 10}},
		},
	}
	
	service := NewService(cfg, queue)
	service.resolver = resolver
	service.client = &mockSMTPClient{}
	
	// Add test email
	testEmail := &email.Email{
		ID:      "test-1",
		From:    "sender@test.com",
		To:      []string{"recipient@example.com"},
		Subject: "Test",
		Body:    "Test body",
		Status:  email.StatusQueued,
	}
	queue.Enqueue(testEmail)
	
	// Start service
	ctx, cancel := context.WithCancel(context.Background())
	go service.Start(ctx)
	
	// Wait for processing (need more time for worker to pick up email)
	time.Sleep(1500 * time.Millisecond)
	
	// Stop service
	cancel()
	time.Sleep(100 * time.Millisecond)
	
	// Check if email was delivered
	if !queue.delivered["test-1"] {
		t.Error("Email should have been marked as delivered")
	}
}

func TestDeliveryService_ProcessEmail(t *testing.T) {
	cfg := &config.DeliveryConfig{
		Workers:           1,
		DNSCacheTTL:       5 * time.Minute,
		ConnectionTimeout: 30 * time.Second,
	}
	
	queue := newMockQueue()
	resolver := &mockDNSResolver{
		mx: map[string][]*net.MX{
			"example.com": {{Host: "mail.example.com", Pref: 10}},
		},
	}
	
	service := NewService(cfg, queue)
	service.resolver = resolver
	client := &mockSMTPClient{}
	service.client = client
	
	testEmail := &email.Email{
		ID:      "test-1",
		From:    "sender@test.com",
		To:      []string{"recipient@example.com"},
		Subject: "Test",
		Body:    "Test body",
	}
	
	err := service.processEmail(context.Background(), testEmail)
	if err != nil {
		t.Fatalf("Failed to process email: %v", err)
	}
	
	// Check if email was sent
	if len(client.sent) != 1 {
		t.Fatalf("Expected 1 email sent, got %d", len(client.sent))
	}
	
	if client.sent[0].ID != testEmail.ID {
		t.Errorf("Expected email ID %s, got %s", testEmail.ID, client.sent[0].ID)
	}
}

func TestDeliveryService_RetryOnFailure(t *testing.T) {
	cfg := &config.DeliveryConfig{
		Workers:           1,
		DNSCacheTTL:       5 * time.Minute,
		ConnectionTimeout: 30 * time.Second,
	}
	
	queue := newMockQueue()
	resolver := &mockDNSResolver{
		mx: map[string][]*net.MX{
			"example.com": {{Host: "mail.example.com", Pref: 10}},
		},
	}
	
	service := NewService(cfg, queue)
	service.resolver = resolver
	service.client = &mockSMTPClient{shouldErr: true}
	
	testEmail := &email.Email{
		ID:         "test-1",
		From:       "sender@test.com",
		To:         []string{"recipient@example.com"},
		Subject:    "Test",
		Body:       "Test body",
		RetryCount: 0,
	}
	
	err := service.processEmail(context.Background(), testEmail)
	if err == nil {
		t.Fatal("Expected error for failed delivery")
	}
	
	// processEmail doesn't mark as failed, the worker does
	// So we just check that an error was returned
}

func TestDeliveryService_DNSCache(t *testing.T) {
	cfg := &config.DeliveryConfig{
		Workers:           1,
		DNSCacheTTL:       100 * time.Millisecond,
		ConnectionTimeout: 30 * time.Second,
	}
	
	lookupCount := 0
	resolver := &mockDNSResolver{
		mx: map[string][]*net.MX{
			"example.com": {{Host: "mail.example.com", Pref: 10}},
		},
	}
	
	// Wrap resolver to count lookups
	countingResolver := &dnsResolver{
		lookupMX: func(domain string) ([]*net.MX, error) {
			lookupCount++
			return resolver.LookupMX(domain)
		},
	}
	
	service := NewService(cfg, newMockQueue())
	service.resolver = countingResolver
	
	// First lookup
	mx1, err := service.getMXRecords("example.com")
	if err != nil {
		t.Fatalf("Failed to get MX records: %v", err)
	}
	
	// Second lookup (should be cached)
	mx2, err := service.getMXRecords("example.com")
	if err != nil {
		t.Fatalf("Failed to get MX records: %v", err)
	}
	
	if lookupCount != 1 {
		t.Errorf("Expected 1 DNS lookup, got %d", lookupCount)
	}
	
	if len(mx1) != len(mx2) {
		t.Error("Cached MX records don't match")
	}
	
	// Wait for cache to expire
	time.Sleep(150 * time.Millisecond)
	
	// Third lookup (cache expired)
	_, err = service.getMXRecords("example.com")
	if err != nil {
		t.Fatalf("Failed to get MX records: %v", err)
	}
	
	if lookupCount != 2 {
		t.Errorf("Expected 2 DNS lookups after cache expiry, got %d", lookupCount)
	}
}