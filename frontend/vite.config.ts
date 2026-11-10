import react from '@vitejs/plugin-react'
import { defineConfig } from 'vitest/config'

// The Go backend during development (started with docker compose). Proxying
// keeps the browser on one origin, exactly like nginx does in the Docker image,
// so the backend needs no CORS.
const backendURL = 'http://localhost:8080'

export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/api': backendURL,
      '/swagger': backendURL,
    },
  },
  test: {
    environment: 'jsdom',
    setupFiles: ['./src/test/setup.ts'],
    coverage: {
      provider: 'v8',
      include: ['src/**/*.{ts,tsx}'],
      exclude: ['src/**/*.test.{ts,tsx}', 'src/test/**', 'src/main.tsx', 'src/api/schema.gen.ts'],
      reporter: ['text', 'html'],
    },
  },
})
