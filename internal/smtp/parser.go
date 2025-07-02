package smtp

import (
	"bytes"
	"io"
	"net/mail"
	"strings"
	
	"github.com/tpdoyle87/simple-email-server/pkg/email"
)

func parseEmail(from string, to []string, r io.Reader) (*email.Email, error) {
	// Read the entire message
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(r); err != nil {
		return nil, err
	}
	
	// Parse message
	msg, err := mail.ReadMessage(bytes.NewReader(buf.Bytes()))
	if err != nil {
		return nil, err
	}
	
	// Extract headers
	headers := make(map[string]string)
	for k, v := range msg.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}
	
	// Read body
	body, err := io.ReadAll(msg.Body)
	if err != nil {
		return nil, err
	}
	
	// Create email object
	e := &email.Email{
		From:    from,
		To:      to,
		Subject: headers["Subject"],
		Headers: headers,
		Body:    string(body),
	}
	
	// Extract CC and BCC if present
	if cc := headers["Cc"]; cc != "" {
		e.CC = parseAddressList(cc)
	}
	
	if bcc := headers["Bcc"]; bcc != "" {
		e.BCC = parseAddressList(bcc)
	}
	
	return e, nil
}

func parseAddressList(addresses string) []string {
	var result []string
	for _, addr := range strings.Split(addresses, ",") {
		trimmed := strings.TrimSpace(addr)
		if trimmed != "" {
			// Extract email from "Name <email>" format
			if parsed, err := mail.ParseAddress(trimmed); err == nil {
				result = append(result, parsed.Address)
			} else {
				result = append(result, trimmed)
			}
		}
	}
	return result
}