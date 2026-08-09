import { test, expect } from '@playwright/test'
import { clearMailpitMessages, getMagicLinkUrl } from './helpers/mailpit'
import { TEST_EMAIL } from './helpers/env'

test.describe('Auth flow', () => {
  test.beforeEach(async () => {
    await clearMailpitMessages()
  })

  test('login → receive magic link → verify → redirect to dashboard', async ({
    page,
    context,
  }) => {
    // Start at login page
    await page.goto('/login')
    await expect(page.getByText('Sign in to my')).toBeVisible()

    // Submit email
    await page.getByPlaceholder('you@example.com').fill(TEST_EMAIL)
    await page.getByRole('button', { name: 'Send magic link' }).click()

    // Wait for "Check your email" screen
    await expect(page.getByText('Check your email')).toBeVisible()

    // Poll Mailpit for the magic link URL
    const verifyUrl = await getMagicLinkUrl(TEST_EMAIL)
    expect(verifyUrl).toContain('/auth/verify?token=')

    // Navigate to verify URL
    await page.goto(verifyUrl)

    // Should redirect to dashboard after verification
    await page.waitForURL('**/')
    await expect(page.getByText('my')).toBeVisible()

    // Session cookie should be set
    const cookies = await context.cookies()
    const sessionCookie = cookies.find((c) => c.name === 'my_session')
    expect(sessionCookie).toBeDefined()
    expect(sessionCookie!.value).toBeTruthy()
  })

  test('invalid token shows error', async ({ page }) => {
    await page.goto('/auth/verify?token=invalid-token-12345')

    // The route must settle into the error card (regression: it used to
    // blank-frame on the idle mutation state, and this assertion used to
    // accept "Verifying..." as a pass — a tautology).
    await expect(page.getByText('Link expired or invalid')).toBeVisible({
      timeout: 15000,
    })

    // The API itself returns 401 for invalid tokens.
    const resp = await page.request.post('/api/v1/auth/verify', {
      data: { token: 'invalid-token-12345' },
    })
    expect(resp.status()).toBe(401)
    const body = await resp.json()
    expect(body.error).toBe('invalid token')
  })

  test('unauthenticated user redirected to login', async ({ page }) => {
    await page.goto('/')

    // Wait for redirect to login page (client-side router redirect)
    // Use text assertion instead of URL check because the navigation
    // may complete before waitForURL can register
    await expect(page.getByText('Sign in to my')).toBeVisible({
      timeout: 15000,
    })
  })
})
