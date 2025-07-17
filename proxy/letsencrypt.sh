#!/bin/bash

# Let's Encrypt certificate management script for RCode proxy server
# This script helps with obtaining and renewing Let's Encrypt certificates

set -e

# Configuration
DOMAIN="${DOMAIN:-}"
EMAIL="${EMAIL:-}"
WEBROOT="${WEBROOT:-/var/www/letsencrypt}"
CERT_PATH="/etc/letsencrypt/live/${DOMAIN}/fullchain.pem"
KEY_PATH="/etc/letsencrypt/live/${DOMAIN}/privkey.pem"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Print usage
usage() {
    echo "Usage: $0 [command]"
    echo ""
    echo "Commands:"
    echo "  setup       - Initial setup and certificate request"
    echo "  renew       - Renew existing certificate"
    echo "  status      - Check certificate status"
    echo "  cron        - Setup automatic renewal via cron"
    echo ""
    echo "Environment variables:"
    echo "  DOMAIN      - Your domain name (required for setup)"
    echo "  EMAIL       - Your email for Let's Encrypt notifications (required for setup)"
    echo "  WEBROOT     - Web root for HTTP challenge (default: /var/www/letsencrypt)"
    echo ""
    echo "Example:"
    echo "  DOMAIN=proxy.example.com EMAIL=admin@example.com $0 setup"
}

# Check if certbot is installed
check_certbot() {
    if ! command -v certbot &> /dev/null; then
        echo -e "${RED}Error: certbot is not installed${NC}"
        echo "Please install certbot first:"
        echo "  Ubuntu/Debian: sudo apt-get install certbot"
        echo "  CentOS/RHEL: sudo yum install certbot"
        echo "  macOS: brew install certbot"
        exit 1
    fi
}

# Initial setup
setup() {
    if [ -z "$DOMAIN" ] || [ -z "$EMAIL" ]; then
        echo -e "${RED}Error: DOMAIN and EMAIL environment variables are required${NC}"
        usage
        exit 1
    fi

    echo -e "${GREEN}Setting up Let's Encrypt for domain: $DOMAIN${NC}"
    
    # Create webroot directory
    echo "Creating webroot directory: $WEBROOT"
    sudo mkdir -p "$WEBROOT"
    
    # Request certificate using webroot plugin
    echo "Requesting certificate..."
    sudo certbot certonly \
        --webroot \
        --webroot-path "$WEBROOT" \
        --email "$EMAIL" \
        --agree-tos \
        --no-eff-email \
        --domains "$DOMAIN"
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}Certificate obtained successfully!${NC}"
        echo ""
        echo "Certificate path: $CERT_PATH"
        echo "Private key path: $KEY_PATH"
        echo ""
        echo "To run the proxy with HTTPS:"
        echo "  go run proxy.go -cert=\"$CERT_PATH\" -key=\"$KEY_PATH\""
        echo ""
        echo "Or with HTTP to HTTPS redirect:"
        echo "  go run proxy.go -cert=\"$CERT_PATH\" -key=\"$KEY_PATH\" -tls-only"
    else
        echo -e "${RED}Failed to obtain certificate${NC}"
        exit 1
    fi
}

# Renew certificate
renew() {
    echo -e "${GREEN}Attempting to renew certificates...${NC}"
    
    # Test renewal
    sudo certbot renew --dry-run
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}Dry run successful. Performing actual renewal...${NC}"
        sudo certbot renew
        
        # Reload proxy server if running via systemd
        if systemctl is-active --quiet rcode-proxy; then
            echo "Reloading rcode-proxy service..."
            sudo systemctl reload rcode-proxy
        fi
    else
        echo -e "${RED}Certificate renewal test failed${NC}"
        exit 1
    fi
}

# Check certificate status
status() {
    echo -e "${GREEN}Checking certificate status...${NC}"
    
    if [ -z "$DOMAIN" ]; then
        # List all certificates
        sudo certbot certificates
    else
        # Check specific certificate
        if [ -f "$CERT_PATH" ]; then
            echo "Certificate found for $DOMAIN"
            echo ""
            # Show certificate details
            openssl x509 -in "$CERT_PATH" -noout -dates
        else
            echo -e "${YELLOW}No certificate found for $DOMAIN${NC}"
        fi
    fi
}

# Setup automatic renewal via cron
setup_cron() {
    echo -e "${GREEN}Setting up automatic renewal via cron...${NC}"
    
    # Create renewal script
    SCRIPT_PATH="/usr/local/bin/renew-rcode-proxy-cert.sh"
    
    sudo tee "$SCRIPT_PATH" > /dev/null << 'EOF'
#!/bin/bash
# Renew Let's Encrypt certificate and reload proxy

certbot renew --quiet

# Reload proxy server if running via systemd
if systemctl is-active --quiet rcode-proxy; then
    systemctl reload rcode-proxy
fi
EOF
    
    sudo chmod +x "$SCRIPT_PATH"
    
    # Add cron job (twice daily as recommended by Let's Encrypt)
    CRON_JOB="0 0,12 * * * $SCRIPT_PATH"
    
    # Check if cron job already exists
    if crontab -l 2>/dev/null | grep -q "$SCRIPT_PATH"; then
        echo -e "${YELLOW}Cron job already exists${NC}"
    else
        # Add cron job
        (crontab -l 2>/dev/null; echo "$CRON_JOB") | crontab -
        echo -e "${GREEN}Cron job added successfully${NC}"
        echo "Certificate will be checked for renewal twice daily"
    fi
}

# Main script logic
check_certbot

case "${1:-}" in
    setup)
        setup
        ;;
    renew)
        renew
        ;;
    status)
        status
        ;;
    cron)
        setup_cron
        ;;
    *)
        usage
        exit 1
        ;;
esac