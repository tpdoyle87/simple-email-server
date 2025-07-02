package delivery

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"
	
	"github.com/tpdoyle87/simple-email-server/internal/config"
	"github.com/tpdoyle87/simple-email-server/internal/queue"
	"github.com/tpdoyle87/simple-email-server/pkg/email"
)

type DNSResolver interface {
	LookupMX(domain string) ([]*net.MX, error)
}

type SMTPClient interface {
	Send(ctx context.Context, host string, email *email.Email) error
}

type Service struct {
	config   *config.DeliveryConfig
	queue    queue.Queue
	resolver DNSResolver
	client   SMTPClient
	maxRetry int
	
	dnsCache     map[string]*dnsCacheEntry
	dnsCacheMu   sync.RWMutex
	
	wg           sync.WaitGroup
}

type dnsCacheEntry struct {
	mx        []*net.MX
	expiresAt time.Time
}

type dnsResolver struct {
	lookupMX func(string) ([]*net.MX, error)
}

func (d *dnsResolver) LookupMX(domain string) ([]*net.MX, error) {
	if d.lookupMX != nil {
		return d.lookupMX(domain)
	}
	return net.LookupMX(domain)
}

func NewService(cfg *config.DeliveryConfig, q queue.Queue) *Service {
	return &Service{
		config:   cfg,
		queue:    q,
		resolver: &dnsResolver{},
		client:   NewSMTPClient(cfg.ConnectionTimeout),
		dnsCache: make(map[string]*dnsCacheEntry),
		maxRetry: 5, // Default max retry
	}
}

func (s *Service) Start(ctx context.Context) {
	log.Printf("Starting delivery service with %d workers", s.config.Workers)
	
	// Start workers
	for i := 0; i < s.config.Workers; i++ {
		s.wg.Add(1)
		go s.worker(ctx, i)
	}
	
	// Wait for context cancellation
	<-ctx.Done()
	
	log.Println("Stopping delivery service...")
	s.wg.Wait()
	log.Println("Delivery service stopped")
}

func (s *Service) worker(ctx context.Context, id int) {
	defer s.wg.Done()
	
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Dequeue emails
			emails, err := s.queue.Dequeue(10)
			if err != nil {
				log.Printf("Worker %d: Failed to dequeue emails: %v", id, err)
				continue
			}
			
			// Process emails
			for _, e := range emails {
				if err := s.processEmail(ctx, e); err != nil {
					log.Printf("Worker %d: Failed to deliver email %s: %v", id, e.ID, err)
					
					// Mark as failed with retry
					shouldRetry := e.RetryCount < s.maxRetry
					if err := s.queue.MarkFailed(e.ID, err.Error(), shouldRetry); err != nil {
						log.Printf("Worker %d: Failed to mark email %s as failed: %v", id, e.ID, err)
					}
				} else {
					// Mark as delivered
					if err := s.queue.MarkDelivered(e.ID); err != nil {
						log.Printf("Worker %d: Failed to mark email %s as delivered: %v", id, e.ID, err)
					}
				}
			}
		}
	}
}

func (s *Service) processEmail(ctx context.Context, e *email.Email) error {
	// Extract domain from first recipient
	if len(e.To) == 0 {
		return fmt.Errorf("no recipients")
	}
	
	domain := extractDomain(e.To[0])
	if domain == "" {
		return fmt.Errorf("invalid recipient domain")
	}
	
	// Get MX records
	mxRecords, err := s.getMXRecords(domain)
	if err != nil {
		return fmt.Errorf("failed to get MX records: %w", err)
	}
	
	// Try each MX server
	var lastErr error
	for _, mx := range mxRecords {
		// Create context with timeout
		deliveryCtx, cancel := context.WithTimeout(ctx, s.config.ConnectionTimeout)
		
		// Attempt delivery
		err := s.client.Send(deliveryCtx, mx.Host, e)
		cancel()
		
		if err == nil {
			log.Printf("Email %s delivered to %s", e.ID, mx.Host)
			return nil
		}
		
		lastErr = err
		log.Printf("Failed to deliver email %s to %s: %v", e.ID, mx.Host, err)
	}
	
	if lastErr != nil {
		return fmt.Errorf("all MX servers failed: %w", lastErr)
	}
	
	return fmt.Errorf("no MX servers found")
}

func (s *Service) getMXRecords(domain string) ([]*net.MX, error) {
	// Check cache
	s.dnsCacheMu.RLock()
	entry, exists := s.dnsCache[domain]
	s.dnsCacheMu.RUnlock()
	
	if exists && entry.expiresAt.After(time.Now()) {
		return entry.mx, nil
	}
	
	// Lookup MX records
	mx, err := s.resolver.LookupMX(domain)
	if err != nil {
		return nil, err
	}
	
	// Update cache
	s.dnsCacheMu.Lock()
	s.dnsCache[domain] = &dnsCacheEntry{
		mx:        mx,
		expiresAt: time.Now().Add(s.config.DNSCacheTTL),
	}
	s.dnsCacheMu.Unlock()
	
	return mx, nil
}

func extractDomain(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return ""
	}
	return parts[1]
}

