package delivery

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/smtp"
	"strings"
	"time"
	
	"github.com/tpdoyle87/simple-email-server/pkg/email"
)

type SimpleSMTPClient struct {
	timeout time.Duration
}

func NewSMTPClient(timeout time.Duration) *SimpleSMTPClient {
	return &SimpleSMTPClient{
		timeout: timeout,
	}
}

func (c *SimpleSMTPClient) Send(ctx context.Context, host string, e *email.Email) error {
	// Add port if not present
	if !strings.Contains(host, ":") {
		host = host + ":25"
	}
	
	// Create dialer with timeout
	dialer := &net.Dialer{
		Timeout: c.timeout,
	}
	
	// Dial with context
	conn, err := dialer.DialContext(ctx, "tcp", host)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.Close()
	
	// Create SMTP client
	client, err := smtp.NewClient(conn, strings.Split(host, ":")[0])
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Close()
	
	// Try STARTTLS
	if ok, _ := client.Extension("STARTTLS"); ok {
		config := &tls.Config{ServerName: strings.Split(host, ":")[0]}
		if err = client.StartTLS(config); err != nil {
			// Log but continue without TLS
			fmt.Printf("STARTTLS failed: %v\n", err)
		}
	}
	
	// Set sender
	if err = client.Mail(e.From); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}
	
	// Set recipients
	for _, to := range e.Recipients() {
		if err = client.Rcpt(to); err != nil {
			return fmt.Errorf("failed to set recipient %s: %w", to, err)
		}
	}
	
	// Send data
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}
	
	// Write email
	if err = writeEmail(w, e); err != nil {
		w.Close()
		return fmt.Errorf("failed to write email: %w", err)
	}
	
	if err = w.Close(); err != nil {
		return fmt.Errorf("failed to close data writer: %w", err)
	}
	
	// Quit
	return client.Quit()
}

func writeEmail(w io.Writer, e *email.Email) error {
	// Write headers
	headers := []string{
		fmt.Sprintf("From: %s", e.From),
		fmt.Sprintf("To: %s", strings.Join(e.To, ", ")),
		fmt.Sprintf("Subject: %s", e.Subject),
		fmt.Sprintf("Date: %s", time.Now().Format(time.RFC1123Z)),
		"MIME-Version: 1.0",
	}
	
	if len(e.CC) > 0 {
		headers = append(headers, fmt.Sprintf("Cc: %s", strings.Join(e.CC, ", ")))
	}
	
	// Add custom headers
	for k, v := range e.Headers {
		if !isStandardHeader(k) {
			headers = append(headers, fmt.Sprintf("%s: %s", k, v))
		}
	}
	
	// Determine content type
	if e.HTML != "" {
		headers = append(headers, "Content-Type: text/html; charset=utf-8")
	} else {
		headers = append(headers, "Content-Type: text/plain; charset=utf-8")
	}
	
	// Write headers
	for _, h := range headers {
		if _, err := fmt.Fprintf(w, "%s\r\n", h); err != nil {
			return err
		}
	}
	
	// Empty line between headers and body
	if _, err := fmt.Fprint(w, "\r\n"); err != nil {
		return err
	}
	
	// Write body
	body := e.Body
	if e.HTML != "" {
		body = e.HTML
	}
	
	_, err := fmt.Fprint(w, body)
	return err
}

func isStandardHeader(key string) bool {
	standard := []string{"from", "to", "cc", "bcc", "subject", "date", "mime-version", "content-type"}
	lower := strings.ToLower(key)
	for _, s := range standard {
		if lower == s {
			return true
		}
	}
	return false
}