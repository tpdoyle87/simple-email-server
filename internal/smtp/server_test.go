package smtp

import (
	"net"
	"net/smtp"
	"strings"
	"testing"
	"time"
	
	"github.com/tpdoyle87/simple-email-server/internal/config"
	"github.com/tpdoyle87/simple-email-server/pkg/email"
)

type mockQueue struct {
	emails []*email.Email
}

func (m *mockQueue) Enqueue(e *email.Email) error {
	m.emails = append(m.emails, e)
	return nil
}

func TestNewServer(t *testing.T) {
	cfg := &config.ServerConfig{
		Hostname:      "localhost",
		ListenAddress: "127.0.0.1:0",
	}
	
	queue := &mockQueue{}
	server := NewServer(cfg, queue, 25*1024*1024)
	
	if server == nil {
		t.Fatal("Expected server to be created")
	}
	
	if server.hostname != cfg.Hostname {
		t.Errorf("Expected hostname %s, got %s", cfg.Hostname, server.hostname)
	}
}

func TestServer_StartStop(t *testing.T) {
	cfg := &config.ServerConfig{
		Hostname:      "localhost",
		ListenAddress: "127.0.0.1:0",
	}
	
	queue := &mockQueue{}
	server := NewServer(cfg, queue, 25*1024*1024)
	
	// Start server
	errChan := make(chan error, 1)
	go func() {
		err := server.Start()
		if err != nil && !strings.Contains(err.Error(), "use of closed network connection") {
			errChan <- err
		}
	}()
	
	// Wait for server to start
	time.Sleep(100 * time.Millisecond)
	
	// Check if server is listening
	addr := server.Address()
	if addr == "" {
		t.Fatal("Server should have an address")
	}
	
	// Try to connect
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	conn.Close()
	
	// Stop server
	if err := server.Stop(); err != nil {
		t.Errorf("Failed to stop server: %v", err)
	}
	
	// Check for start errors
	select {
	case err := <-errChan:
		t.Errorf("Server start error: %v", err)
	case <-time.After(100 * time.Millisecond):
		// No error, good
	}
}

func TestServer_HandleEmail(t *testing.T) {
	cfg := &config.ServerConfig{
		Hostname:      "localhost",
		ListenAddress: "127.0.0.1:0",
	}
	
	queue := &mockQueue{}
	server := NewServer(cfg, queue, 25*1024*1024)
	
	// Start server
	go func() {
		server.Start()
	}()
	
	time.Sleep(100 * time.Millisecond)
	
	addr := server.Address()
	
	// Send test email
	from := "sender@example.com"
	to := []string{"recipient@example.com"}
	msg := []byte("Subject: Test\r\n\r\nThis is a test email")
	
	err := smtp.SendMail(addr, nil, from, to, msg)
	if err != nil {
		t.Fatalf("Failed to send email: %v", err)
	}
	
	// Wait for email to be processed
	time.Sleep(100 * time.Millisecond)
	
	// Check if email was queued
	if len(queue.emails) != 1 {
		t.Fatalf("Expected 1 email in queue, got %d", len(queue.emails))
	}
	
	queued := queue.emails[0]
	if queued.From != from {
		t.Errorf("Expected from %s, got %s", from, queued.From)
	}
	if len(queued.To) != 1 || queued.To[0] != to[0] {
		t.Errorf("Expected to %v, got %v", to, queued.To)
	}
	
	server.Stop()
}