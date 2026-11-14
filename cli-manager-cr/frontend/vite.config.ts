/// <reference types="vitest/config" />
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// 개발 서버는 브라우저에서 가짜 백엔드(src/ipc/fakeBackend.ts)로 동작한다.
export default defineConfig({
  plugins: [react()],
  // 정보 화면이 저장소 루트의 LICENSE 를 ?raw 로 읽으므로 상위 폴더도 허용한다 (D73).
  server: { port: 5173, strictPort: true, fs: { allow: ['..'] } },
  build: { outDir: 'dist', emptyOutDir: true, sourcemap: false },
  test: {
    environment: 'jsdom',
    include: ['src/**/*.test.{ts,tsx}'],
    globals: false,
    setupFiles: ['src/test/setup.ts'],
  },
});
