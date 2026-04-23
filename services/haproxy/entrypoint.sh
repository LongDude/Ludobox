#!/bin/sh
set -eu

TEMPLATES_DIR=/usr/local/etc/haproxy/templates
CONFIG_PATH=/usr/local/etc/haproxy/haproxy.cfg
CERT_DIR=/etc/letsencrypt/live
PEM_DIR=/etc/letsencrypt/haproxy

DOMAIN="${DOMAIN:-localhost}"
FULLCHAIN_PATH="$CERT_DIR/$DOMAIN/fullchain.pem"
PRIVKEY_PATH="$CERT_DIR/$DOMAIN/privkey.pem"
PEM_PATH="$PEM_DIR/$DOMAIN.pem"

mkdir -p "$PEM_DIR"

if [ -f "$FULLCHAIN_PATH" ] && [ -f "$PRIVKEY_PATH" ]; then
  echo "SSL cert found for $DOMAIN, enabling HTTPS"
  cat "$FULLCHAIN_PATH" "$PRIVKEY_PATH" > "$PEM_PATH"
  chmod 600 "$PEM_PATH"
  sed "s|\${HAPROXY_CERT_PEM_PATH}|$PEM_PATH|g" "$TEMPLATES_DIR/https.cfg.template" > "$CONFIG_PATH"
else
  echo "No SSL cert for $DOMAIN, starting HTTP-only ingress"
  cp "$TEMPLATES_DIR/http_only.cfg.template" "$CONFIG_PATH"
fi

exec haproxy -W -db -f "$CONFIG_PATH"
