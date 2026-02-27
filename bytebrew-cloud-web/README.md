# ByteBrew Cloud Web

Web portal for managing subscriptions, billing, and teams. SPA built on React 19.

## Stack

- **React 19** + TypeScript 5
- **Vite 6** (build + dev server)
- **TailwindCSS 4**
- **@tanstack/react-router** (type-safe routing)
- **@tanstack/react-query** (data fetching)

## Quick Start

### 1. Install Dependencies

```bash
npm install
```

### 2. Start Dev Server

```bash
npm run dev
```

Dev server starts at `http://localhost:5173`.

> Requires a running ByteBrew Cloud API (default `http://localhost:60402`).

### 3. Build

```bash
npm run build
```

Output in `dist/`. Preview the built bundle:

```bash
npm run preview
```

## Pages

| Path | Page | Description |
|------|------|-------------|
| `/` | Landing | Landing page |
| `/login` | Login | Sign in |
| `/register` | Register | Sign up |
| `/dashboard` | Dashboard | Main panel (requires auth) |
| `/billing` | Billing | Subscription management (Stripe) |
| `/settings` | Settings | Account settings |
| `/team` | Team | Team management (Teams tier) |

## Configuration

API URL is configured via Vite environment variables:

```bash
# .env.local
VITE_API_URL=http://localhost:60402
```

## Project Structure

```
bytebrew-cloud-web/
├── src/
│   ├── api/            # API client (auth, billing, teams)
│   ├── pages/          # Pages
│   ├── components/     # Reusable components
│   ├── hooks/          # React hooks
│   ├── router.tsx      # Routing
│   └── main.tsx        # Entry point
├── public/             # Static assets
└── vite.config.ts
```
