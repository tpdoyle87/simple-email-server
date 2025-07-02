package smtp

import (
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"
	
	"github.com/emersion/go-smtp"
	"github.com/google/uuid"
	"github.com/tpdoyle87/simple-email-server/internal/config"
	"github.com/tpdoyle87/simple-email-server/pkg/email"
)

type Queue interface {
	Enqueue(*email.Email) error
}

type Server struct {
	config         *config.ServerConfig
	queue          Queue
	maxMessageSize int64
	hostname       string
	
	smtpServer *smtp.Server
	listener   net.Listener
	mu         sync.RWMutex
}

func NewServer(cfg *config.ServerConfig, queue Queue, maxMessageSize int64) *Server {
	s := &Server{
		config:         cfg,
		queue:          queue,
		maxMessageSize: maxMessageSize,
		hostname:       cfg.Hostname,
	}
	
	backend := &smtpBackend{
		server: s,
	}
	
	smtpServer := smtp.NewServer(backend)
	smtpServer.Addr = cfg.ListenAddress
	smtpServer.Domain = cfg.Hostname
	smtpServer.MaxMessageBytes = maxMessageSize
	smtpServer.MaxRecipients = 100
	smtpServer.ReadTimeout = 10 * time.Second
	smtpServer.WriteTimeout = 10 * time.Second
	smtpServer.AllowInsecureAuth = !cfg.TLS.Enabled
	
	s.smtpServer = smtpServer
	
	return s
}

func (s *Server) Start() error {
	listener, err := net.Listen("tcp", s.config.ListenAddress)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	
	s.mu.Lock()
	s.listener = listener
	s.mu.Unlock()
	
	log.Printf("SMTP server listening on %s", listener.Addr())
	
	return s.smtpServer.Serve(listener)
}

func (s *Server) Stop() error {
	s.mu.RLock()
	listener := s.listener
	s.mu.RUnlock()
	
	if listener != nil {
		return listener.Close()
	}
	
	return s.smtpServer.Close()
}

func (s *Server) Address() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if s.listener != nil {
		return s.listener.Addr().String()
	}
	
	return ""
}

type smtpBackend struct {
	server *Server
}

func (b *smtpBackend) NewSession(c *smtp.Conn) (smtp.Session, error) {
	return &smtpSession{
		server: b.server,
		conn:   c,
	}, nil
}

type smtpSession struct {
	server     *Server
	conn       *smtp.Conn
	from       string
	to         []string
	authPassed bool
}

func (s *smtpSession) AuthPlain(username, password string) error {
	// TODO: Implement authentication
	s.authPassed = true
	return nil
}

func (s *smtpSession) Mail(from string, opts *smtp.MailOptions) error {
	s.from = from
	return nil
}

func (s *smtpSession) Rcpt(to string, opts *smtp.RcptOptions) error {
	s.to = append(s.to, to)
	return nil
}

func (s *smtpSession) Data(r io.Reader) error {
	// Parse email
	parsedEmail, err := parseEmail(s.from, s.to, r)
	if err != nil {
		return fmt.Errorf("failed to parse email: %w", err)
	}
	
	// Validate email
	if err := parsedEmail.Validate(s.server.maxMessageSize); err != nil {
		return fmt.Errorf("invalid email: %w", err)
	}
	
	// Add metadata
	parsedEmail.ID = uuid.New().String()
	parsedEmail.Status = email.StatusQueued
	parsedEmail.CreatedAt = time.Now()
	parsedEmail.UpdatedAt = time.Now()
	
	// Queue email
	if err := s.server.queue.Enqueue(parsedEmail); err != nil {
		return fmt.Errorf("failed to queue email: %w", err)
	}
	
	log.Printf("Email %s queued from %s to %v", parsedEmail.ID, parsedEmail.From, parsedEmail.To)
	
	return nil
}

func (s *smtpSession) Reset() {
	s.from = ""
	s.to = nil
}

func (s *smtpSession) Logout() error {
	return nil
}