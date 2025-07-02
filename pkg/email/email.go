package email

import (
	"errors"
	"net/mail"
	"strings"
	"time"
)

var (
	ErrInvalidFrom       = errors.New("invalid from address")
	ErrNoRecipients      = errors.New("no recipients specified")
	ErrInvalidRecipient  = errors.New("invalid recipient address")
	ErrEmptySubject      = errors.New("empty subject")
	ErrEmptyBody         = errors.New("empty body")
	ErrMessageTooLarge   = errors.New("message too large")
)

type Status string

const (
	StatusPending   Status = "pending"
	StatusQueued    Status = "queued"
	StatusSending   Status = "sending"
	StatusDelivered Status = "delivered"
	StatusFailed    Status = "failed"
	StatusBounced   Status = "bounced"
)

type Email struct {
	ID          string            `json:"id"`
	From        string            `json:"from"`
	To          []string          `json:"to"`
	CC          []string          `json:"cc,omitempty"`
	BCC         []string          `json:"bcc,omitempty"`
	Subject     string            `json:"subject"`
	Body        string            `json:"body"`
	HTML        string            `json:"html,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Attachments []Attachment      `json:"attachments,omitempty"`
	
	Status      Status            `json:"status"`
	RetryCount  int               `json:"retry_count"`
	LastError   string            `json:"last_error,omitempty"`
	
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	ScheduledAt *time.Time        `json:"scheduled_at,omitempty"`
	DeliveredAt *time.Time        `json:"delivered_at,omitempty"`
}

type Attachment struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Data        []byte `json:"data"`
}

func (e *Email) Validate(maxMessageSize int64) error {
	if e.From == "" {
		return ErrInvalidFrom
	}
	
	if _, err := mail.ParseAddress(e.From); err != nil {
		return ErrInvalidFrom
	}
	
	if len(e.To) == 0 {
		return ErrNoRecipients
	}
	
	for _, addr := range e.To {
		if _, err := mail.ParseAddress(addr); err != nil {
			return ErrInvalidRecipient
		}
	}
	
	for _, addr := range e.CC {
		if _, err := mail.ParseAddress(addr); err != nil {
			return ErrInvalidRecipient
		}
	}
	
	for _, addr := range e.BCC {
		if _, err := mail.ParseAddress(addr); err != nil {
			return ErrInvalidRecipient
		}
	}
	
	if strings.TrimSpace(e.Subject) == "" {
		return ErrEmptySubject
	}
	
	if strings.TrimSpace(e.Body) == "" && strings.TrimSpace(e.HTML) == "" {
		return ErrEmptyBody
	}
	
	size := int64(len(e.Body) + len(e.HTML))
	for _, att := range e.Attachments {
		size += int64(len(att.Data))
	}
	
	if size > maxMessageSize {
		return ErrMessageTooLarge
	}
	
	return nil
}

func (e *Email) Recipients() []string {
	recipients := make([]string, 0, len(e.To)+len(e.CC)+len(e.BCC))
	recipients = append(recipients, e.To...)
	recipients = append(recipients, e.CC...)
	recipients = append(recipients, e.BCC...)
	return recipients
}