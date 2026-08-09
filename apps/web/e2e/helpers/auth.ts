import { type Page } from '@playwright/test'
import { clearMailpitMessages, getMagicLinkUrl } from './mailpit'
import { TEST_EMAIL } from './env'

/**
 * Perform magic link login flow: clear mail, submit email, poll link, verify.
 * Returns after redirect to dashboard.
 */
export async function login(page: Page): Promise<void> {
  await clearMailpitMessages()

  await page.goto('/login')
  await page.getByPlaceholder('you@example.com').fill(TEST_EMAIL)
  await page.getByRole('button', { name: 'Send magic link' }).click()
  await page.getByText('Check your email').waitFor()

  const verifyUrl = await getMagicLinkUrl(TEST_EMAIL)
  await page.goto(verifyUrl)
  await page.waitForURL('**/')
}
