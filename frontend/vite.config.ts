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
      // 5273 is the frontend host port (was 3000). This
      // port is also the OAuth redirect origin — keep it in sync with the API's
      // AUTH_REDIRECT_URI/CORS and the redirect registered in the Bungie app.
      port: 5273,
      allowedHosts: tunnelHost ? [tunnelHost] : [],
      proxy: {
        '/api': {
          target: 'http://localhost:8081',
          changeOrigin: true,
        },
      },
    },
    preview: {
      port: 5273,
    },
    build: {
      outDir: 'dist',
      sourcemap: false,
    },
    test: {
      globals: true,
      environment: 'jsdom',
      setupFiles: ['./src/__tests__/setup.ts'],
      coverage: {
        provider: 'v8',
        // Count every source file, not just files imported by tests — otherwise
        // the number is flattered by whatever happens to be untested.
        include: ['src/**/*.{ts,tsx}'],
        exclude: [
          'src/__tests__/**',
          'src/vite-env.d.ts',
          'src/main.tsx',
          'src/index.tsx',
          'src/types/**', // type-only modules have no executable statements
        ],
        // Floor for the current suite — ratchet upward as tests grow. Branch
        // coverage sits at ~74% and lines at ~83%; the gates leave headroom so
        // an incidental change doesn't redden CI while still catching regressions.
        thresholds: {
          lines: 70,
          branches: 65,
        },
      },
    },
  }
})
