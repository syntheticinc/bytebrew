import { defineConfig, loadEnv } from 'vite';
import react from '@vitejs/plugin-react';

// Wave 1+7 auth env vars (exposed to client via `import.meta.env`):
//   VITE_AUTH_MODE      'local' (default) | 'external'
//                       'local'     — admin hits POST /api/v1/auth/local-session
//                                     on boot to mint a JWT for the single
//                                     local admin user. Self-hosted default.
//                       'external'  — admin expects a `#at=...&rt=...` hash
//                                     fragment; if missing, it redirects to
//                                     VITE_LANDING_URL + '/login?return_to=…'.
//
//   VITE_LANDING_URL    Required when VITE_AUTH_MODE=external. Absolute URL
//                       of the external identity landing page. No default —
//                       a missing value in external mode throws at runtime.
//
// See src/hooks/useAuth.ts for the consumer.
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), 'VITE_');
  const authMode = env.VITE_AUTH_MODE ?? 'local';

  if (authMode === 'external' && !env.VITE_LANDING_URL) {
    // Fail the build early rather than ship a bundle that will redirect
    // to `undefined/login` at runtime.
    throw new Error(
      'VITE_AUTH_MODE=external requires VITE_LANDING_URL to be set (build-time env var)',
    );
  }

  return {
    plugins: [react()],
    base: '/admin/',
    define: {
      // Make the default explicit in the bundle so the SPA doesn't have
      // to branch on `undefined`.
      'import.meta.env.VITE_AUTH_MODE': JSON.stringify(authMode),
    },
    server: {
      port: 3010,
      proxy: {
        '/api': {
          target: 'http://localhost:8443',
          changeOrigin: true,
        },
      },
    },
  };
});
