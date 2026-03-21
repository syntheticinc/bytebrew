---
title: "Admin Dashboard: Login"
description: Access and authenticate with the ByteBrew Engine Admin Dashboard.
---

The Admin Dashboard is a web-based interface for managing all aspects of your ByteBrew Engine. It is protected by username/password authentication, and access credentials are configured through environment variables.

## Accessing the Dashboard

- Navigate to `http://localhost:8443/admin` in your browser (default URL).
- Enter the credentials configured via `ADMIN_USER` and `ADMIN_PASSWORD` environment variables.
- On successful login, a JWT token is issued with a 24-hour expiration.
- The token is stored in `localStorage` and sent automatically with all API requests.

```bash
# Set credentials in your docker-compose.yml or .env file
ADMIN_USER=admin
ADMIN_PASSWORD=your-secure-password

# The dashboard is served at:
# http://localhost:8443/admin
```

## Security recommendations

- **Change default credentials** -- never use "admin/admin" in production.
- **Use HTTPS** -- put a reverse proxy (Caddy, nginx) in front of the engine with TLS.
- **Network isolation** -- restrict dashboard access to internal networks or VPN.
- **Token expiration** -- tokens expire after 24 hours. Re-login is required after expiration.

:::caution[No multi-user support yet]
The current Admin Dashboard supports a single admin user. Multi-user support with role-based access control is planned for a future release. For team access, share the admin credentials securely or use API keys with scoped permissions.
:::

## Troubleshooting

- **Login fails with correct credentials** -- verify `ADMIN_USER` and `ADMIN_PASSWORD` are set and the engine was restarted after changing them.
- **Dashboard returns 401 after a while** -- your JWT token expired. Reload the page to trigger a re-login.
- **Dashboard not loading** -- check that port 8443 is exposed in Docker and not blocked by a firewall.

---

## What's next

- [Agents](/admin/agents/)
- [Models](/admin/models/)
- [API Keys](/admin/api-keys/)
