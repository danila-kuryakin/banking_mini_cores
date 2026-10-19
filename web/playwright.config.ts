import { defineConfig } from '@playwright/test';

// Сквозной сценарий идёт против поднятого стенда: dev-сервер Vite или контейнер web.
export default defineConfig({
  testDir: './e2e',
  timeout: 180_000,
  expect: { timeout: 20_000 },
  use: {
    baseURL: process.env.E2E_BASE_URL || 'http://localhost:5173',
    trace: 'retain-on-failure',
  },
});
