import { test, expect } from '@playwright/test'
import { login } from './helpers/auth'

const runId = Date.now()
let uid = 0
function unique(name: string): string {
  uid++
  return `${name} ${runId}-${uid}`
}

test.describe('Habits', () => {
  test.beforeEach(async ({ page }) => {
    await login(page)
  })

  test('create a daily habit', async ({ page }) => {
    const name = unique('Read 20 pages')
    await page.goto('/habits')

    // Click Add Habit
    await page.getByRole('button', { name: 'Add Habit' }).click()

    // Fill name
    await page.getByLabel('Name').fill(name)

    // Color picker — click the third color swatch
    const colorButtons = page.locator('button[class*="rounded-full"]')
    await colorButtons.nth(2).click()

    // Frequency defaults to Daily — leave it
    // Submit
    await page.getByRole('button', { name: 'Create Habit' }).click()

    // Habit should appear in list
    await expect(page.getByText(name)).toBeVisible()
  })

  test('create a weekly habit with target', async ({ page }) => {
    const name = unique('Gym workout')
    await page.goto('/habits')

    await page.getByRole('button', { name: 'Add Habit' }).click()
    await page.getByLabel('Name').fill(name)

    // Switch to weekly frequency
    await page.getByLabel('Frequency').click()
    await page.getByRole('option', { name: 'Weekly' }).click()

    // Set target per week
    await page.getByLabel('Times/week').fill('3')

    await page.getByRole('button', { name: 'Create Habit' }).click()

    await expect(page.getByText(name)).toBeVisible()
  })

  test('toggle habit completion', async ({ page }) => {
    const name = unique('Meditate')
    await page.goto('/habits')

    // First create a habit
    await page.getByRole('button', { name: 'Add Habit' }).click()
    await page.getByLabel('Name').fill(name)
    await page.getByRole('button', { name: 'Create Habit' }).click()
    await expect(page.getByText(name)).toBeVisible()

    // Toggle it — the card's toggle button carries a stable accessible name.
    const toggleButton = page.getByRole('button', { name: `Toggle ${name}` })
    await toggleButton.click()

    // Button should report the completed state (optimistic update flips
    // aria-pressed immediately).
    await expect(toggleButton).toHaveAttribute('aria-pressed', 'true')
  })
})
