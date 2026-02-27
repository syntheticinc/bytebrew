#!/bin/bash
set -euo pipefail

if [ "$(id -u)" -ne 0 ]; then
  echo "This script must be run as root (or with sudo)"
  exit 1
fi

echo "=== Vector Cloud VPS Setup (Ubuntu 24.04) ==="

# Install PostgreSQL and Caddy
apt-get update
apt-get install -y postgresql postgresql-contrib caddy openssl

# Create PostgreSQL user and database
DB_PASSWORD=$(openssl rand -hex 16)
sudo -u postgres createuser vector
sudo -u postgres createdb -O vector vector
sudo -u postgres psql -c "ALTER USER vector WITH PASSWORD '$DB_PASSWORD';"
echo "PostgreSQL: user 'vector' and database 'vector' created"
echo "  Database password: $DB_PASSWORD (save this for config.yaml)"

# Create system user for the service (no login shell)
useradd -r -s /usr/sbin/nologin vector
echo "System user 'vector' created"

# Create directories
mkdir -p /etc/vector /var/www/vector-cloud-web
chown vector:vector /etc/vector
chmod 750 /etc/vector
echo "Directories created: /etc/vector, /var/www/vector-cloud-web"

# Create deploy user for CI/CD
useradd -m -s /bin/bash deploy
echo "Deploy user created"

# Copy systemd unit
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cp "$SCRIPT_DIR/vector-cloud-api.service" /etc/systemd/system/
systemctl daemon-reload
systemctl enable vector-cloud-api
echo "Systemd service installed and enabled"

# Copy Caddyfile
cp "$SCRIPT_DIR/Caddyfile" /etc/caddy/Caddyfile
systemctl reload caddy
echo "Caddyfile installed"

echo ""
echo "=== Setup complete ==="
echo ""
echo "Next steps:"
echo "  1. Copy config.production.yaml to /etc/vector/config.yaml and fill in all CHANGE_ME values"
echo "  2. Generate license key: go run ./cmd/keygen"
echo "  3. Set the DB password in config.yaml to match the one set above"
echo "  4. Setup Stripe prices: STRIPE_SECRET_KEY=sk_... go run ./cmd/stripe-setup"
echo "  5. Configure deploy user sudoers (restricted to deploy operations only):"
echo "     cat > /etc/sudoers.d/deploy << 'SUDOERS'"
echo "deploy ALL=(ALL) NOPASSWD: /bin/systemctl stop vector-cloud-api, /bin/systemctl start vector-cloud-api, /bin/mv /tmp/vector-deploy/build/vector-cloud-api /usr/local/bin/vector-cloud-api, /bin/chmod +x /usr/local/bin/vector-cloud-api, /bin/rm -rf /var/www/vector-cloud-web, /bin/mv /tmp/vector-deploy/vector-cloud-web/dist /var/www/vector-cloud-web"
echo "SUDOERS"
echo "     chmod 440 /etc/sudoers.d/deploy"
echo "  6. Add deploy user SSH key:"
echo "     sudo mkdir -p /home/deploy/.ssh"
echo "     echo 'YOUR_PUBLIC_KEY' | sudo tee /home/deploy/.ssh/authorized_keys"
echo "     sudo chown -R deploy:deploy /home/deploy/.ssh"
echo "     sudo chmod 700 /home/deploy/.ssh && sudo chmod 600 /home/deploy/.ssh/authorized_keys"
echo "  7. Point DNS: vector.yourdomain.com -> VPS IP"
echo "  8. Start the service: sudo systemctl start vector-cloud-api"
