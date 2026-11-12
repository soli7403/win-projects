import { resolve } from 'path'
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  base: './',
  resolve: { alias: { '@renderer': resolve(__dirname, 'src'), '@shared': resolve(__dirname, 'shared') } },
  plugins: [react()],
  build: { outDir: 'dist', emptyOutDir: true }
})
