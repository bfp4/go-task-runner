import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// Proxy the Go backend routes so the browser talks to the same origin as the
// dev server. This avoids needing CORS on the Go side during development.
const backend = 'http://localhost:8080'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 3000,
    proxy: {
      '/all-posts': backend,
      '/posts': backend,
      '/create-post': backend,
    },
  },
})
