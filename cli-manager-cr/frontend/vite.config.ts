/// <reference types="vitest/config" />
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// 개발 서버는 브라우저에서 가짜 백엔드(src/ipc/fakeBackend.ts)로 동작한다.
export default defineConfig({
  plugins: [react()],
  server: { port: 5173, strictPort: true },
  build: { outDir: 'dist', emptyOutDir: true, sourcemap: false },
  test: {
    environment: 'jsdom',
    include: ['src/**/*.test.{ts,tsx}'],
    globals: false,
    setupFiles: ['src/test/setup.ts'],
  },
});
