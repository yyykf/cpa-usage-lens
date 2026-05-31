import path from 'node:path'
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// 开发时把 /api 代理到本地 backend；生产由 nginx 反代。
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    port: 5173,
    proxy: {
      '/api': { target: 'http://localhost:8080', changeOrigin: true },
    },
  },
})
