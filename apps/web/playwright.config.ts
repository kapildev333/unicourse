import { defineConfig } from '@playwright/test'
import { baseURL } from './e2e/env'

export default defineConfig({
  testDir: 'e2e',
  globalSetup: './e2e/global-setup.ts',
  globalTeardown: './e2e/global-teardown.ts',
  timeout: 120_000,
  expect: { timeout: 10_000 },
  retries: process.env.CI ? 1 : 0,
  workers: 1,
  reporter: [['list'], ['html', { open: 'never', outputFolder: 'playwright-report' }]],
  outputDir: 'test-results',
  use: { baseURL, trace: 'retain-on-failure' },
  projects: [
    { name: 'chromium', use: { browserName: 'chromium' } },
    { name: 'screens', testMatch: 'screens.ts', use: { browserName: 'chromium', actionTimeout: 15_000 } },
  ],
})
