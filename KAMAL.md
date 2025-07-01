# Kamal Deployment Guide

This guide shows how to deploy the Simple Email Server using [Kamal](https://kamal-deploy.org/), a deployment tool that makes it easy to deploy containerized applications.

## Prerequisites

- A server with Ubuntu/Debian (fresh VPS recommended)
- Docker registry account (Docker Hub, GitHub Container Registry, etc.)
- Kamal installed locally: `gem install kamal`
- SSH access to your server

## Step 1: Kamal Configuration

Create `config/deploy.yml` in your project root:

```yaml
# config/deploy.yml
service: simple-email-server

image: yourusername/simple-email-server

servers:
  web:
    - YOUR_SERVER_IP
    options:
      network: "host"  # Required for SMTP port binding

registry:
  username: YOUR_REGISTRY_USERNAME
  password:
    - DOCKER_REGISTRY_PASSWORD

env:
  clear:
    SERVER_HOSTNAME: mail.yourdomain.com
    API_LISTEN_ADDRESS: "127.0.0.1:8080"
    SMTP_LISTEN_ADDRESS: "0.0.0.0:587"
  secret:
    - API_AUTH_TOKEN

accessories:
  data:
    image: busybox
    host: YOUR_SERVER_IP
    volumes:
      - "/var/email-server/data:/data"
      - "/var/email-server/config:/config"
    cmd: sleep infinity

builder:
  multiarch: false
  dockerfile: Dockerfile.production

traefik:
  options:
    publish:
      - "443:443"
    volume:
      - "/var/email-server/certs:/certs"
  args:
    entryPoints.websecure.address: ":443"
    entryPoints.smtp.address: ":587"
    certificatesResolvers.letsencrypt.acme.email: "admin@yourdomain.com"
    certificatesResolvers.letsencrypt.acme.storage: "/certs/acme.json"
    certificatesResolvers.letsencrypt.acme.httpChallenge.entryPoint: "web"

healthcheck:
  path: /health
  port: 8080
  interval: 10s
```

## Step 2: Create Production Dockerfile

Create `Dockerfile.production`:

```dockerfile
# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -o emailserver cmd/emailserver/main.go

# Production stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/emailserver .

# Copy default config
COPY config/production.yaml /app/config/config.yaml

# Create data directory
RUN mkdir -p /data /config

EXPOSE 587 8080

CMD ["./emailserver", "-config", "/config/config.yaml"]
```

## Step 3: Environment Configuration

Create `.env` file for secrets:

```bash
# .env
DOCKER_REGISTRY_PASSWORD=your-registry-password
API_AUTH_TOKEN=your-secret-api-token
```

## Step 4: Server Preparation

SSH into your server and prepare it:

```bash
# Update system
sudo apt update && sudo apt upgrade -y

# Open required ports
sudo ufw allow 22/tcp    # SSH
sudo ufw allow 587/tcp   # SMTP
sudo ufw allow 443/tcp   # HTTPS
sudo ufw enable

# Create directories
sudo mkdir -p /var/email-server/{data,config,certs}
sudo chmod 755 /var/email-server
```

## Step 5: DNS Configuration

Before deploying, configure your DNS:

```
# A Record
mail.yourdomain.com    A    YOUR_SERVER_IP

# SPF Record
yourdomain.com    TXT    "v=spf1 ip4:YOUR_SERVER_IP ~all"

# MX Record (if receiving mail)
yourdomain.com    MX    10    mail.yourdomain.com
```

## Step 6: Deploy with Kamal

```bash
# Setup (first time only)
kamal setup

# Deploy
kamal deploy

# Check status
kamal app details

# View logs
kamal app logs

# Run commands in container
kamal app exec 'emailserver -version'
```

## Step 7: Configuration Management

Upload your production config:

```bash
# Create production config locally
cat > config/production.yaml << EOF
server:
  hostname: "mail.yourdomain.com"
  listen_address: "0.0.0.0:587"
  tls:
    enabled: true
    auto_tls: true  # Uses Let's Encrypt

api:
  listen_address: "127.0.0.1:8080"
  auth_token: "${API_AUTH_TOKEN}"

queue:
  storage_path: "/data/queue"
  max_retry: 5
  retry_delay: "5m"

delivery:
  workers: 20
  dns_cache_ttl: "5m"
  
limits:
  max_recipients: 100
  max_message_size: "25MB"
  rate_limit: "100/minute"

logging:
  level: "info"
  file: "/data/logs/emailserver.log"
EOF

# Upload to server
scp config/production.yaml root@YOUR_SERVER_IP:/var/email-server/config/config.yaml

# Restart to apply
kamal app restart
```

## Common Kamal Commands

```bash
# Deploy updates
kamal deploy

# Rollback
kamal rollback

# View running containers
kamal app containers

# Stop application
kamal app stop

# Start application
kamal app start

# Remove application
kamal app remove

# Run database migrations (if needed)
kamal app exec 'emailserver migrate'

# Interactive shell
kamal app exec -i bash
```

## Monitoring and Maintenance

### Check Email Queue
```bash
kamal app exec 'emailserver queue status'
```

### View Metrics
```bash
# Prometheus metrics endpoint
curl -H "Authorization: Bearer $API_AUTH_TOKEN" http://YOUR_SERVER_IP:8080/metrics
```

### Backup Data
```bash
# On server
sudo tar -czf email-backup-$(date +%Y%m%d).tar.gz /var/email-server/data
```

## Troubleshooting

### Port Already in Use
```bash
# Check what's using port 587
sudo lsof -i :587

# Kill process if needed
sudo kill -9 PID
```

### Permission Issues
```bash
# Fix permissions
sudo chown -R 1000:1000 /var/email-server
```

### TLS Certificate Issues
```bash
# Check certificate
kamal app exec 'emailserver cert check'

# Force renewal
kamal app exec 'emailserver cert renew'
```

## Integration with Your App

If deploying alongside a Rails/Node.js app with Kamal:

```yaml
# In your main app's deploy.yml
accessories:
  email:
    image: yourusername/simple-email-server
    host: YOUR_SERVER_IP
    port: "127.0.0.1:8080:8080"
    env:
      secret:
        - API_AUTH_TOKEN
    volumes:
      - "/var/email-server/data:/data"
```

Then configure your app:
```ruby
# Rails example
Rails.application.config.email_server = {
  url: "http://127.0.0.1:8080",
  token: ENV["API_AUTH_TOKEN"]
}
```

## Security Best Practices

1. **Use Kamal secrets** for sensitive data
2. **Enable automatic TLS** via Let's Encrypt
3. **Restrict API** to localhost only
4. **Set up fail2ban** for SMTP brute force protection
5. **Monitor logs** for suspicious activity

## Next Steps

- Set up monitoring dashboards
- Configure backup automation
- Implement DKIM signing
- Add custom email templates