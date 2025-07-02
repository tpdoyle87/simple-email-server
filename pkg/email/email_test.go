package email

import (
	"testing"
	"time"
)

func TestEmail_Validate(t *testing.T) {
	tests := []struct {
		name           string
		email          *Email
		maxMessageSize int64
		wantErr        error
	}{
		{
			name: "valid email",
			email: &Email{
				From:    "sender@example.com",
				To:      []string{"recipient@example.com"},
				Subject: "Test Subject",
				Body:    "Test Body",
			},
			maxMessageSize: 25 * 1024 * 1024,
			wantErr:        nil,
		},
		{
			name: "missing from",
			email: &Email{
				To:      []string{"recipient@example.com"},
				Subject: "Test Subject",
				Body:    "Test Body",
			},
			maxMessageSize: 25 * 1024 * 1024,
			wantErr:        ErrInvalidFrom,
		},
		{
			name: "invalid from address",
			email: &Email{
				From:    "invalid-email",
				To:      []string{"recipient@example.com"},
				Subject: "Test Subject",
				Body:    "Test Body",
			},
			maxMessageSize: 25 * 1024 * 1024,
			wantErr:        ErrInvalidFrom,
		},
		{
			name: "no recipients",
			email: &Email{
				From:    "sender@example.com",
				Subject: "Test Subject",
				Body:    "Test Body",
			},
			maxMessageSize: 25 * 1024 * 1024,
			wantErr:        ErrNoRecipients,
		},
		{
			name: "invalid recipient",
			email: &Email{
				From:    "sender@example.com",
				To:      []string{"invalid-recipient"},
				Subject: "Test Subject",
				Body:    "Test Body",
			},
			maxMessageSize: 25 * 1024 * 1024,
			wantErr:        ErrInvalidRecipient,
		},
		{
			name: "empty subject",
			email: &Email{
				From: "sender@example.com",
				To:   []string{"recipient@example.com"},
				Body: "Test Body",
			},
			maxMessageSize: 25 * 1024 * 1024,
			wantErr:        ErrEmptySubject,
		},
		{
			name: "empty body and html",
			email: &Email{
				From:    "sender@example.com",
				To:      []string{"recipient@example.com"},
				Subject: "Test Subject",
			},
			maxMessageSize: 25 * 1024 * 1024,
			wantErr:        ErrEmptyBody,
		},
		{
			name: "valid with html only",
			email: &Email{
				From:    "sender@example.com",
				To:      []string{"recipient@example.com"},
				Subject: "Test Subject",
				HTML:    "<p>Test HTML</p>",
			},
			maxMessageSize: 25 * 1024 * 1024,
			wantErr:        nil,
		},
		{
			name: "message too large",
			email: &Email{
				From:    "sender@example.com",
				To:      []string{"recipient@example.com"},
				Subject: "Test Subject",
				Body:    "Test Body",
			},
			maxMessageSize: 5,
			wantErr:        ErrMessageTooLarge,
		},
		{
			name: "valid with cc and bcc",
			email: &Email{
				From:    "sender@example.com",
				To:      []string{"recipient@example.com"},
				CC:      []string{"cc@example.com"},
				BCC:     []string{"bcc@example.com"},
				Subject: "Test Subject",
				Body:    "Test Body",
			},
			maxMessageSize: 25 * 1024 * 1024,
			wantErr:        nil,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.email.Validate(tt.maxMessageSize)
			if err != tt.wantErr {
				t.Errorf("Email.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEmail_Recipients(t *testing.T) {
	email := &Email{
		To:  []string{"to1@example.com", "to2@example.com"},
		CC:  []string{"cc1@example.com", "cc2@example.com"},
		BCC: []string{"bcc1@example.com"},
	}
	
	recipients := email.Recipients()
	expected := []string{
		"to1@example.com",
		"to2@example.com",
		"cc1@example.com",
		"cc2@example.com",
		"bcc1@example.com",
	}
	
	if len(recipients) != len(expected) {
		t.Errorf("Recipients() returned %d recipients, expected %d", len(recipients), len(expected))
	}
	
	for i, r := range recipients {
		if r != expected[i] {
			t.Errorf("Recipients()[%d] = %s, expected %s", i, r, expected[i])
		}
	}
}

func BenchmarkEmail_Validate(b *testing.B) {
	email := &Email{
		From:    "sender@example.com",
		To:      []string{"recipient1@example.com", "recipient2@example.com"},
		CC:      []string{"cc@example.com"},
		Subject: "Test Subject",
		Body:    "Test Body",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = email.Validate(25 * 1024 * 1024)
	}
}

func TestEmailStatus(t *testing.T) {
	email := &Email{
		ID:        "test-id",
		Status:    StatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	if email.Status != StatusPending {
		t.Errorf("Expected status %s, got %s", StatusPending, email.Status)
	}
	
	email.Status = StatusQueued
	if email.Status != StatusQueued {
		t.Errorf("Expected status %s, got %s", StatusQueued, email.Status)
	}
}