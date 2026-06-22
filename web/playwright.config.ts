import { defineConfig } from '@playwright/test'

export default defineConfig({
  testDir: './e2e',
  fullyParallel: false,
  workers: 1,
  reporter: 'list',
  use: {
    baseURL: process.env.WEB_BASE_URL ?? 'http://127.0.0.1:5173',
    headless: true,
    viewport: { width: 1280, height: 900 },
  },
})
