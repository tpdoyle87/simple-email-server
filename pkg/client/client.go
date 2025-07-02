package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is the email server client
type Client struct {
	baseURL    string
	authToken  string
	httpClient *http.Client
}

// Email represents an email to send
type Email struct {
	From        string            `json:"from"`
	To          []string          `json:"to"`
	CC          []string          `json:"cc,omitempty"`
	BCC         []string          `json:"bcc,omitempty"`
	Subject     string            `json:"subject"`
	Body        string            `json:"body"`
	HTML        string            `json:"html,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	ScheduledAt *time.Time        `json:"scheduled_at,omitempty"`
}

// SendResponse is the response from sending an email
type SendResponse struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// StatusResponse is the response from checking email status
type StatusResponse struct {
	ID          string     `json:"id"`
	Status      string     `json:"status"`
	RetryCount  int        `json:"retry_count"`
	LastError   string     `json:"last_error,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeliveredAt *time.Time `json:"delivered_at,omitempty"`
}

// StatsResponse is the response from the stats endpoint
type StatsResponse struct {
	QueueSize      int   `json:"queue_size"`
	TotalSent      int64 `json:"total_sent"`
	TotalDelivered int64 `json:"total_delivered"`
	TotalFailed    int64 `json:"total_failed"`
}

// New creates a new email server client
func New(baseURL, authToken string) *Client {
	return &Client{
		baseURL:   baseURL,
		authToken: authToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewWithHTTPClient creates a new client with a custom HTTP client
func NewWithHTTPClient(baseURL, authToken string, httpClient *http.Client) *Client {
	return &Client{
		baseURL:    baseURL,
		authToken:  authToken,
		httpClient: httpClient,
	}
}

// Send sends a single email
func (c *Client) Send(email *Email) (*SendResponse, error) {
	body, err := json.Marshal(email)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal email: %w", err)
	}
	
	req, err := http.NewRequest("POST", c.baseURL+"/send", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.authToken)
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, body)
	}
	
	var sendResp SendResponse
	if err := json.NewDecoder(resp.Body).Decode(&sendResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return &sendResp, nil
}

// SendBatch sends multiple emails in one request
func (c *Client) SendBatch(emails []*Email) ([]*SendResponse, error) {
	body, err := json.Marshal(emails)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal emails: %w", err)
	}
	
	req, err := http.NewRequest("POST", c.baseURL+"/send/batch", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.authToken)
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, body)
	}
	
	var responses []*SendResponse
	if err := json.NewDecoder(resp.Body).Decode(&responses); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return responses, nil
}

// GetStatus gets the status of an email by ID
func (c *Client) GetStatus(id string) (*StatusResponse, error) {
	req, err := http.NewRequest("GET", c.baseURL+"/status/"+id, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Authorization", "Bearer "+c.authToken)
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, body)
	}
	
	var statusResp StatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&statusResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return &statusResp, nil
}

// GetStats gets server statistics
func (c *Client) GetStats() (*StatsResponse, error) {
	req, err := http.NewRequest("GET", c.baseURL+"/stats", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Authorization", "Bearer "+c.authToken)
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, body)
	}
	
	var statsResp StatsResponse
	if err := json.NewDecoder(resp.Body).Decode(&statsResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return &statsResp, nil
}