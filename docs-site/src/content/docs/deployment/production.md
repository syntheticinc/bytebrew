---
title: Production Deployment
description: Deploy ByteBrew Engine on a VPS with Caddy reverse proxy, automatic SSL, systemd, database backups, and monitoring.
---

This guide covers deploying ByteBrew Engine on a VPS with proper TLS, process management, backups, and monitoring. The target stack is Ubuntu + Docker Compose + Caddy (reverse proxy with automatic Let's Encrypt SSL).

## VPS setup

### Requirements

- Ubuntu 22.04+ (or any Linux with Docker support)
- 2+ GB RAM (4 GB recommended for engine + PostgreSQL + Ollama)
- Docker and Docker Compose installed
- A domain name pointed to your server's IP (e.g., `engine.yourdomain.com`)

### Initial setup

```bash
# Update system
apt update && apt upgrade -y

# Install Docker
curl -fsSL https://get.docker.com | sh
systemctl enable docker

# Install Caddy
apt install -y debian-keyring debian-archive-keyring apt-transport-https
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | tee /etc/apt/sources.list.d/caddy-stable.list
apt update && apt install -y caddy
```

## Docker Compose for production

Create `/opt/bytebrew/docker-compose.yml`:

```yaml
version: "3.8"

services:
  engine:
    image: ghcr.io/syntheticinc/bytebrew-engine:latest
    ports:
      - "127.0.0.1:8443:8443"    # Only bind to localhost (Caddy proxies)
      - "127.0.0.1:8443:8443"
    env_file: .env
    environment:
      - DATABASE_URL=postgres://bytebrew:${DB_PASSWORD}@postgres:5432/bytebrew?sslmode=disable
    extra_hosts:
      - "host.docker.internal:host-gateway"    # For Ollama on host
    volumes:
      - ./agents.yaml:/app/agents.yaml:ro
      - ./knowledge:/app/knowledge:ro
      - engine-data:/app/data
    depends_on:
      postgres:
        condition: service_healthy
    restart: unless-stopped

  postgres:
    image: pgvector/pgvector:pg16
    environment:
      - POSTGRES_USER=bytebrew
      - POSTGRES_PASSWORD=${DB_PASSWORD}
      - POSTGRES_DB=bytebrew
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U bytebrew"]
      interval: 5s
      timeout: 3s
      retries: 5
    restart: unless-stopped

volumes:
  pgdata:
  engine-data:
```

Create `/opt/bytebrew/.env`:

```bash
ADMIN_USER=admin
ADMIN_PASSWORD=a-very-secure-password
DB_PASSWORD=another-secure-password
OPENAI_API_KEY=sk-your-key
```

:::caution[Secure your .env]
Set restrictive permissions: `chmod 600 /opt/bytebrew/.env`. This file contains secrets that should not be world-readable.
:::

## Caddy reverse proxy

Caddy provides automatic HTTPS with Let's Encrypt certificates. No manual certificate management required.

Create `/etc/caddy/Caddyfile`:

```
engine.yourdomain.com {
    # API endpoint
    handle /api/* {
        reverse_proxy localhost:8443
    }

    # Webhook endpoints
    handle /webhooks/* {
        reverse_proxy localhost:8443
    }

    # Admin Dashboard
    handle /admin* {
        reverse_proxy localhost:8443
    }

    # Default: API
    handle {
        reverse_proxy localhost:8443
    }
}
```

```bash
# Reload Caddy
systemctl reload caddy

# Verify TLS
curl https://engine.yourdomain.com/api/v1/health
```

## SSL/TLS

Caddy handles TLS automatically:

- Obtains a Let's Encrypt certificate on first request.
- Renews certificates automatically before expiration.
- Redirects HTTP to HTTPS by default.
- Supports HTTP/2 out of the box.

No additional configuration needed. Just ensure port 80 and 443 are open in your firewall:

```bash
ufw allow 80/tcp
ufw allow 443/tcp
```

## systemd service (optional)

If you want Docker Compose managed by systemd for automatic start on boot:

Create `/etc/systemd/system/bytebrew.service`:

```ini
[Unit]
Description=ByteBrew Engine
After=docker.service
Requires=docker.service

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=/opt/bytebrew
ExecStart=/usr/bin/docker compose up -d
ExecStop=/usr/bin/docker compose down

[Install]
WantedBy=multi-user.target
```

```bash
systemctl daemon-reload
systemctl enable bytebrew
systemctl start bytebrew
```

## Database backup strategy

### Automated daily backups

Create `/opt/bytebrew/backup.sh`:

```bash
#!/bin/bash
BACKUP_DIR="/opt/bytebrew/backups"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
mkdir -p "$BACKUP_DIR"

# Dump the database
docker compose -f /opt/bytebrew/docker-compose.yml exec -T postgres \
  pg_dump -U bytebrew bytebrew | gzip > "$BACKUP_DIR/bytebrew_$TIMESTAMP.sql.gz"

# Keep last 30 days
find "$BACKUP_DIR" -name "*.sql.gz" -mtime +30 -delete

echo "Backup completed: bytebrew_$TIMESTAMP.sql.gz"
```

```bash
chmod +x /opt/bytebrew/backup.sh

# Add to crontab (daily at 3 AM)
echo "0 3 * * * /opt/bytebrew/backup.sh" | crontab -
```

### Restoring from backup

```bash
gunzip -c backups/bytebrew_20250319_030000.sql.gz | \
  docker compose exec -T postgres psql -U bytebrew bytebrew
```

## Monitoring

### Health endpoint polling

The `/api/v1/health` endpoint is unauthenticated and designed for monitoring:

```bash
# Simple health check script
curl -sf http://localhost:8443/api/v1/health > /dev/null || echo "ByteBrew Engine is DOWN"
```

### UptimeRobot / Uptime Kuma

Point your monitoring service at:

```
https://engine.yourdomain.com/api/v1/health
```

Expected response: HTTP 200 with `{"status":"ok"}`.

### Log monitoring

```bash
# Follow engine logs
docker compose -f /opt/bytebrew/docker-compose.yml logs -f engine

# Last 100 lines
docker compose -f /opt/bytebrew/docker-compose.yml logs engine --tail 100
```

### Resource monitoring

```bash
# Container resource usage
docker stats bytebrew-engine-1 bytebrew-postgres-1
```

## Security checklist

- [ ] Change default `ADMIN_USER` and `ADMIN_PASSWORD`
- [ ] Use strong, unique `DB_PASSWORD`
- [ ] `.env` file has `chmod 600`
- [ ] Only `127.0.0.1` port binding (Caddy handles external access)
- [ ] Firewall allows only ports 80, 443, and SSH
- [ ] API keys use `${VAR}` in YAML, never hardcoded
- [ ] Webhook triggers have `secret` configured
- [ ] Create scoped API tokens instead of sharing admin tokens
- [ ] Automated database backups running

---

## What's next

- [Docker Deployment](/docs/deployment/docker/)
- [Model Selection](/docs/deployment/model-selection/)
- [API Reference](/docs/getting-started/api-reference/)
