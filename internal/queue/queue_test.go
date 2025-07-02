package queue

import (
	"sync"
	"testing"
	"time"
	
	"github.com/tpdoyle87/simple-email-server/pkg/email"
)

func TestMemoryQueue_EnqueueDequeue(t *testing.T) {
	q := NewMemoryQueue(10)
	
	// Test enqueue
	e := &email.Email{
		ID:      "test-1",
		From:    "sender@example.com",
		To:      []string{"recipient@example.com"},
		Subject: "Test",
		Body:    "Test body",
		Status:  email.StatusQueued,
	}
	
	err := q.Enqueue(e)
	if err != nil {
		t.Fatalf("Failed to enqueue email: %v", err)
	}
	
	// Test dequeue
	emails, err := q.Dequeue(1)
	if err != nil {
		t.Fatalf("Failed to dequeue emails: %v", err)
	}
	
	if len(emails) != 1 {
		t.Fatalf("Expected 1 email, got %d", len(emails))
	}
	
	if emails[0].ID != e.ID {
		t.Errorf("Expected email ID %s, got %s", e.ID, emails[0].ID)
	}
	
	if emails[0].Status != email.StatusSending {
		t.Errorf("Expected status %s, got %s", email.StatusSending, emails[0].Status)
	}
}

func TestMemoryQueue_MaxSize(t *testing.T) {
	q := NewMemoryQueue(2)
	
	// Fill queue
	for i := 0; i < 2; i++ {
		e := &email.Email{
			ID:     "test-" + string(rune(i)),
			Status: email.StatusQueued,
		}
		err := q.Enqueue(e)
		if err != nil {
			t.Fatalf("Failed to enqueue email %d: %v", i, err)
		}
	}
	
	// Try to exceed max size
	e := &email.Email{
		ID:     "test-3",
		Status: email.StatusQueued,
	}
	err := q.Enqueue(e)
	if err == nil {
		t.Error("Expected error when exceeding max size")
	}
}

func TestMemoryQueue_BatchDequeue(t *testing.T) {
	q := NewMemoryQueue(100)
	
	// Enqueue 10 emails
	for i := 0; i < 10; i++ {
		e := &email.Email{
			ID:     "test-" + string(rune(i)),
			Status: email.StatusQueued,
		}
		err := q.Enqueue(e)
		if err != nil {
			t.Fatalf("Failed to enqueue email %d: %v", i, err)
		}
	}
	
	// Dequeue batch of 5
	emails, err := q.Dequeue(5)
	if err != nil {
		t.Fatalf("Failed to dequeue emails: %v", err)
	}
	
	if len(emails) != 5 {
		t.Fatalf("Expected 5 emails, got %d", len(emails))
	}
	
	// Check remaining queue that can be dequeued (status = queued)
	remaining, _ := q.Dequeue(100)
	if len(remaining) != 5 {
		t.Errorf("Expected 5 more emails to dequeue, got %d", len(remaining))
	}
}

func TestMemoryQueue_MarkDelivered(t *testing.T) {
	q := NewMemoryQueue(10)
	
	e := &email.Email{
		ID:     "test-1",
		Status: email.StatusQueued,
	}
	
	err := q.Enqueue(e)
	if err != nil {
		t.Fatalf("Failed to enqueue email: %v", err)
	}
	
	emails, _ := q.Dequeue(1)
	if len(emails) != 1 {
		t.Fatal("Failed to dequeue email")
	}
	
	// Mark as delivered
	err = q.MarkDelivered(emails[0].ID)
	if err != nil {
		t.Fatalf("Failed to mark email as delivered: %v", err)
	}
	
	// Email should no longer be in queue
	size := q.Size()
	if size != 0 {
		t.Errorf("Expected queue size 0, got %d", size)
	}
}

func TestMemoryQueue_MarkFailed(t *testing.T) {
	q := NewMemoryQueue(10)
	
	e := &email.Email{
		ID:         "test-1",
		Status:     email.StatusQueued,
		RetryCount: 0,
	}
	
	err := q.Enqueue(e)
	if err != nil {
		t.Fatalf("Failed to enqueue email: %v", err)
	}
	
	emails, _ := q.Dequeue(1)
	if len(emails) != 1 {
		t.Fatal("Failed to dequeue email")
	}
	
	// Mark as failed with retry
	err = q.MarkFailed(emails[0].ID, "Connection refused", true)
	if err != nil {
		t.Fatalf("Failed to mark email as failed: %v", err)
	}
	
	// Email should be back in queue
	size := q.Size()
	if size != 1 {
		t.Errorf("Expected queue size 1, got %d", size)
	}
	
	// Wait a moment for retry scheduling
	time.Sleep(10 * time.Millisecond)
	
	// Force dequeue ignoring schedule for test
	emails, _ = q.Dequeue(1)
	if len(emails) > 0 {
		// Should not dequeue because it's scheduled for future
		t.Error("Email should be scheduled for future retry")
	}
	
	// Get the email directly to check retry count
	q.mu.RLock()
	retryEmail := q.emailMap["test-1"]
	q.mu.RUnlock()
	
	if retryEmail == nil {
		t.Fatal("Email should still be in queue")
	}
	if retryEmail.RetryCount != 1 {
		t.Errorf("Expected retry count 1, got %d", retryEmail.RetryCount)
	}
	if retryEmail.LastError != "Connection refused" {
		t.Errorf("Expected last error 'Connection refused', got '%s'", retryEmail.LastError)
	}
}

func TestMemoryQueue_Concurrent(t *testing.T) {
	q := NewMemoryQueue(1000)
	
	var wg sync.WaitGroup
	
	// Concurrent enqueuers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				e := &email.Email{
					ID:     string(rune(id*10 + j)),
					Status: email.StatusQueued,
				}
				q.Enqueue(e)
			}
		}(i)
	}
	
	// Concurrent dequeuers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				q.Dequeue(5)
				time.Sleep(1 * time.Millisecond)
			}
		}()
	}
	
	wg.Wait()
}

func BenchmarkMemoryQueue_Enqueue(b *testing.B) {
	q := NewMemoryQueue(b.N + 1)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e := &email.Email{
			ID:     string(rune(i)),
			Status: email.StatusQueued,
		}
		q.Enqueue(e)
	}
}

func BenchmarkMemoryQueue_Dequeue(b *testing.B) {
	q := NewMemoryQueue(b.N + 1)
	
	// Pre-fill queue
	for i := 0; i < b.N; i++ {
		e := &email.Email{
			ID:     string(rune(i)),
			Status: email.StatusQueued,
		}
		q.Enqueue(e)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.Dequeue(1)
	}
}