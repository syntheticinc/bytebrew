#!/bin/bash
set -euo pipefail

if [ "$(id -u)" -ne 0 ]; then
  echo "This script must be run as root (or with sudo)"
  exit 1
fi

echo "=== ByteBrew Cloud VPS Setup (Ubuntu 24.04) ==="

# Install PostgreSQL and Caddy
apt-get update
apt-get install -y postgresql postgresql-contrib caddy openssl

# Create PostgreSQL user and database
DB_PASSWORD=$(openssl rand -hex 16)
sudo -u postgres createuser bytebrew
sudo -u postgres createdb -O bytebrew bytebrew
sudo -u postgres psql -c "ALTER USER bytebrew WITH PASSWORD '$DB_PASSWORD';"
echo "PostgreSQL: user 'bytebrew' and database 'bytebrew' created"
echo "  Database password: $DB_PASSWORD (save this for config.yaml)"

# Create system user for the service (no login shell)
useradd -r -s /usr/sbin/nologin bytebrew
echo "System user 'bytebrew' created"

# Create directories
mkdir -p /etc/bytebrew /var/www/bytebrew-cloud-web
chown bytebrew:bytebrew /etc/bytebrew
chmod 750 /etc/bytebrew
echo "Directories created: /etc/bytebrew, /var/www/bytebrew-cloud-web"

# Create deploy user for CI/CD
useradd -m -s /bin/bash deploy
echo "Deploy user created"

# Copy systemd units
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cp "$SCRIPT_DIR/bytebrew-cloud-api.service" /etc/systemd/system/
cp "$SCRIPT_DIR/bytebrew-bridge.service" /etc/systemd/system/
systemctl daemon-reload
systemctl enable bytebrew-cloud-api
systemctl enable bytebrew-bridge
echo "Systemd services installed and enabled (bytebrew-cloud-api, bytebrew-bridge)"

# Create bridge environment file
BRIDGE_AUTH_TOKEN=$(openssl rand -hex 32)
cat > /etc/bytebrew/bridge.env << EOF
BRIDGE_PORT=8443
BRIDGE_AUTH_TOKEN=$BRIDGE_AUTH_TOKEN
EOF
chown bytebrew:bytebrew /etc/bytebrew/bridge.env
chmod 640 /etc/bytebrew/bridge.env
echo "Bridge environment file created: /etc/bytebrew/bridge.env"
echo "  Auth token: $BRIDGE_AUTH_TOKEN (save this for CLI config)"

# Copy Caddyfile
cp "$SCRIPT_DIR/Caddyfile" /etc/caddy/Caddyfile
systemctl reload caddy
echo "Caddyfile installed"

# Configure UFW firewall
echo "Configuring UFW firewall..."
ufw default deny incoming
ufw default allow outgoing
ufw allow ssh
ufw allow 80/tcp
ufw allow 443/tcp
ufw --force enable
echo "UFW configured and enabled (ssh, 80, 443)"

