import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        timeout: 120000, // Connection timeout: 120 seconds
        proxyTimeout: 120000, // Proxy timeout for the entire request: 120 seconds
        configure: (proxy) => {
          // Increase socket timeout for long-running requests
          proxy.on('proxyReq', (proxyReq, req, res) => {
            req.setTimeout(120000);
            res.setTimeout(120000);
          });
          // Log proxy errors for debugging
          proxy.on('error', (err, req, res) => {
            console.error('Proxy error:', err.message, 'for', req.url);
          });
          // Log when proxy request is made
          proxy.on('proxyReq', (proxyReq, req) => {
            console.log('Proxying:', req.method, req.url, '-> http://localhost:8080');
          });
        },
      },
    },
  },
  build: {
    outDir: 'dist',
    sourcemap: true,
  },
})
