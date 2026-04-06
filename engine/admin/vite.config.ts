import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  base: '/admin/',
  server: {
    port: 3010,
    proxy: {
      '/api': {
        target: 'http://localhost:8443',
        changeOrigin: true,
      },
    },
  },
});
