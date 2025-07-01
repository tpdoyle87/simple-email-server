# Simple Email Server

[![Tests](https://github.com/yourusername/simple-email-server/actions/workflows/test.yml/badge.svg)](https://github.com/yourusername/simple-email-server/actions/workflows/test.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/yourusername/simple-email-server)](https://goreportcard.com/report/github.com/yourusername/simple-email-server)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/github/go-mod/go-version/yourusername/simple-email-server)](https://go.dev/)
[![Docker Pulls](https://img.shields.io/docker/pulls/yourusername/simple-email-server)](https://hub.docker.com/r/yourusername/simple-email-server)

A high-performance, self-hosted email server written in Go. Send emails from your applications without relying on third-party services like SendGrid or AWS SES.

## Features

- 🚀 **High Performance** - Handle 500+ emails/second
- 🔒 **Secure** - TLS/STARTTLS support with Let's Encrypt integration
- 🛠 **Easy Integration** - Simple HTTP API for any application
- 📦 **Queue System** - Reliable delivery with automatic retries
- 🐳 **Docker Ready** - Deploy with Docker or Kamal in minutes
- 📊 **Monitoring** - Built-in metrics and health checks
- 🔧 **Configurable** - Extensive configuration options

## Quick Start

### Using Docker

```bash
docker run -d \
  -p 587:587 \
  -p 127.0.0.1:8080:8080 \
  -v ./config.yaml:/app/config.yaml \
  -e API_AUTH_TOKEN=your-secret-token \
  yourusername/simple-email-server
```

### Using Kamal

```bash
# Install Kamal
gem install kamal

# Deploy
kamal setup
kamal deploy
```

See [KAMAL.md](KAMAL.md) for detailed instructions.

### From Source

```bash
# Clone repository
git clone https://github.com/yourusername/simple-email-server
cd simple-email-server

# Build
go build -o emailserver cmd/emailserver/main.go

# Run
./emailserver -config config.yaml
```

## Configuration

Create a `config.yaml` file:

```yaml
server:
  hostname: "mail.yourdomain.com"
  listen_address: "0.0.0.0:587"
  
api:
  listen_address: "127.0.0.1:8080"
  auth_token: "your-secret-token"

queue:
  max_retry: 5
  retry_delay: "5m"

delivery:
  workers: 20
  dns_cache_ttl: "5m"
```

See [config/example.yaml](config/example.yaml) for all options.

## API Usage

### Send Email

```bash
curl -X POST http://localhost:8080/send \
  -H "Authorization: Bearer your-secret-token" \
  -H "Content-Type: application/json" \
  -d '{
    "from": "app@yourdomain.com",
    "to": ["user@example.com"],
    "subject": "Hello",
    "body": "This is a test email"
  }'
```

### Check Status

```bash
curl http://localhost:8080/status/email-id \
  -H "Authorization: Bearer your-secret-token"
```

## Integration Examples

### Go

```go
import "github.com/yourusername/simple-email-server/pkg/client"

client := client.New("http://localhost:8080", "your-secret-token")
err := client.Send(&client.Email{
    From:    "app@yourdomain.com",
    To:      []string{"user@example.com"},
    Subject: "Test",
    Body:    "Hello from Go!",
})
```

### Python

```python
import requests

requests.post(
    "http://localhost:8080/send",
    headers={"Authorization": "Bearer your-secret-token"},
    json={
        "from": "app@yourdomain.com",
        "to": ["user@example.com"],
        "subject": "Test",
        "body": "Hello from Python!"
    }
)
```

### Node.js

```javascript
fetch('http://localhost:8080/send', {
    method: 'POST',
    headers: {
        'Authorization': 'Bearer your-secret-token',
        'Content-Type': 'application/json'
    },
    body: JSON.stringify({
        from: 'app@yourdomain.com',
        to: ['user@example.com'],
        subject: 'Test',
        body: 'Hello from Node.js!'
    })
})
```

## DNS Configuration

Configure these DNS records for your domain:

```
# A Record (required)
mail.yourdomain.com    A    YOUR_SERVER_IP

# SPF Record (required)
yourdomain.com    TXT    "v=spf1 ip4:YOUR_SERVER_IP ~all"

# MX Record (optional - for receiving)
yourdomain.com    MX    10    mail.yourdomain.com

# Reverse DNS (contact your host)
YOUR_SERVER_IP    PTR    mail.yourdomain.com
```

## Performance

- **Throughput**: 500+ emails/second
- **Latency**: <100ms API response time
- **Queue**: Handles 100k+ queued emails
- **Memory**: ~100MB for 10k emails

## Development

```bash
# Run tests
go test ./...

# Run benchmarks
go test -bench=. ./...

# Build with race detector
go build -race ./cmd/emailserver

# Generate mocks
go generate ./...
```

See [CLAUDE.md](CLAUDE.md) for development guidelines.

## Deployment

- **Docker**: See [docker-compose.yml](docker-compose.yml)
- **Kamal**: See [KAMAL.md](KAMAL.md)
- **Manual**: See [DEPLOYMENT.md](DEPLOYMENT.md)
- **Kubernetes**: See [k8s/](k8s/)

## Monitoring

### Metrics

Prometheus metrics available at `/metrics`:

- `emailserver_emails_sent_total`
- `emailserver_emails_failed_total`
- `emailserver_queue_depth`
- `emailserver_delivery_duration_seconds`

### Health Check

```bash
curl http://localhost:8080/health
```

## Security

- ✅ TLS/STARTTLS encryption
- ✅ API authentication
- ✅ Rate limiting
- ✅ IP allowlisting
- ✅ Automatic SPF checking

See [SECURITY.md](SECURITY.md) for best practices.

## Troubleshooting

### Emails going to spam?
1. Verify SPF record
2. Set up reverse DNS
3. Start with low volume
4. Monitor IP reputation

### Port 25 blocked?
Use port 587 with STARTTLS instead.

### High memory usage?
Reduce `queue.max_size` and `delivery.workers`.

See [TROUBLESHOOTING.md](TROUBLESHOOTING.md) for more.

## Contributing

1. Fork the repository
2. Create your feature branch
3. Write tests (TDD approach)
4. Ensure tests pass
5. Update documentation
6. Submit pull request

See [CONTRIBUTING.md](CONTRIBUTING.md) for details.

## License

MIT License - see [LICENSE](LICENSE)

## Support

- 📧 Email: support@yourdomain.com
- 💬 Discord: [Join our server](https://discord.gg/example)
- 🐛 Issues: [GitHub Issues](https://github.com/yourusername/simple-email-server/issues)

## Roadmap

- [ ] DKIM signing
- [ ] Webhook notifications
- [ ] Template system
- [ ] Web UI dashboard
- [ ] Bounce handling
- [ ] Multiple domain support

## Acknowledgments

Built with Go and these excellent libraries:
- [emersion/go-smtp](https://github.com/emersion/go-smtp)
- [spf13/viper](https://github.com/spf13/viper)
- [prometheus/client_golang](https://github.com/prometheus/client_golang)