# RCode Proxy Server with HTTPS Support

This proxy server forwards requests to the Anthropic API and now supports HTTPS with Let's Encrypt certificates.

## Features

- HTTP and HTTPS support
- Automatic HTTP to HTTPS redirect option
- Let's Encrypt certificate integration
- Server-sent events (SSE) streaming support
- Preserves all headers including OAuth tokens

## Quick Start

### HTTP Mode (Default)

```bash
go run proxy.go
```

The proxy will run on port 8001 by default.

### HTTPS Mode with Let's Encrypt

1. **Obtain a Let's Encrypt certificate:**

```bash
# Install certbot if not already installed
sudo apt-get install certbot  # Ubuntu/Debian
# or
sudo yum install certbot       # CentOS/RHEL
# or
brew install certbot           # macOS

# Get certificate using the provided script
DOMAIN=proxy.yourdomain.com EMAIL=you@example.com ./letsencrypt.sh setup
```

2. **Run the proxy with HTTPS:**

```bash
go run proxy.go \
  -cert="/etc/letsencrypt/live/proxy.yourdomain.com/fullchain.pem" \
  -key="/etc/letsencrypt/live/proxy.yourdomain.com/privkey.pem"
```

3. **With HTTP to HTTPS redirect:**

```bash
go run proxy.go \
  -cert="/etc/letsencrypt/live/proxy.yourdomain.com/fullchain.pem" \
  -key="/etc/letsencrypt/live/proxy.yourdomain.com/privkey.pem" \
  -tls-only
```

## Command Line Options

- `-http` - HTTP address to listen on (default: ":8001")
- `-https` - HTTPS address to listen on (default: ":8443")
- `-cert` - Path to TLS certificate file
- `-key` - Path to TLS private key file
- `-tls-only` - Only serve over HTTPS, redirect HTTP to HTTPS

## Building and Deploying

1. **Build the binary:**

```bash
cd proxy
go build -o proxy
```

2. **Deploy to server:**

```bash
# Copy binary to server
scp proxy your-server:/opt/rcode-proxy/

# Copy systemd service file
scp rcode-proxy.service your-server:/tmp/

# On the server:
sudo mv /tmp/rcode-proxy.service /etc/systemd/system/
sudo systemctl daemon-reload
```

3. **Configure the service:**

Edit `/etc/systemd/system/rcode-proxy.service` and replace `YOUR_DOMAIN` with your actual domain.

4. **Start the service:**

```bash
sudo systemctl enable rcode-proxy
sudo systemctl start rcode-proxy
```

## Certificate Management

### Automatic Renewal

Set up automatic certificate renewal:

```bash
./letsencrypt.sh cron
```

This will check for certificate renewal twice daily and reload the proxy if needed.

### Manual Renewal

```bash
./letsencrypt.sh renew
```

### Check Certificate Status

```bash
./letsencrypt.sh status
```

## Using the Proxy

Once running, configure your RCode client to use the proxy:

```bash
# For HTTP
MSG_PROXY=http://proxy.yourdomain.com:8001 go run main.go

# For HTTPS
MSG_PROXY=https://proxy.yourdomain.com go run main.go
```

## Security Notes

- Always use HTTPS in production environments
- The proxy preserves all headers, including authentication tokens
- Consider implementing rate limiting and access controls
- Keep your Let's Encrypt certificates up to date with automatic renewal

## Troubleshooting

1. **Certificate not found:** Ensure the certificate paths are correct and the user has read permissions.

2. **Port 443 permission denied:** Run with sudo or use a higher port (e.g., 8443) and configure a reverse proxy.

3. **Certificate renewal fails:** Check that port 80 is accessible for the HTTP challenge.

## Let's Encrypt Helper Script

The `letsencrypt.sh` script provides easy certificate management:

- `setup` - Initial certificate request
- `renew` - Renew existing certificate
- `status` - Check certificate status
- `cron` - Setup automatic renewal

Example:
```bash
DOMAIN=proxy.example.com EMAIL=admin@example.com ./letsencrypt.sh setup
```