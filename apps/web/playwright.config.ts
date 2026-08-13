import { defineConfig } from '@playwright/test'

export default defineConfig({
  testDir: './e2e',
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 1,
  workers: 1,
  timeout: 60000,
  // The html reporter spawns a server when opened; the github reporter is the
  // CI-native one. html is still produced for artifact upload.
  reporter: process.env.CI
    ? [['github'], ['html', { outputFolder: 'playwright-report', open: 'never' }]]
    : [['html', { outputFolder: 'playwright-report' }], ['list']],
  use: {
    baseURL: process.env.E2E_BASE_URL ?? 'http://localhost:5173',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
  },
  // Binary mode: the production artifact serves the SPA at :8080 (embedded
  // assets + SPA fallback), so CI needs no Vite and no proxy. Locally the
  // suite reuses an already-running binary; CI always starts fresh.
  webServer: {
    command: '../../bin/my',
    url: 'http://localhost:8080/api/v1/health',
    reuseExistingServer: !process.env.CI,
    timeout: 120000,
  },
  projects: [{ name: 'chromium', use: { browserName: 'chromium' } }],
})
