package queue

import (
	"errors"
	"sync"
	"time"
	
	"github.com/tpdoyle87/simple-email-server/pkg/email"
)

var (
	ErrQueueFull  = errors.New("queue is full")
	ErrEmailNotFound = errors.New("email not found")
)

type Queue interface {
	Enqueue(*email.Email) error
	Dequeue(count int) ([]*email.Email, error)
	MarkDelivered(id string) error
	MarkFailed(id string, reason string, retry bool) error
	Size() int
}

type MemoryQueue struct {
	mu        sync.RWMutex
	emails    []*email.Email
	emailMap  map[string]*email.Email
	maxSize   int
}

func NewMemoryQueue(maxSize int) *MemoryQueue {
	return &MemoryQueue{
		emails:   make([]*email.Email, 0),
		emailMap: make(map[string]*email.Email),
		maxSize:  maxSize,
	}
}

func (q *MemoryQueue) Enqueue(e *email.Email) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	
	if len(q.emails) >= q.maxSize {
		return ErrQueueFull
	}
	
	e.UpdatedAt = time.Now()
	q.emails = append(q.emails, e)
	q.emailMap[e.ID] = e
	
	return nil
}

func (q *MemoryQueue) Dequeue(count int) ([]*email.Email, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	
	result := make([]*email.Email, 0, count)
	
	// Find emails ready to send
	now := time.Now()
	for i := 0; i < len(q.emails) && len(result) < count; i++ {
		e := q.emails[i]
		
		// Skip if scheduled for future
		if e.ScheduledAt != nil && e.ScheduledAt.After(now) {
			continue
		}
		
		// Skip if already sending or not queued
		if e.Status != email.StatusQueued {
			continue
		}
		
		// Mark as sending
		e.Status = email.StatusSending
		e.UpdatedAt = now
		result = append(result, e)
	}
	
	return result, nil
}

func (q *MemoryQueue) MarkDelivered(id string) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	
	e, exists := q.emailMap[id]
	if !exists {
		return ErrEmailNotFound
	}
	
	// Update status
	now := time.Now()
	e.Status = email.StatusDelivered
	e.UpdatedAt = now
	e.DeliveredAt = &now
	
	// Remove from queue
	q.removeEmail(id)
	
	return nil
}

func (q *MemoryQueue) MarkFailed(id string, reason string, retry bool) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	
	e, exists := q.emailMap[id]
	if !exists {
		return ErrEmailNotFound
	}
	
	// Update email
	e.LastError = reason
	e.UpdatedAt = time.Now()
	
	if retry {
		e.Status = email.StatusQueued
		e.RetryCount++
		
		// Calculate next retry time with exponential backoff
		retryDelay := time.Duration(e.RetryCount) * 5 * time.Minute
		nextRetry := time.Now().Add(retryDelay)
		e.ScheduledAt = &nextRetry
	} else {
		e.Status = email.StatusFailed
		q.removeEmail(id)
	}
	
	return nil
}

func (q *MemoryQueue) Size() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	
	return len(q.emails)
}

func (q *MemoryQueue) removeEmail(id string) {
	// Remove from slice
	for i, e := range q.emails {
		if e.ID == id {
			q.emails = append(q.emails[:i], q.emails[i+1:]...)
			break
		}
	}
	
	// Remove from map
	delete(q.emailMap, id)
}