import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  server: {
    proxy: {
      // Override when the backend is not on the default port, e.g.
      // API_PROXY=http://localhost:8090 npm run dev
      '/api': process.env.API_PROXY || 'http://localhost:8080'
    }
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true
  }
})
