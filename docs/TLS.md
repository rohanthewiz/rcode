# TLS/HTTPS Configuration for RCode

This document describes how to configure and use TLS/HTTPS with the RCode server.

## Quick Start

### 1. Generate Self-Signed Certificates (Development)

For local development, you can generate self-signed certificates:

```bash
cd scripts
./generate-certs.sh
```

This will create:
- `certs/localhost.crt` - The certificate file
- `certs/localhost.key` - The private key file

**Note**: Browsers will show a security warning for self-signed certificates. You'll need to accept the warning to proceed.

### 2. Enable TLS

Set the environment variable to enable TLS:

```bash
export RCODE_TLS_ENABLED=true
```

### 3. Run the Server

```bash
go run main.go
```

The server will now:
- Serve HTTPS on port 8443 (by default)
- Redirect HTTP traffic from port 8000 to HTTPS

## Configuration Options

You can configure TLS using environment variables:

| Environment Variable | Description | Default |
|---------------------|-------------|---------|
| `RCODE_TLS_ENABLED` | Enable TLS (set to "true") | false |
| `RCODE_TLS_PORT` | HTTPS port | :8443 |
| `RCODE_TLS_CERT` | Path to certificate file | certs/localhost.crt |
| `RCODE_TLS_KEY` | Path to private key file | certs/localhost.key |

## Production Certificates

For production use, you should obtain proper certificates from a Certificate Authority (CA):

### Option 1: Let's Encrypt (Free)

Use [certbot](https://certbot.eff.org/) to obtain free certificates:

```bash
certbot certonly --standalone -d yourdomain.com
```

Then set the paths:
```bash
export RCODE_TLS_CERT=/etc/letsencrypt/live/yourdomain.com/fullchain.pem
export RCODE_TLS_KEY=/etc/letsencrypt/live/yourdomain.com/privkey.pem
```

### Option 2: Commercial Certificate

1. Generate a Certificate Signing Request (CSR):
   ```bash
   openssl req -new -newkey rsa:4096 -nodes -keyout server.key -out server.csr
   ```

2. Submit the CSR to your Certificate Authority

3. Once you receive the certificate, configure the paths:
   ```bash
   export RCODE_TLS_CERT=/path/to/your/certificate.crt
   export RCODE_TLS_KEY=/path/to/your/private.key
   ```

## Local Development with mkcert

For a better local development experience without browser warnings, use [mkcert](https://github.com/FiloSottile/mkcert):

1. Install mkcert:
   ```bash
   brew install mkcert  # macOS
   # or see https://github.com/FiloSottile/mkcert for other platforms
   ```

2. Install the local CA:
   ```bash
   mkcert -install
   ```

3. Generate certificates:
   ```bash
   cd certs
   mkcert localhost 127.0.0.1 ::1
   ```

4. Update the certificate paths:
   ```bash
   export RCODE_TLS_CERT=certs/localhost+2.pem
   export RCODE_TLS_KEY=certs/localhost+2-key.pem
   ```

## Testing HTTPS

Once TLS is enabled:

1. Access the HTTPS endpoint:
   ```
   https://localhost:8443
   ```

2. Verify HTTP redirect:
   ```
   curl -I http://localhost:8000
   ```
   Should return a 301 redirect to HTTPS.

## Security Notes

1. **Never commit certificates or private keys** to version control
2. **Use strong key sizes** (minimum 2048-bit RSA, preferably 4096-bit)
3. **Keep private keys secure** with appropriate file permissions:
   ```bash
   chmod 600 certs/*.key
   ```
4. **Regularly update certificates** before they expire
5. **Use TLS 1.2 or higher** (enforced by default in rweb)

## Troubleshooting

### Certificate Errors

If you see certificate errors:
1. Check that the certificate and key files exist at the specified paths
2. Verify the certificate matches the private key:
   ```bash
   openssl x509 -noout -modulus -in cert.crt | openssl md5
   openssl rsa -noout -modulus -in private.key | openssl md5
   ```
   The output should match.

### Port Already in Use

If port 8443 is already in use, change it:
```bash
export RCODE_TLS_PORT=:8543
```

### Permission Denied on Port 443

To use the standard HTTPS port (443), you need elevated privileges:
```bash
sudo go run main.go
```

Or use a reverse proxy like nginx to forward from port 443 to your application port.