#!/bin/bash

# Script to generate self-signed certificates for local development
# These certificates should NOT be used in production

CERT_DIR="../certs"
DAYS=365

# Create certs directory if it doesn't exist
mkdir -p "$CERT_DIR"

# Generate a self-signed certificate
openssl req -x509 \
    -newkey rsa:4096 \
    -keyout "$CERT_DIR/localhost.key" \
    -out "$CERT_DIR/localhost.crt" \
    -days $DAYS \
    -nodes \
    -subj "/C=US/ST=State/L=City/O=RCode/CN=localhost" \
    -addext "subjectAltName=DNS:localhost,IP:127.0.0.1"

echo "Self-signed certificate generated:"
echo "  Certificate: $CERT_DIR/localhost.crt"
echo "  Private Key: $CERT_DIR/localhost.key"
echo ""
echo "Note: This is a self-signed certificate for development only."
echo "Browsers will show a security warning that you'll need to accept."