echo ""
echo "=== Setup complete ==="
echo ""
echo "Next steps:"
echo "  1. Generate license key: go run ./cmd/keygen"
echo "  2. Setup Stripe prices: STRIPE_SECRET_KEY=sk_... go run ./cmd/stripe-setup"
echo "  3. Add all secrets to GitHub repo Settings > Secrets and variables > Actions:"
echo "     DATABASE_URL, AUTH_JWT_SECRET, LICENSE_PRIVATE_KEY_HEX,"
echo "     STRIPE_SECRET_KEY, STRIPE_WEBHOOK_SECRET, STRIPE_PRICES_*,"
echo "     DEEPINFRA_API_KEY, EMAIL_RESEND_API_KEY, BRIDGE_AUTH_TOKEN,"
echo "     VPS_HOST, VPS_SSH_KEY"
echo "  4. Configure deploy user sudoers (restricted to deploy operations only):"
echo "     cat > /etc/sudoers.d/deploy << 'SUDOERS'"
echo "deploy ALL=(ALL) NOPASSWD: \\"
echo "  /bin/systemctl stop bytebrew-cloud-api, \\"
echo "  /bin/systemctl start bytebrew-cloud-api, \\"
echo "  /bin/systemctl stop bytebrew-bridge, \\"
echo "  /bin/systemctl start bytebrew-bridge, \\"
echo "  /bin/cp /usr/local/bin/bytebrew-cloud-api /usr/local/bin/bytebrew-cloud-api.bak, \\"
echo "  /bin/mv /tmp/bytebrew-deploy/build/bytebrew-cloud-api /usr/local/bin/bytebrew-cloud-api, \\"
echo "  /bin/mv /usr/local/bin/bytebrew-cloud-api.bak /usr/local/bin/bytebrew-cloud-api, \\"
echo "  /bin/chmod +x /usr/local/bin/bytebrew-cloud-api, \\"
echo "  /bin/mv /tmp/bytebrew-deploy/build/bytebrew-bridge /usr/local/bin/bytebrew-bridge, \\"
echo "  /bin/chmod +x /usr/local/bin/bytebrew-bridge, \\"
echo "  /bin/cp /usr/local/bin/bytebrew-bridge /usr/local/bin/bytebrew-bridge.bak, \\"
echo "  /bin/mv /usr/local/bin/bytebrew-bridge.bak /usr/local/bin/bytebrew-bridge, \\"
echo "  /bin/rm -f /usr/local/bin/bytebrew-bridge.bak, \\"
echo "  /bin/rm -rf /var/www/bytebrew-cloud-web, \\"
echo "  /bin/rm -f /usr/local/bin/bytebrew-cloud-api.bak, \\"
echo "  /bin/mv /tmp/bytebrew-deploy/bytebrew-cloud-web/dist /var/www/bytebrew-cloud-web, \\"
echo "  /bin/cp /tmp/bytebrew-deploy/deploy/config.production.yaml /etc/bytebrew/config.yaml, \\"
echo "  /usr/bin/tee /etc/bytebrew/cloud-api.env, \\"
echo "  /usr/bin/tee /etc/bytebrew/bridge.env, \\"
echo "  /bin/chown bytebrew:bytebrew /etc/bytebrew/cloud-api.env, \\"
echo "  /bin/chown bytebrew:bytebrew /etc/bytebrew/bridge.env, \\"
echo "  /bin/chown bytebrew:bytebrew /etc/bytebrew/config.yaml, \\"
echo "  /bin/chmod 640 /etc/bytebrew/cloud-api.env, \\"
echo "  /bin/chmod 640 /etc/bytebrew/bridge.env, \\"
echo "  /bin/chmod 640 /etc/bytebrew/config.yaml, \\"
echo "  /bin/journalctl -u bytebrew-cloud-api *, \\"
echo "  /bin/journalctl -u bytebrew-bridge *"
echo "SUDOERS"
echo "     chmod 440 /etc/sudoers.d/deploy"
echo "  5. Add deploy user SSH key:"
echo "     sudo mkdir -p /home/deploy/.ssh"
echo "     echo 'YOUR_PUBLIC_KEY' | sudo tee /home/deploy/.ssh/authorized_keys"
echo "     sudo chown -R deploy:deploy /home/deploy/.ssh"
echo "     sudo chmod 700 /home/deploy/.ssh && sudo chmod 600 /home/deploy/.ssh/authorized_keys"
echo "  6. Point DNS records for bytebrew.ai:"
echo "     app.bytebrew.ai    -> VPS_IP (A record)"
echo "     api.bytebrew.ai    -> VPS_IP (A record)"
echo "     bridge.bytebrew.ai -> VPS_IP (A record)"
echo "     bytebrew.ai        -> VPS_IP (A record)"
echo "  7. Start the services:"
echo "     sudo systemctl start bytebrew-cloud-api"
echo "     sudo systemctl start bytebrew-bridge"
