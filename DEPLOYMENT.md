# Deployment Guide

> **Prefer Kamal?** Check out [KAMAL.md](KAMAL.md) for automated deployment with Kamal.

## Prerequisites

- Go 1.21+ installed
- Linux/Unix server (Ubuntu, Debian, CentOS, etc.)
- Domain name with DNS control
- Static IP address (recommended)
- Port 25/587 open (check with hosting provider)

## Quick Start

### 1. Clone and Build

```bash
git clone https://github.com/yourusername/simple-email-server
cd simple-email-server
go build -o emailserver cmd/emailserver/main.go
```

### 2. Basic Configuration

Create `config.yaml`:

```yaml
server:
  hostname: "mail.yourdomain.com"
  listen_address: "0.0.0.0:587"
  tls:
    enabled: true
    cert_file: "/path/to/cert.pem"
    key_file: "/path/to/key.pem"

api:
  listen_address: "127.0.0.1:8080"
  auth_token: "your-secret-token-here"

queue:
  max_retry: 5
  retry_delay: "5m"
  max_queue_size: 10000

delivery:
  workers: 20
  dns_cache_ttl: "5m"
  connection_timeout: "30s"
  
limits:
  max_recipients: 100
  max_message_size: "25MB"
  rate_limit: "100/minute"
```

### 3. DNS Configuration

Add these records to your domain:

```
# MX Record (for receiving mail - optional)
yourdomain.com.    MX    10    mail.yourdomain.com.

# A Record (required)
mail.yourdomain.com.    A    YOUR_SERVER_IP

# SPF Record (required for sending)
yourdomain.com.    TXT    "v=spf1 ip4:YOUR_SERVER_IP ~all"

# Reverse DNS (PTR) - Contact your hosting provider
YOUR_SERVER_IP    PTR    mail.yourdomain.com.
```

### 4. Running the Server

```bash
# Run directly
./emailserver -config config.yaml

# Or use systemd (recommended)
sudo cp emailserver.service /etc/systemd/system/
sudo systemctl enable emailserver
sudo systemctl start emailserver
```

## Docker Deployment

```bash
# Build image
docker build -t simple-email-server .

# Run with docker-compose
docker-compose up -d
```

Example `docker-compose.yml`:

```yaml
version: '3.8'
services:
  emailserver:
    image: simple-email-server
    ports:
      - "587:587"
      - "127.0.0.1:8080:8080"
    volumes:
      - ./config.yaml:/app/config.yaml
      - ./data:/app/data
    environment:
      - API_AUTH_TOKEN=${API_AUTH_TOKEN}
    restart: unless-stopped
```

## Application Integration

### Go Application

```go
import "github.com/yourusername/simple-email-server/pkg/client"

// Create client
emailClient := client.New("http://localhost:8080", "your-auth-token")

// Send email
err := emailClient.Send(&client.Email{
    From:    "app@yourdomain.com",
    To:      []string{"user@example.com"},
    Subject: "Hello from my app",
    Body:    "This is a test email",
})
```

### HTTP API

```bash
curl -X POST http://localhost:8080/send \
  -H "Authorization: Bearer your-auth-token" \
  -H "Content-Type: application/json" \
  -d '{
    "from": "app@yourdomain.com",
    "to": ["user@example.com"],
    "subject": "Test Email",
    "body": "This is a test"
  }'
```

### Other Languages

```python
# Python example
import requests

response = requests.post(
    "http://localhost:8080/send",
    headers={"Authorization": "Bearer your-auth-token"},
    json={
        "from": "app@yourdomain.com",
        "to": ["user@example.com"],
        "subject": "Test Email",
        "body": "This is a test"
    }
)
```

## Security Considerations

1. **API Token**: Generate strong tokens: `openssl rand -base64 32`
2. **Firewall**: Only expose port 587 publicly, keep API internal
3. **TLS**: Always use HTTPS/TLS for API and STARTTLS for SMTP
4. **Rate Limiting**: Configure appropriate limits to prevent abuse
5. **Monitoring**: Set up alerts for queue depth and failed deliveries

## Troubleshooting

### Port 25 Blocked
Many cloud providers block port 25. Use port 587 with STARTTLS instead.

### Emails Going to Spam
1. Ensure SPF record is correct
2. Set up reverse DNS (PTR record)
3. Start with low volume to build IP reputation
4. Consider DKIM signing (advanced feature)

### High Queue Depth
- Increase delivery workers
- Check for network issues
- Verify recipient servers aren't blocking you

## Performance Tuning

```yaml
# For high-volume sending
delivery:
  workers: 50  # Increase workers
  connection_pool_size: 100
  dns_cache_ttl: "15m"  # Longer cache

queue:
  batch_size: 200  # Process more emails per batch
  
# For low-resource servers
delivery:
  workers: 5
  connection_pool_size: 20
  
limits:
  rate_limit: "10/minute"
```