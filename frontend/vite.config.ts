import { defineConfig } from 'vitest/config'
import { loadEnv } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig(({ mode }) => {
  // Load env files (e.g. frontend/.env.local). The empty prefix loads all vars,
  // not just VITE_*, so we can read NGROK_HOST below.
  const env = loadEnv(mode, process.cwd(), '')

  // Optional dev tunnel host (ngrok, Cloudflare Tunnel, etc.). Set NGROK_HOST in
  // frontend/.env.local — kept out of source control so the URL isn't committed.
  const tunnelHost = env.NGROK_HOST

  return {
    plugins: [react()],
    server: {
      port: 3000,
      allowedHosts: tunnelHost ? [tunnelHost] : [],
      proxy: {
        '/api': {
          target: 'http://localhost:8081',
          changeOrigin: true,
        },
      },
    },
    preview: {
      port: 3000,
    },
    build: {
      outDir: 'dist',
      sourcemap: false,
    },
    test: {
      globals: true,
      environment: 'jsdom',
      setupFiles: ['./src/__tests__/setup.ts'],
    },
  }
})
