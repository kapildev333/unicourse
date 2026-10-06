/// <reference types="vitest/config" />
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { tanstackRouter } from '@tanstack/router-plugin/vite'

export default defineConfig({
  plugins: [tanstackRouter({ target: 'react', autoCodeSplitting: false }), react(), tailwindcss()],
  server: { proxy: { '/api': 'http://localhost:8080' } },
  test: { environment: 'jsdom', setupFiles: ['./src/test-setup.ts'] },
})
