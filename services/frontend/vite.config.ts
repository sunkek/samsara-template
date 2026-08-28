// defineConfig comes from vitest/config, not vite: it is the same config plus
// the `test` block below.
import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
// feat:if backend
import { resolveProxyTarget } from './config/devProxy'

const proxyTarget = resolveProxyTarget(import.meta.dirname)
// feat:end

export default defineConfig({
  plugins: [react()],
// feat:if backend
  server: {
    proxy: {
      '/api': { target: proxyTarget, changeOrigin: true },
    },
  },
  define: {
    'import.meta.env.VITE_API_BASE': JSON.stringify('/api/v1'),
  },
// feat:end
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test/setup.ts'],
// feat:if backend
    include: ['src/**/*.test.{ts,tsx}', 'config/**/*.test.ts'],
// feat:else
//~     include: ['src/**/*.test.{ts,tsx}'],
// feat:end
  },
})
