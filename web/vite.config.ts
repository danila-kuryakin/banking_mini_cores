/// <reference types="vitest" />
import { defineConfig, loadEnv } from 'vite';
import react from '@vitejs/plugin-react';

// Гейтвей без CORS, поэтому в dev фронт ходит на свой origin, а Vite проксирует /api.
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '');
  const target = env.VITE_API_TARGET || 'http://localhost:8080';

  return {
    plugins: [react()],
    build: {
      rollupOptions: {
        output: {
          // Библиотеки отдельным чанком: меняются реже кода приложения и дольше живут в кеше.
          manualChunks: {
            react: ['react', 'react-dom', 'react-router-dom'],
            vendor: ['@tanstack/react-query', 'react-hook-form', 'zod', '@hookform/resolvers', 'lucide-react'],
          },
        },
      },
    },
    server: {
      port: 5173,
      proxy: { '/api': { target, changeOrigin: true } },
    },
    preview: {
      port: 4173,
      proxy: { '/api': { target, changeOrigin: true } },
    },
    test: {
      environment: 'jsdom',
      globals: true,
      setupFiles: ['./src/test-setup.ts'],
      include: ['src/**/*.test.{ts,tsx}'],
    },
  };
});
